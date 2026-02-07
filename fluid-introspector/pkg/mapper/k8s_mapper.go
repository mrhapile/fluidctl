package mapper

import (
	"context"
	"fmt"
	"strings"

	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/k8s"
	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// K8sMapper implements real Kubernetes discovery for Fluid resources.
type K8sMapper struct {
	client client.Client
}

// NewK8sMapper creates a mapper that talks to the API server.
func NewK8sMapper(c k8s.Client) *K8sMapper {
	// Cast the custom interface back to controller-runtime client
	return &K8sMapper{client: c}
}

// MapDataset discovers the Dataset and all related resources in the cluster.
func (m *K8sMapper) MapDataset(ctx context.Context, name, namespace string) (*types.ResourceGraph, error) {
	graph := &types.ResourceGraph{}

	// 1. Discover Dataset
	datasetInfo, err := m.mapDatasetCR(ctx, name, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("dataset %s/%s not found", namespace, name)
		}
		return nil, fmt.Errorf("failed to get dataset: %w", err)
	}
	graph.Dataset = datasetInfo

	// 2. Discover Runtime (Alluxio, Jindo, JuiceFS, etc.)
	// We try common runtime kinds.
	runtimeInfo, err := m.discoverRuntime(ctx, name, namespace)
	if err != nil {
		// Log error? For now, if we fail to find *any* runtime, we leave it nil (which triggers RUNTIME_MISSING)
		// strictly speaking, an error from discoverRuntime means API failure, not just "not found"
		// But let's proceed to return partial graph if API reads fail?
		// No, let's bubble up API errors, but handle NotFound gracefully inside discoverRuntime.
		return nil, fmt.Errorf("failed to discover runtime: %w", err)
	}
	graph.Runtime = runtimeInfo

	// 3. Discover Infrastructure (PVC/PV)
	infraInfo, err := m.discoverInfrastructure(ctx, name, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to discover infrastructure: %w", err)
	}
	graph.Infrastructure = infraInfo

	return graph, nil
}

// mapDatasetCR fetches the Dataset CR via Unstructured.
func (m *K8sMapper) mapDatasetCR(ctx context.Context, name, namespace string) (*types.DatasetInfo, error) {
	u := &unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "data.fluid.io",
		Version: "v1alpha1",
		Kind:    "Dataset",
	})

	key := client.ObjectKey{Name: name, Namespace: namespace}
	if err := m.client.Get(ctx, key, u); err != nil {
		return nil, err
	}

	// Extract Status fields
	statusPhase, _, _ := unstructured.NestedString(u.Object, "status", "phase")
	// If phase is empty, it might be Pending or newly created
	if statusPhase == "" {
		statusPhase = "NotReady"
	}

	// Determine "Bound" status based on phase or condition?
	// Fluid Datasets are "Bound" when they are ready to use.
	// Commonly Phase == Bound is the check.
	status := "NotBound"
	if strings.EqualFold(statusPhase, "Bound") {
		status = "Bound"
	}

	return &types.DatasetInfo{
		Name:      u.GetName(),
		Namespace: u.GetNamespace(),
		Status:    status,
		Phase:     statusPhase,
		Labels:    u.GetLabels(),
		Object:    u, // Store raw object for debugging/extensions
	}, nil
}

// discoverRuntime attempts to find the matching Runtime CR.
func (m *K8sMapper) discoverRuntime(ctx context.Context, name, namespace string) (*types.RuntimeInfo, error) {
	// Priority list of runtimes to check
	kinds := []string{"AlluxioRuntime", "JindoRuntime", "JuiceFSRuntime", "ThinRuntime"}

	for _, kind := range kinds {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   "data.fluid.io",
			Version: "v1alpha1",
			Kind:    kind,
		})

		key := client.ObjectKey{Name: name, Namespace: namespace}
		if err := m.client.Get(ctx, key, u); err == nil {
			// Found it! Map it.
			return m.mapRuntime(ctx, u, kind)
		} else if !apierrors.IsNotFound(err) {
			// Real API error
			return nil, err
		}
	}

	// None found
	return nil, nil
}

func (m *K8sMapper) mapRuntime(ctx context.Context, u *unstructured.Unstructured, kind string) (*types.RuntimeInfo, error) {
	info := &types.RuntimeInfo{
		Name:   u.GetName(),
		Type:   kind,
		Phase:  getNestedString(u, "status", "phase"),
		Object: u,
	}

	// Inspect Workloads (StatefulSets/DaemonSets)
	// We rely on labels: fluid.io/dataset=<name> + role=maste/worker/fuse
	// Or names: <name>-master, <name>-worker, <name>-fuse. Names are standard in Fluid.

	// Master (StatefulSet)
	masterName := fmt.Sprintf("%s-master", u.GetName())
	masterSTS, err := m.getReadyStatefulSet(ctx, masterName, u.GetNamespace())
	if err == nil && masterSTS != nil {
		info.Master = &types.ComponentInfo{
			Name:        masterName,
			Replicas:    *masterSTS.Spec.Replicas,
			Ready:       masterSTS.Status.ReadyReplicas,
			State:       determineComponentState(masterSTS.Status.ReadyReplicas, *masterSTS.Spec.Replicas),
			StatefulSet: masterSTS,
		}
	}

	// Worker (StatefulSet or DaemonSet)
	// Try StatefulSet first
	workerName := fmt.Sprintf("%s-worker", u.GetName())
	workerSTS, err := m.getReadyStatefulSet(ctx, workerName, u.GetNamespace())
	if err == nil && workerSTS != nil {
		info.Worker = &types.ComponentInfo{
			Name:        workerName,
			Replicas:    *workerSTS.Spec.Replicas,
			Ready:       workerSTS.Status.ReadyReplicas,
			State:       determineComponentState(workerSTS.Status.ReadyReplicas, *workerSTS.Spec.Replicas),
			StatefulSet: workerSTS,
		}
	} else {
		// Try DaemonSet
		workerDS, err := m.getReadyDaemonSet(ctx, workerName, u.GetNamespace())
		if err == nil && workerDS != nil {
			info.Worker = &types.ComponentInfo{
				Name:      workerName,
				Replicas:  workerDS.Status.DesiredNumberScheduled,
				Ready:     workerDS.Status.NumberReady,
				State:     determineComponentState(workerDS.Status.NumberReady, workerDS.Status.DesiredNumberScheduled),
				DaemonSet: workerDS,
			}
		}
	}

	// Fuse (DaemonSet)
	fuseName := fmt.Sprintf("%s-fuse", u.GetName())
	fuseDS, err := m.getReadyDaemonSet(ctx, fuseName, u.GetNamespace())
	if err == nil && fuseDS != nil {
		info.Fuse = &types.ComponentInfo{
			Name:      fuseName,
			Replicas:  fuseDS.Status.DesiredNumberScheduled,
			Ready:     fuseDS.Status.NumberReady,
			State:     determineComponentState(fuseDS.Status.NumberReady, fuseDS.Status.DesiredNumberScheduled),
			DaemonSet: fuseDS,
		}
	}

	return info, nil
}

func (m *K8sMapper) getReadyStatefulSet(ctx context.Context, name, namespace string) (*appsv1.StatefulSet, error) {
	sts := &appsv1.StatefulSet{}
	key := client.ObjectKey{Name: name, Namespace: namespace}
	if err := m.client.Get(ctx, key, sts); err != nil {
		return nil, err
	}
	return sts, nil
}

func (m *K8sMapper) getReadyDaemonSet(ctx context.Context, name, namespace string) (*appsv1.DaemonSet, error) {
	ds := &appsv1.DaemonSet{}
	key := client.ObjectKey{Name: name, Namespace: namespace}
	if err := m.client.Get(ctx, key, ds); err != nil {
		return nil, err
	}
	return ds, nil
}

// discoverInfrastructure maps PVC/PV.
func (m *K8sMapper) discoverInfrastructure(ctx context.Context, name, namespace string) (*types.InfrastructureInfo, error) {
	infra := &types.InfrastructureInfo{}

	// Fetch PVC
	pvc := &corev1.PersistentVolumeClaim{}
	key := client.ObjectKey{Name: name, Namespace: namespace}
	if err := m.client.Get(ctx, key, pvc); err == nil {
		infra.PVC = &types.PVCInfo{
			Name:   pvc.Name,
			Status: string(pvc.Status.Phase),
			Object: pvc,
		}

		// If bound, fetch PV
		if pvc.Spec.VolumeName != "" {
			pv := &corev1.PersistentVolume{}
			pvKey := client.ObjectKey{Name: pvc.Spec.VolumeName} // PV is cluster-scoped
			if err := m.client.Get(ctx, pvKey, pv); err == nil {
				infra.PV = &types.PVInfo{
					Name:   pv.Name,
					Status: string(pv.Status.Phase),
					Object: pv,
				}
			}
		}
	} else if !apierrors.IsNotFound(err) {
		return nil, err
	}

	return infra, nil
}

// Helpers

func getNestedString(u *unstructured.Unstructured, fields ...string) string {
	val, _, _ := unstructured.NestedString(u.Object, fields...)
	return val
}

func determineComponentState(ready, desired int32) string {
	if desired == 0 {
		return "ComponentsScaledDown"
	}
	if ready == desired {
		return "Ready"
	}
	if ready > 0 {
		return "PartialReady"
	}
	return "NotReady"
}
