// Package k8sx construye los clientes de Kubernetes (kubeconfig, clientset, metrics).
package k8sx

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Clients agrupa los clientes de Kubernetes que consumen las tools. Metrics es
// nil cuando no se pudo construir el cliente de métricas; las tools deben
// tolerarlo (R1.4).
type Clients struct {
	// Kube es el clientset tipado contra la API de Kubernetes.
	Kube kubernetes.Interface
	// Metrics accede a metrics.k8s.io (equivalente a `kubectl top`). Puede ser nil.
	Metrics Metrics
	// Dynamic accede a recursos arbitrarios (CRDs como los de Karpenter). Puede ser nil.
	Dynamic dynamic.Interface
	// Context es el nombre del contexto de kubeconfig en uso.
	Context string
}

// Load construye los clientes a partir del kubeconfig y del contexto indicado.
// Si context es "", usa el contexto actual del kubeconfig. Devuelve un error
// claro cuando no hay un contexto válido (R1.2).
//
// kubeconfigPath vacío usa la resolución estándar (KUBECONFIG o ~/.kube/config).
func Load(kubeconfigPath, context string) (*Clients, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}

	overrides := &clientcmd.ConfigOverrides{}
	if context != "" {
		overrides.CurrentContext = context
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	rawConfig, err := clientConfig.RawConfig()
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el kubeconfig: %w", err)
	}

	resolvedContext := context
	if resolvedContext == "" {
		resolvedContext = rawConfig.CurrentContext
	}
	if resolvedContext == "" {
		return nil, fmt.Errorf("no hay un contexto de kubectl activo: configura uno con `kubectl config use-context <nombre>` o pasa --context")
	}
	if _, ok := rawConfig.Contexts[resolvedContext]; !ok {
		return nil, fmt.Errorf("el contexto %q no existe en el kubeconfig", resolvedContext)
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("no se pudo construir la configuración del cliente para el contexto %q: %w", resolvedContext, err)
	}

	return newClients(restConfig, resolvedContext)
}

// newClients construye los clientes tipados a partir de un *rest.Config ya resuelto.
func newClients(restConfig *rest.Config, context string) (*Clients, error) {
	kube, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("no se pudo construir el clientset de Kubernetes: %w", err)
	}

	// El cliente de métricas es opcional: si falla su construcción, las tools
	// continúan sin datos de uso (R1.4). No es un error fatal.
	var metrics Metrics
	if mc, mErr := metricsv.NewForConfig(restConfig); mErr == nil {
		metrics = realMetrics{client: mc}
	}

	// El cliente dinámico es opcional; se usa para CRDs (Karpenter).
	var dyn dynamic.Interface
	if dc, dErr := dynamic.NewForConfig(restConfig); dErr == nil {
		dyn = dc
	}

	return &Clients{
		Kube:    kube,
		Metrics: metrics,
		Dynamic: dyn,
		Context: context,
	}, nil
}
