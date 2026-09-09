package cost

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/aws/smithy-go/ptr"

	"github.com/arheanja-ops/kubefin/internal/model"
)

// ExplorerAPI abstrae la operación de Cost Explorer usada, para mockearla.
type ExplorerAPI interface {
	GetCostAndUsage(ctx context.Context, in *costexplorer.GetCostAndUsageInput, optFns ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
}

// CECostPerRequestUSD es el costo por request paginado de la Cost Explorer API,
// según la documentación oficial de AWS.
const CECostPerRequestUSD = 0.01

// ErrCEAPINotOptedIn se devuelve cuando se intenta usar la CE API sin el opt-in
// explícito (R2.3, R6.2).
var ErrCEAPINotOptedIn = errors.New(
	"la Cost Explorer API tiene costo ($0.01/request) y está deshabilitada por defecto: " +
		"reintenta con --use-ce-api para autorizarla, o usa el CUR (gratis)")

// CEWarning es el aviso de costo que debe mostrarse antes de invocar la CE API (R2.2).
const CEWarning = "AVISO: la Cost Explorer API cobra $0.01 por request paginado. " +
	"Procediendo porque se pasó --use-ce-api."

// ByServiceCE consulta la Cost Explorer API agrupando por servicio en los
// últimos `days` días. Solo debe llamarse tras confirmar el opt-in con
// EnsureCEOptIn; de lo contrario incurre en costo.
func ByServiceCE(ctx context.Context, api ExplorerAPI, days int) ([]model.CostRow, error) {
	now := time.Now().UTC()
	start := windowStart(now, days)
	end := now

	out, err := api.GetCostAndUsage(ctx, &costexplorer.GetCostAndUsageInput{
		TimePeriod: &cetypes.DateInterval{
			Start: aws.String(start.Format("2006-01-02")),
			End:   aws.String(end.Format("2006-01-02")),
		},
		Granularity: cetypes.GranularityMonthly,
		Metrics:     []string{"UnblendedCost"},
		GroupBy: []cetypes.GroupDefinition{{
			Type: cetypes.GroupDefinitionTypeDimension,
			Key:  ptr.String("SERVICE"),
		}},
	})
	if err != nil {
		return nil, fmt.Errorf("la Cost Explorer API falló: %w", err)
	}

	period := model.Period{Start: start, End: end}
	agg := map[string]float64{}
	currency := "USD"
	for _, result := range out.ResultsByTime {
		for _, g := range result.Groups {
			dim := "(sin valor)"
			if len(g.Keys) > 0 {
				dim = g.Keys[0]
			}
			if m, ok := g.Metrics["UnblendedCost"]; ok {
				if cur := aws.ToString(m.Unit); cur != "" {
					currency = cur
				}
				var amt float64
				_, _ = fmt.Sscanf(aws.ToString(m.Amount), "%g", &amt)
				agg[dim] += amt
			}
		}
	}
	return toRows(agg, currency, period), nil
}

// EnsureCEOptIn aplica el guardarraíl de costo: rechaza si no hubo opt-in
// explícito (R2.3). Cuando useCEAPI es true, escribe el aviso de costo en warn.
func EnsureCEOptIn(useCEAPI bool, warn func(string)) error {
	if !useCEAPI {
		return ErrCEAPINotOptedIn
	}
	if warn != nil {
		warn(CEWarning)
	}
	return nil
}
