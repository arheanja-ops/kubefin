package toolset

import (
	"context"
	"encoding/json"

	"github.com/arheanja-ops/kubefin/internal/awsx"
	"github.com/arheanja-ops/kubefin/internal/tools/cost"
	"github.com/arheanja-ops/kubefin/internal/tools/optimize"
)

// costArgs son los parámetros de la tool cost_by_service.
type costArgs struct {
	Bucket string `json:"bucket"`
	Prefix string `json:"prefix"`
	Days   int    `json:"days"`
}

// awsClients carga y verifica los clientes de AWS.
func awsClients(ctx context.Context, deps Deps) (*awsx.Clients, error) {
	c, err := awsx.Load(ctx, deps.Region)
	if err != nil {
		return nil, err
	}
	if _, err := c.VerifyIdentity(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// registerCost añade la tool de costo por servicio (CUR-first, CE API opt-in).
func registerCost(r *Registry) {
	r.Register(Tool{
		Name:        "cost_by_service",
		Description: "Costo de AWS agrupado por servicio en los últimos N días, leído del Cost & Usage Report (CUR) en S3 (gratis). Requiere bucket y prefix del CUR.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"bucket": map[string]any{"type": "string", "description": "bucket S3 del CUR"},
				"prefix": map[string]any{"type": "string", "description": "prefijo del export del CUR"},
				"days":   map[string]any{"type": "integer", "description": "ventana en días (por defecto 30)"},
			},
			"required": []string{"bucket"},
		},
		Handler: func(ctx context.Context, deps Deps, raw json.RawMessage) (any, error) {
			var a costArgs
			if len(raw) > 0 {
				if err := json.Unmarshal(raw, &a); err != nil {
					return nil, err
				}
			}
			if a.Days <= 0 {
				a.Days = 30
			}
			if a.Prefix == "" {
				a.Prefix = "cur"
			}

			// El CUR es gratis; la CE API (opt-in) queda fuera de esta tool para
			// no incurrir en costo desde una superficie automática (R4.4).
			c, err := awsClients(ctx, deps)
			if err != nil {
				return nil, err
			}
			return cost.ByService(ctx, c.S3, cost.Source{Bucket: a.Bucket, Prefix: a.Prefix}, a.Days)
		},
	})
}

// registerOptimize añade la tool de rightsizing de Compute Optimizer.
func registerOptimize(r *Registry) {
	r.Register(Tool{
		Name:        "optimize_compute",
		Description: "Recomendaciones de rightsizing de Compute Optimizer para EC2/EBS, con ahorro mensual estimado.",
		Parameters:  emptyObjectSchema(),
		Handler: func(ctx context.Context, deps Deps, _ json.RawMessage) (any, error) {
			c, err := awsClients(ctx, deps)
			if err != nil {
				return nil, err
			}
			return optimize.ComputeOptimizer(ctx, c.ComputeOptimizer)
		},
	})
}
