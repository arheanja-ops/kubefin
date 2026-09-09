package render

import (
	"fmt"

	"github.com/arheanja-ops/kubefin/internal/model"
)

// CostRows renderiza filas de costo en el formato indicado.
func CostRows(rows []model.CostRow, format Format) (string, error) {
	switch format {
	case FormatJSON:
		return toJSON(rows)
	case FormatMarkdown:
		return costTable(rows).renderMarkdown(), nil
	case FormatConfluence:
		return costTable(rows).renderConfluence(), nil
	case FormatTable:
		return costTable(rows).renderTable(), nil
	default:
		return "", fmt.Errorf("formato no soportado: %q", format)
	}
}

func costTable(rows []model.CostRow) *table {
	t := newTable("DIMENSIÓN", "MONTO", "PERIODO")
	for _, r := range rows {
		period := fmt.Sprintf("%s → %s",
			r.Period.Start.Format("2006-01-02"), r.Period.End.Format("2006-01-02"))
		t.add(r.Dimension, r.Amount.String(), period)
	}
	return t
}
