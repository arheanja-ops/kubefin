package model

import (
	"fmt"
	"time"
)

// Money representa un monto monetario con su divisa. El valor se guarda como
// float64 en la unidad de la divisa (p. ej. dólares), suficiente para reportes
// de costo agregados; no se usa para contabilidad de precisión.
type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// String formatea el monto con dos decimales y su divisa.
func (m Money) String() string {
	cur := m.Currency
	if cur == "" {
		cur = "USD"
	}
	return fmt.Sprintf("%.2f %s", m.Amount, cur)
}

// Period es un intervalo de tiempo [Start, End) para agrupar costos.
type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// CostRow es una fila de costo agrupada por una dimensión (servicio, tag, cuenta)
// dentro de un periodo.
type CostRow struct {
	Dimension string `json:"dimension"`
	Period    Period `json:"period"`
	Amount    Money  `json:"amount"`
}

// GroupBy enumera las dimensiones por las que se puede agrupar el costo (R2.4).
type GroupBy string

const (
	// GroupByService agrupa por servicio de AWS (lineItem/ProductCode).
	GroupByService GroupBy = "service"
	// GroupByTag agrupa por el valor de un tag de asignación de costos.
	GroupByTag GroupBy = "tag"
)
