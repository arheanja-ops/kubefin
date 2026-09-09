package optimize

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer"
	cotypes "github.com/aws/aws-sdk-go-v2/service/computeoptimizer/types"

	"github.com/arheanja-ops/kubefin/internal/model"
)

type fakeCO struct {
	ec2 *computeoptimizer.GetEC2InstanceRecommendationsOutput
	ebs *computeoptimizer.GetEBSVolumeRecommendationsOutput
}

func (f *fakeCO) GetEC2InstanceRecommendations(_ context.Context, _ *computeoptimizer.GetEC2InstanceRecommendationsInput, _ ...func(*computeoptimizer.Options)) (*computeoptimizer.GetEC2InstanceRecommendationsOutput, error) {
	return f.ec2, nil
}

func (f *fakeCO) GetEBSVolumeRecommendations(_ context.Context, _ *computeoptimizer.GetEBSVolumeRecommendationsInput, _ ...func(*computeoptimizer.Options)) (*computeoptimizer.GetEBSVolumeRecommendationsOutput, error) {
	return f.ebs, nil
}

func TestComputeOptimizer_SkipsOptimizedIncludesSaving(t *testing.T) {
	api := &fakeCO{
		ec2: &computeoptimizer.GetEC2InstanceRecommendationsOutput{
			InstanceRecommendations: []cotypes.InstanceRecommendation{
				{
					InstanceName: aws.String("web-1"),
					InstanceArn:  aws.String("arn:aws:ec2:...:instance/i-1"),
					Finding:      cotypes.FindingOverProvisioned,
					RecommendationOptions: []cotypes.InstanceRecommendationOption{{
						InstanceType: aws.String("t3.small"),
						Rank:         1,
						SavingsOpportunity: &cotypes.SavingsOpportunity{
							EstimatedMonthlySavings: &cotypes.EstimatedMonthlySavings{
								Currency: cotypes.CurrencyUsd,
								Value:    12.34,
							},
						},
					}},
				},
				{
					InstanceName: aws.String("db-1"),
					InstanceArn:  aws.String("arn:aws:ec2:...:instance/i-2"),
					Finding:      cotypes.FindingOptimized, // debe omitirse
				},
			},
		},
		ebs: &computeoptimizer.GetEBSVolumeRecommendationsOutput{},
	}

	findings, err := ComputeOptimizer(context.Background(), api)
	if err != nil {
		t.Fatalf("ComputeOptimizer error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("esperaba 1 hallazgo (el Optimized se omite), obtuve %d", len(findings))
	}
	f := findings[0]
	if f.EstimatedSaving == nil || f.EstimatedSaving.Amount != 12.34 {
		t.Errorf("esperaba ahorro 12.34, obtuve %+v", f.EstimatedSaving)
	}
	if f.EstimatedSaving.Currency != "USD" {
		t.Errorf("divisa esperada USD, obtuve %q", f.EstimatedSaving.Currency)
	}
}

func TestCorrelatePodRightsizing(t *testing.T) {
	nss := []model.NamespaceUsage{
		{
			Namespace: "over",
			Requests:  model.Resource{CPUMilli: 900},
			Usage:     &model.Resource{CPUMilli: 100},
		},
		{
			Namespace: "fine",
			Requests:  model.Resource{CPUMilli: 200},
			Usage:     &model.Resource{CPUMilli: 150},
		},
		{
			Namespace: "nometrics",
			Requests:  model.Resource{CPUMilli: 500},
			Usage:     nil,
		},
	}

	findings := CorrelatePodRightsizing(nss)
	if len(findings) != 1 {
		t.Fatalf("esperaba 1 hallazgo (solo 'over'), obtuve %d", len(findings))
	}
	if findings[0].Severity != model.SeverityWarning {
		t.Errorf("severidad esperada warning, obtuve %s", findings[0].Severity)
	}
}
