package render

import (
	"fmt"
	"strings"
)

// table representa datos tabulares con encabezado y filas.
type table struct {
	headers []string
	rows    [][]string
}

func newTable(headers ...string) *table { return &table{headers: headers} }

func (t *table) add(cells ...string) { t.rows = append(t.rows, cells) }

// renderTable produce una tabla de texto alineada por columnas.
func (t *table) renderTable() string {
	widths := t.columnWidths()
	var b strings.Builder
	writeRow(&b, t.headers, widths)
	sep := make([]string, len(t.headers))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}
	writeRow(&b, sep, widths)
	for _, row := range t.rows {
		writeRow(&b, row, widths)
	}
	return b.String()
}

// renderMarkdown produce una tabla en Markdown (GitHub-flavored).
func (t *table) renderMarkdown() string {
	var b strings.Builder
	b.WriteString("| " + strings.Join(t.headers, " | ") + " |\n")
	seps := make([]string, len(t.headers))
	for i := range seps {
		seps[i] = "---"
	}
	b.WriteString("| " + strings.Join(seps, " | ") + " |\n")
	for _, row := range t.rows {
		b.WriteString("| " + strings.Join(escapeCells(row, "|", "\\|"), " | ") + " |\n")
	}
	return b.String()
}

// renderConfluence produce una tabla en Confluence Wiki Markup.
// Encabezados con "||", celdas con "|".
func (t *table) renderConfluence() string {
	var b strings.Builder
	b.WriteString("||" + strings.Join(t.headers, "||") + "||\n")
	for _, row := range t.rows {
		b.WriteString("|" + strings.Join(escapeCells(row, "|", "\\|"), "|") + "|\n")
	}
	return b.String()
}

func (t *table) columnWidths() []int {
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	return widths
}

func writeRow(b *strings.Builder, cells []string, widths []int) {
	parts := make([]string, len(widths))
	for i := range widths {
		cell := ""
		if i < len(cells) {
			cell = cells[i]
		}
		parts[i] = fmt.Sprintf("%-*s", widths[i], cell)
	}
	b.WriteString(strings.TrimRight(strings.Join(parts, "  "), " "))
	b.WriteString("\n")
}

func escapeCells(cells []string, from, to string) []string {
	out := make([]string, len(cells))
	for i, c := range cells {
		out[i] = strings.ReplaceAll(c, from, to)
	}
	return out
}
