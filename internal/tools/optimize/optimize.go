// Package optimize implementa las recomendaciones de rightsizing: Compute
// Optimizer (EC2/EBS) y la correlación con el uso real de pods de la Fase 1,
// con estimación de ahorro cuando hay datos de costo (R3).
package optimize

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/computeoptimizer"
	cotypes "github.com/aws/aws-sdk-go-v2/service/computeoptimizer/types"

	"github.com/arheanja-ops/kubefin/internal/awsx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

// overprovisionRatio: request/uso a partir del cual se recomienda reducir requests.
const overprovisionRatio = 3

// ComputeOptimizer obtiene recomendaciones de EC2 y EBS de Compute Optimizer
// (gratis) y las convierte en hallazgos con ahorro estimado cuando AWS lo provee
// (R3.1, R3.3, R3.4).
func ComputeOptimizer(ctx context.Context, api awsx.ComputeOptimizerAPI) ([]model.Finding, error) {
	var findings []model.Finding

	ec2, err := api.GetEC2InstanceRecommendations(ctx, &computeoptimizer.GetEC2InstanceRecommendationsInput{})
	if err != nil {
		return nil, fmt.Errorf("la consulta EC2 a Compute Optimizer falló: %w", err)
	}
	findings = append(findings, ec2Findings(ec2.InstanceRecommendations)...)

	ebs, err := api.GetEBSVolumeRecommendations(ctx, &computeoptimizer.GetEBSVolumeRecommendationsInput{})
	if err != nil {
		return nil, fmt.Errorf("la consulta EBS a Compute Optimizer falló: %w", err)
	}
	findings = append(findings, ebsFindings(ebs.VolumeRecommendations)...)

	return findings, nil
}

func ec2Findings(recs []cotypes.InstanceRecommendation) []model.Finding {
	var out []model.Finding
	for i := range recs {
		rec := &recs[i]
		if rec.Finding == cotypes.FindingOptimized {
			continue
		}
		arn := aws.ToString(rec.InstanceArn)
		best, saving := bestOption(rec.RecommendationOptions)
		f := model.Finding{
			Severity:       severityForFinding(rec.Finding),
			Title:          fmt.Sprintf("EC2 %s: %s", aws.ToString(rec.InstanceName), rec.Finding),
			Evidence:       fmt.Sprintf("Instancia %s clasificada como %s por Compute Optimizer.", arn, rec.Finding),
			Recommendation: fmt.Sprintf("Considera cambiar al tipo recomendado %s.", best),
		}
		if saving != nil {
			f.EstimatedSaving = saving
		}
		out = append(out, f)
	}
	return out
}

func ebsFindings(recs []cotypes.VolumeRecommendation) []model.Finding {
	var out []model.Finding
	for i := range recs {
		rec := &recs[i]
		if rec.Finding == cotypes.EBSFindingOptimized {
			continue
		}
		out = append(out, model.Finding{
			Severity:       model.SeverityWarning,
			Title:          fmt.Sprintf("EBS %s: %s", aws.ToString(rec.VolumeArn), rec.Finding),
			Evidence:       fmt.Sprintf("Volumen clasificado como %s por Compute Optimizer.", rec.Finding),
			Recommendation: "Revisa el tipo/tamaño del volumen según la recomendación de Compute Optimizer.",
		})
	}
	return out
}

// bestOption elige la opción con menor rank y extrae el ahorro mensual estimado.
func bestOption(opts []cotypes.InstanceRecommendationOption) (string, *model.Money) {
	if len(opts) == 0 {
		return "(sin opción)", nil
	}
	best := opts[0]
	for i := range opts {
		if opts[i].Rank < best.Rank {
			best = opts[i]
		}
	}
	instanceType := aws.ToString(best.InstanceType)
	if best.SavingsOpportunity != nil && best.SavingsOpportunity.EstimatedMonthlySavings != nil {
		s := best.SavingsOpportunity.EstimatedMonthlySavings
		return instanceType, &model.Money{
			Amount:   s.Value,
			Currency: string(s.Currency),
		}
	}
	return instanceType, nil
}

func severityForFinding(f cotypes.Finding) model.Severity {
	if f == cotypes.FindingUnderProvisioned {
		return model.SeverityCritical
	}
	return model.SeverityWarning
}

// CorrelatePodRightsizing correlaciona el uso real de recursos por namespace
// (Fase 1) con las solicitudes, emitiendo hallazgos de sobredimensionamiento a
// nivel de pods (R3.2). Es determinista y no requiere AWS.
func CorrelatePodRightsizing(namespaces []model.NamespaceUsage) []model.Finding {
	var out []model.Finding
	for _, ns := range namespaces {
		if ns.Usage == nil || ns.Requests.CPUMilli == 0 {
			continue
		}
		if ns.Usage.CPUMilli*int64(overprovisionRatio) < ns.Requests.CPUMilli {
			out = append(out, model.Finding{
				Severity: model.SeverityWarning,
				Title:    fmt.Sprintf("Rightsizing: namespace %s sobredimensionado en CPU", ns.Namespace),
				Evidence: fmt.Sprintf("CPU solicitada %dm vs. uso real %dm (>%dx).",
					ns.Requests.CPUMilli, ns.Usage.CPUMilli, overprovisionRatio),
				Recommendation: "Reduce los requests de CPU hacia el uso real para liberar capacidad de nodos y reducir costo.",
			})
		}
	}
	return out
}
