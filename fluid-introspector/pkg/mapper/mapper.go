package mapper

import (
	"context"

	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/k8s"
	"github.com/fluid-cloudnative/fluid-introspector/fluid-introspector/pkg/types"
)

// ResourceMapper orchestrates the discovery and correlation of Fluid resources.
type ResourceMapper struct {
	client k8s.Client
}

// NewResourceMapper initializes a new mapper with a Kubernetes client.
func NewResourceMapper(client k8s.Client) *ResourceMapper {
	return &ResourceMapper{
		client: client,
	}
}

// MapDataset is the primary entry point. It fetches the Dataset CR and recursively
// discovers all related Runtime components and infrastructure.
func (m *ResourceMapper) MapDataset(ctx context.Context, name, namespace string) (*types.ResourceGraph, error) {
	graph := &types.ResourceGraph{}

	// Step 1: Discover Dataset
	dataset, err := m.discoverDataset(ctx, name, namespace)
	if err != nil {
		return nil, err
	}
	graph.Dataset = dataset

	// Step 2: Find bound Runtime
	// This logic will reside in runtime.go
	runtime, err := m.discoverRuntime(ctx, dataset)
	if err != nil {
		// Log error but proceed if possible? Or return partial graph?
		// For now, return error as runtime is critical.
		return nil, err
	}
	graph.Runtime = runtime

	// Step 3: Map Infrastructure (PVCs, PVs)
	// resources.go logic
	infra, err := m.discoverInfrastructure(ctx, dataset, runtime)
	if err != nil {
		return nil, err
	}
	graph.Infrastructure = infra

	return graph, nil
}
