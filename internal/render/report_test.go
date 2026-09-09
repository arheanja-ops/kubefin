package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/arheanja-ops/kubefin/internal/model"
)

func sampleReport() *model.Report {
	return &model.Report{
		Context:          "minikube",
		MetricsAvailable: true,
		Nodes: []model.Node{{
			Name: "node-a", Ready: true, Schedulable: true,
			InstanceType: "t3.medium", Zone: "us-east-1a",
			Allocatable: model.Resource{CPUMilli: 2000, MemoryBytes: 4 * 1024 * 1024 * 1024},
			Usage:       &model.Resource{CPUMilli: 500, MemoryBytes: 1 * 1024 * 1024 * 1024},
		}},
		Namespaces: []model.NamespaceUsage{{
			Namespace: "default", PodCount: 3,
			Requests: model.Resource{CPUMilli: 750},
		}},
		Findings: []model.Finding{{
			Severity: model.SeverityWarning,
			Title:    "Namespace default sobredimensionado",
			Evidence: "req 750m vs uso 100m",
		}},
	}
}

func TestParseFormat(t *testing.T) {
	cases := map[string]bool{
		"table": true, "JSON": true, "markdown": true, "confluence": true,
		"": false, "xml": false,
	}
	for in, ok := range cases {
		_, err := ParseFormat(in)
		if ok && err != nil {
			t.Errorf("ParseFormat(%q) esperaba éxito, obtuve %v", in, err)
		}
		if !ok && err == nil {
			t.Errorf("ParseFormat(%q) esperaba error", in)
		}
	}
}

func TestReport_JSONRoundTrip(t *testing.T) {
	out, err := Report(sampleReport(), FormatJSON)
	if err != nil {
		t.Fatalf("Report(JSON) error: %v", err)
	}
	var decoded model.Report
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("JSON no deserializable: %v", err)
	}
	if decoded.Context != "minikube" {
		t.Errorf("contexto esperado minikube, obtuve %q", decoded.Context)
	}
	if len(decoded.Nodes) != 1 || decoded.Nodes[0].Usage == nil {
		t.Errorf("el uso del nodo debería preservarse en el round-trip JSON")
	}
}

func TestReport_TableContainsData(t *testing.T) {
	out, err := Report(sampleReport(), FormatTable)
	if err != nil {
		t.Fatalf("Report(table) error: %v", err)
	}
	for _, want := range []string{"node-a", "default", "sobredimensionado", "Nodos:", "Hallazgos:"} {
		if !strings.Contains(out, want) {
			t.Errorf("tabla debería contener %q\n%s", want, out)
		}
	}
}

func TestReport_MarkdownAndConfluence(t *testing.T) {
	md, err := Report(sampleReport(), FormatMarkdown)
	if err != nil {
		t.Fatalf("Report(markdown) error: %v", err)
	}
	if !strings.Contains(md, "| NODO |") || !strings.Contains(md, "# Diagnóstico EKS") {
		t.Errorf("markdown mal formado:\n%s", md)
	}

	cf, err := Report(sampleReport(), FormatConfluence)
	if err != nil {
		t.Fatalf("Report(confluence) error: %v", err)
	}
	if !strings.Contains(cf, "||NODO||") || !strings.Contains(cf, "h1. Diagnóstico EKS") {
		t.Errorf("confluence mal formado:\n%s", cf)
	}
}

func TestReport_MetricsWarning(t *testing.T) {
	r := sampleReport()
	r.MetricsAvailable = false
	out, err := Report(r, FormatTable)
	if err != nil {
		t.Fatalf("Report error: %v", err)
	}
	if !strings.Contains(out, "Metrics Server no disponible") {
		t.Errorf("debería avisar de la ausencia de Metrics Server (R1.4)")
	}
}
