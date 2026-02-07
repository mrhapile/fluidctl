package k8s

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Client is a type alias for the controller-runtime client interface
type Client client.Client

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// Note: Fluid CRDs will be handled via Unstructured, so technically we don't strictly need to add them to scheme
	// if we only use Unstructured for them.
}

// NewClient returns a new Kubernetes client using standard config loading rules.
// It tries in-cluster config first, then KUBECONFIG, then default home dir.
func NewClient() (Client, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	opts := client.Options{
		Scheme: scheme,
	}

	c, err := client.New(config, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return c, nil
}
