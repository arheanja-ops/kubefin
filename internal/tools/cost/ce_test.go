package cost

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

func TestEnsureCEOptIn_RejectsWithoutFlag(t *testing.T) {
	err := EnsureCEOptIn(false, nil)
	if !errors.Is(err, ErrCEAPINotOptedIn) {
		t.Fatalf("sin --use-ce-api debería rechazar con ErrCEAPINotOptedIn, obtuve %v", err)
	}
}

func TestEnsureCEOptIn_WarnsWithFlag(t *testing.T) {
	var warned string
	err := EnsureCEOptIn(true, func(m string) { warned = m })
	if err != nil {
		t.Fatalf("con --use-ce-api no debería fallar, obtuve %v", err)
	}
	if warned != CEWarning {
		t.Errorf("esperaba el aviso de costo, obtuve %q", warned)
	}
}

type fakeCE struct {
	out *costexplorer.GetCostAndUsageOutput
}

func (f *fakeCE) GetCostAndUsage(_ context.Context, _ *costexplorer.GetCostAndUsageInput, _ ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
	return f.out, nil
}

func TestByServiceCE_ParsesGroups(t *testing.T) {
	api := &fakeCE{out: &costexplorer.GetCostAndUsageOutput{
		ResultsByTime: []cetypes.ResultByTime{{
			Groups: []cetypes.Group{
				{
					Keys: []string{"Amazon Elastic Compute Cloud"},
					Metrics: map[string]cetypes.MetricValue{
						"UnblendedCost": {Amount: aws.String("42.00"), Unit: aws.String("USD")},
					},
				},
			},
		}},
	}}

	rows, err := ByServiceCE(context.Background(), api, 30)
	if err != nil {
		t.Fatalf("ByServiceCE error: %v", err)
	}
	if len(rows) != 1 || rows[0].Amount.Amount != 42.0 {
		t.Fatalf("esperaba 42.00 USD, obtuve %+v", rows)
	}
	if rows[0].Amount.Currency != "USD" {
		t.Errorf("divisa esperada USD, obtuve %q", rows[0].Amount.Currency)
	}
}
