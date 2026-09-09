package eks

import (
	"context"
	"fmt"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

// overprovisionRatio es el múltiplo de request sobre uso a partir del cual se
// considera que un namespace está sobredimensionado.
const overprovisionRatio = 3

// restartThreshold es el número de reinicios a partir del cual se emite un hallazgo.
const restartThreshold = 5

// Recommendations deriva hallazgos accionables a partir del reporte recolectado.
// Es determinista y no depende de IA (principio de diseño).
func Recommendations(r *model.Report) []model.Finding {
	findings := []model.Finding{}

	findings = append(findings, nodeFindings(r.Nodes)...)
	findings = append(findings, namespaceFindings(r.Namespaces, r.MetricsAvailable)...)
	findings = append(findings, restartFindings(r.Restarts)...)
	findings = append(findings, componentFindings(r.Components)...)
	findings = append(findings, datadogFindings(r.Datadog)...)
	findings = append(findings, karpenterFindings(r.Karpenter)...)

	return findings
}

func nodeFindings(nodes []model.Node) []model.Finding {
	var out []model.Finding
	for _, n := range nodes {
		if !n.Ready {
			out = append(out, model.Finding{
				Severity:       model.SeverityCritical,
				Title:          fmt.Sprintf("Nodo %s no está Ready", n.Name),
				Evidence:       fmt.Sprintf("El nodo %s reporta condición Ready != True.", n.Name),
				Recommendation: "Revisa el estado del kubelet y los eventos del nodo.",
				Command:        fmt.Sprintf("kubectl describe node %s", n.Name),
			})
			continue
		}
		if !n.Schedulable {
			out = append(out, model.Finding{
				Severity:       model.SeverityWarning,
				Title:          fmt.Sprintf("Nodo %s está cordoned (no schedulable)", n.Name),
				Evidence:       "El nodo tiene spec.unschedulable=true.",
				Recommendation: "Si no es intencional, ejecuta uncordon.",
				Command:        fmt.Sprintf("kubectl uncordon %s", n.Name),
			})
		}
	}
	return out
}

func namespaceFindings(namespaces []model.NamespaceUsage, metricsAvailable bool) []model.Finding {
	if !metricsAvailable {
		return nil
	}
	var out []model.Finding
	for _, ns := range namespaces {
		if ns.Usage == nil || ns.Requests.CPUMilli == 0 {
			continue
		}
		if ns.Usage.CPUMilli*int64(overprovisionRatio) < ns.Requests.CPUMilli {
			out = append(out, model.Finding{
				Severity: model.SeverityWarning,
				Title:    fmt.Sprintf("Namespace %s sobredimensionado en CPU", ns.Namespace),
				Evidence: fmt.Sprintf("CPU solicitada %dm vs. uso real %dm (>%dx).",
					ns.Requests.CPUMilli, ns.Usage.CPUMilli, overprovisionRatio),
				Recommendation: "Ajusta los requests de CPU de las cargas del namespace para acercarlos al uso real.",
			})
		}
	}
	return out
}

func restartFindings(restarts []model.PodRestart) []model.Finding {
	var out []model.Finding
	for _, pr := range restarts {
		if pr.Restarts < restartThreshold {
			continue
		}
		reason := pr.Reason
		if reason == "" {
			reason = "desconocido"
		}
		out = append(out, model.Finding{
			Severity:       model.SeverityWarning,
			Title:          fmt.Sprintf("Pod %s/%s con %d reinicios", pr.Namespace, pr.Name, pr.Restarts),
			Evidence:       fmt.Sprintf("Último motivo de terminación: %s.", reason),
			Recommendation: "Revisa los logs y eventos del pod para diagnosticar la inestabilidad.",
			Command:        fmt.Sprintf("kubectl -n %s logs %s --previous", pr.Namespace, pr.Name),
		})
	}
	return out
}

func componentFindings(components []model.ComponentStatus) []model.Finding {
	var out []model.Finding
	for _, comp := range components {
		if comp.Healthy {
			continue
		}
		out = append(out, model.Finding{
			Severity:       model.SeverityCritical,
			Title:          fmt.Sprintf("Componente core %s/%s degradado", comp.Namespace, comp.Name),
			Evidence:       fmt.Sprintf("%s %s: %d/%d réplicas listas.", comp.Kind, comp.Name, comp.Ready, comp.Desired),
			Recommendation: "Revisa el estado del componente; afecta funciones core del clúster.",
			Command:        fmt.Sprintf("kubectl -n %s describe %s %s", comp.Namespace, comp.Kind, comp.Name),
		})
	}
	return out
}

func datadogFindings(dd *model.DatadogCoverage) []model.Finding {
	if dd == nil || !dd.Installed {
		return nil
	}
	if int(dd.CoveredNodes) < dd.TotalNodes {
		return []model.Finding{{
			Severity:       model.SeverityWarning,
			Title:          "Cobertura incompleta del Datadog Agent",
			Evidence:       fmt.Sprintf("El agente cubre %d de %d nodos.", dd.CoveredNodes, dd.TotalNodes),
			Recommendation: "Revisa por qué el DaemonSet de Datadog no corre en todos los nodos (taints/tolerations).",
		}}
	}
	return nil
}

func karpenterFindings(k *model.KarpenterStatus) []model.Finding {
	if k == nil || !k.Installed {
		return nil
	}
	var out []model.Finding
	if k.NodePools == 0 {
		out = append(out, model.Finding{
			Severity:       model.SeverityWarning,
			Title:          "Karpenter instalado sin NodePools",
			Evidence:       "No se encontró ningún NodePool de Karpenter.",
			Recommendation: "Define al menos un NodePool para que Karpenter aprovisione nodos.",
		})
	}
	for _, issue := range k.Issues {
		out = append(out, model.Finding{
			Severity:       model.SeverityWarning,
			Title:          "Problema en NodeClaim de Karpenter",
			Evidence:       issue,
			Recommendation: "Revisa el estado del NodeClaim y su EC2NodeClass asociada.",
		})
	}
	return out
}

// BuildReport ejecuta todas las tools de diagnóstico y ensambla un Report
// completo con sus hallazgos. Las secciones cuya recolección falle de forma no
// fatal se registran en Warnings sin abortar el resto (R1.4).
func BuildReport(ctx context.Context, c *k8sx.Clients) (*model.Report, error) {
	r := &model.Report{Context: c.Context}
	r.MetricsAvailable = metricsAvailable(ctx, c)
	if !r.MetricsAvailable {
		r.Warnings = append(r.Warnings, "Metrics Server no disponible: se omiten los datos de uso real.")
	}

	nodes, err := Nodes(ctx, c)
	if err != nil {
		return nil, err
	}
	r.Nodes = nodes

	namespaces, err := Namespaces(ctx, c)
	if err != nil {
		return nil, err
	}
	r.Namespaces = namespaces

	if top, tErr := TopPods(ctx, c, "", 10); tErr == nil {
		r.TopPods = top
	} else {
		r.Warnings = append(r.Warnings, "No se pudieron obtener los top pods: "+tErr.Error())
	}

	if quotas, qErr := Quotas(ctx, c); qErr == nil {
		r.Quotas = quotas
	} else {
		r.Warnings = append(r.Warnings, "No se pudieron obtener las quotas: "+qErr.Error())
	}

	if restarts, rErr := PodRestarts(ctx, c, 1); rErr == nil {
		r.Restarts = restarts
	} else {
		r.Warnings = append(r.Warnings, "No se pudieron obtener los reinicios: "+rErr.Error())
	}

	if components, cErr := Components(ctx, c); cErr == nil {
		r.Components = components
	} else {
		r.Warnings = append(r.Warnings, "No se pudieron obtener los componentes core: "+cErr.Error())
	}

	if karp, kErr := Karpenter(ctx, c); kErr == nil {
		r.Karpenter = karp
	} else {
		r.Warnings = append(r.Warnings, "No se pudo evaluar Karpenter: "+kErr.Error())
	}

	if dd, dErr := DatadogAgentCoverage(ctx, c); dErr == nil {
		r.Datadog = dd
	} else {
		r.Warnings = append(r.Warnings, "No se pudo evaluar la cobertura de Datadog: "+dErr.Error())
	}

	r.Findings = Recommendations(r)
	return r, nil
}
