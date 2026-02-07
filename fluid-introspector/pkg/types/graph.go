package types

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceGraph represents the hierarchical structure of a Fluid Dataset and its related resources.
type ResourceGraph struct {
	Dataset     *DatasetInfo     `json:"dataset"`
	Runtime     *RuntimeInfo     `json:"runtime,omitempty"`
	Infrastructure *InfrastructureInfo `json:"infrastructure,omitempty"`
}

// DatasetInfo encapsulates details about the Dataset CR.
type DatasetInfo struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Status    string            `json:"status"`
	Phase     string            `json:"phase"`
	Reason    string            `json:"reason,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	Object    metav1.Object     `json:"-"` // Raw object for internal use
}

// RuntimeInfo encapsulates details about the Runtime CR (Alluxio, Jindo, JuiceFS, etc.).
type RuntimeInfo struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"` // e.g., AlluxioRuntime, JindoRuntime
	Phase     string            `json:"phase"`
	Master    *ComponentInfo    `json:"master,omitempty"`
	Worker    *ComponentInfo    `json:"worker,omitempty"`
	Fuse      *ComponentInfo    `json:"fuse,omitempty"`
	Configs   []ConfigInfo      `json:"configs,omitempty"`
	Object    metav1.Object     `json:"-"`
}

// ComponentInfo represents a specific runtime component (Master, Worker, Fuse).
type ComponentInfo struct {
	Name        string             `json:"name"`
	Replicas    int32              `json:"replicas"`
	Ready       int32              `json:"ready"`
	State       string             `json:"state"` // e.g., "PartialReady", "Ready"
	Pods        []PodInfo          `json:"pods,omitempty"`
	DaemonSet   *appsv1.DaemonSet  `json:"-"`
	StatefulSet *appsv1.StatefulSet `json:"-"`
}

// PodInfo represents a single pod within a component.
type PodInfo struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"` // e.g., Running, Pending
	Node         string            `json:"node,omitempty"`
	Restarts     int32             `json:"restarts"`
	Age          string            `json:"age"`
	LastState    *corev1.ContainerState `json:"lastState,omitempty"`
	Object       *corev1.Pod       `json:"-"`
}

// InfrastructureInfo groups underlying K8s storage resources.
type InfrastructureInfo struct {
	PVC *PVCInfo `json:"pvc,omitempty"`
	PV  *PVInfo  `json:"pv,omitempty"`
}

type PVCInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"` // e.g., Bound
	Object *corev1.PersistentVolumeClaim `json:"-"`
}

type PVInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"` // e.g., Bound
	Object *corev1.PersistentVolume `json:"-"`
}

type ConfigInfo struct {
	Name string `json:"name"`
	Type string `json:"type"` // ConfigMap or Secret
}
