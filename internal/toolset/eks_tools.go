package toolset

import (
	"context"
	"encoding/json"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/tools/eks"
)

// eksClients construye los clientes de Kubernetes a partir de las deps.
func eksClients(deps Deps) (*k8sx.Clients, error) {
	return k8sx.Load(deps.Kubeconfig, deps.KubeContext)
}

// registerEKS añade las tools de diagnóstico EKS al registro.
func registerEKS(r *Registry) {
	r.Register(Tool{
		Name:        "eks_report",
		Description: "Genera un reporte de diagnóstico completo del clúster EKS/Kubernetes actual: nodos, namespaces, componentes core, reinicios y hallazgos con recomendaciones.",
		Parameters:  emptyObjectSchema(),
		Handler: func(ctx context.Context, deps Deps, _ json.RawMessage) (any, error) {
			c, err := eksClients(deps)
			if err != nil {
				return nil, err
			}
			return eks.BuildReport(ctx, c)
		},
	})

	r.Register(Tool{
		Name:        "eks_nodes",
		Description: "Lista los nodos del clúster con su estado, capacidad y uso de recursos.",
		Parameters:  emptyObjectSchema(),
		Handler: func(ctx context.Context, deps Deps, _ json.RawMessage) (any, error) {
			c, err := eksClients(deps)
			if err != nil {
				return nil, err
			}
			return eks.Nodes(ctx, c)
		},
	})

	r.Register(Tool{
		Name:        "eks_namespaces",
		Description: "Agrega el uso de recursos (requests, limits y uso real) por namespace.",
		Parameters:  emptyObjectSchema(),
		Handler: func(ctx context.Context, deps Deps, _ json.RawMessage) (any, error) {
			c, err := eksClients(deps)
			if err != nil {
				return nil, err
			}
			return eks.Namespaces(ctx, c)
		},
	})
}

// emptyObjectSchema es el JSON Schema de una tool sin parámetros.
func emptyObjectSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}
