package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/arheanja-ops/kubefin/internal/model"
)

// Report renderiza un reporte de diagnóstico EKS en el formato indicado.
func Report(r *model.Report, format Format) (string, error) {
	switch format {
	case FormatJSON:
		return toJSON(r)
	case FormatTable:
		return reportText(r, sectionsForTable), nil
	case FormatMarkdown:
		return reportMarkdown(r), nil
	case FormatConfluence:
		return reportConfluence(r), nil
	default:
		return "", fmt.Errorf("formato no soportado: %q", format)
	}
}

// Findings renderiza una lista de hallazgos en el formato indicado.
func Findings(findings []model.Finding, format Format) (string, error) {
	switch format {
	case FormatJSON:
		return toJSON(findings)
	case FormatMarkdown:
		return findingsTable(findings).renderMarkdown(), nil
	case FormatConfluence:
		return findingsTable(findings).renderConfluence(), nil
	case FormatTable:
		return findingsTable(findings).renderTable(), nil
	default:
		return "", fmt.Errorf("formato no soportado: %q", format)
	}
}

func toJSON(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", fmt.Errorf("no se pudo serializar a JSON: %w", err)
	}
	return string(b), nil
}

type tableFn func(*table) string

func sectionsForTable(t *table) string { return t.renderTable() }

// reportText ensambla las secciones de texto/tabla del reporte.
func reportText(r *model.Report, render tableFn) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Diagnóstico EKS — contexto: %s\n", r.Context)
	if !r.MetricsAvailable {
		b.WriteString("(Metrics Server no disponible: sin datos de uso real)\n")
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(&b, "! %s\n", w)
	}
	b.WriteString("\nNodos:\n")
	b.WriteString(render(nodesTable(r.Nodes)))
	b.WriteString("\nNamespaces:\n")
	b.WriteString(render(namespacesTable(r.Namespaces)))
	if len(r.Components) > 0 {
		b.WriteString("\nComponentes core:\n")
		b.WriteString(render(componentsTable(r.Components)))
	}
	if len(r.Restarts) > 0 {
		b.WriteString("\nReinicios:\n")
		b.WriteString(render(restartsTable(r.Restarts)))
	}
	b.WriteString("\nHallazgos:\n")
	b.WriteString(render(findingsTable(r.Findings)))
	return b.String()
}

func reportMarkdown(r *model.Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Diagnóstico EKS — `%s`\n\n", r.Context)
	if !r.MetricsAvailable {
		b.WriteString("> Metrics Server no disponible: sin datos de uso real.\n\n")
	}
	b.WriteString("## Nodos\n\n" + nodesTable(r.Nodes).renderMarkdown() + "\n")
	b.WriteString("## Namespaces\n\n" + namespacesTable(r.Namespaces).renderMarkdown() + "\n")
	if len(r.Components) > 0 {
		b.WriteString("## Componentes core\n\n" + componentsTable(r.Components).renderMarkdown() + "\n")
	}
	if len(r.Restarts) > 0 {
		b.WriteString("## Reinicios\n\n" + restartsTable(r.Restarts).renderMarkdown() + "\n")
	}
	b.WriteString("## Hallazgos\n\n" + findingsTable(r.Findings).renderMarkdown() + "\n")
	return b.String()
}

func reportConfluence(r *model.Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "h1. Diagnóstico EKS — %s\n\n", r.Context)
	if !r.MetricsAvailable {
		b.WriteString("{note}Metrics Server no disponible: sin datos de uso real.{note}\n\n")
	}
	b.WriteString("h2. Nodos\n" + nodesTable(r.Nodes).renderConfluence() + "\n")
	b.WriteString("h2. Namespaces\n" + namespacesTable(r.Namespaces).renderConfluence() + "\n")
	if len(r.Components) > 0 {
		b.WriteString("h2. Componentes core\n" + componentsTable(r.Components).renderConfluence() + "\n")
	}
	if len(r.Restarts) > 0 {
		b.WriteString("h2. Reinicios\n" + restartsTable(r.Restarts).renderConfluence() + "\n")
	}
	b.WriteString("h2. Hallazgos\n" + findingsTable(r.Findings).renderConfluence() + "\n")
	return b.String()
}
