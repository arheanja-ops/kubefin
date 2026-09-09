// Package model contiene los structs compartidos entre tools, render y superficies.
package model

// Severity clasifica la importancia de un hallazgo de diagnóstico.
type Severity string

const (
	// SeverityInfo indica un hallazgo informativo, sin acción requerida.
	SeverityInfo Severity = "info"
	// SeverityWarning indica una condición que merece atención.
	SeverityWarning Severity = "warning"
	// SeverityCritical indica una condición que requiere acción inmediata.
	SeverityCritical Severity = "critical"
)

// Finding es un hallazgo estructurado de diagnóstico. Es la unidad de salida
// común a las tres superficies (CLI, MCP, agente): las tools lo producen y el
// render lo presenta.
type Finding struct {
	Severity       Severity `json:"severity"`
	Title          string   `json:"title"`
	Evidence       string   `json:"evidence"`
	Recommendation string   `json:"recommendation,omitempty"`
	// Command es un comando sugerido; nunca se ejecuta automáticamente (D6, read-only v1).
	Command string `json:"command,omitempty"`
	// EstimatedSaving es el ahorro estimado si aplica; nil cuando no se puede estimar.
	EstimatedSaving *Money `json:"estimatedSaving,omitempty"`
}
