// Package render formatea la salida en tabla, JSON, Markdown y Confluence.
//
// render consume los structs de internal/model y produce texto. No es importado
// por las tools (el flujo de dependencias apunta al core).
package render

import (
	"fmt"
	"strings"
)

// Format enumera los formatos de salida soportados (R1.5).
type Format string

const (
	// FormatTable es una tabla de texto para terminal.
	FormatTable Format = "table"
	// FormatJSON es JSON indentado.
	FormatJSON Format = "json"
	// FormatMarkdown es Markdown estándar (GitHub-flavored).
	FormatMarkdown Format = "markdown"
	// FormatConfluence es Confluence Wiki Markup.
	FormatConfluence Format = "confluence"
)

// ParseFormat valida y normaliza el nombre de un formato.
func ParseFormat(s string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(s))) {
	case FormatTable:
		return FormatTable, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatMarkdown:
		return FormatMarkdown, nil
	case FormatConfluence:
		return FormatConfluence, nil
	default:
		return "", fmt.Errorf("formato desconocido %q (usa: table, json, markdown, confluence)", s)
	}
}
