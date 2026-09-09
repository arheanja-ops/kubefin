package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/arheanja-ops/kubefin/internal/model"
	"github.com/arheanja-ops/kubefin/internal/render"
)

// renderReportSection renderiza un Report parcial (una sección) en el formato
// pedido reutilizando el renderizador de reportes.
func renderReportSection(cmd *cobra.Command, format render.Format, r *model.Report) error {
	out, err := render.Report(r, format)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), out)
	return nil
}

// renderValue renderiza un valor sin tabla dedicada. En formato table cae a una
// representación JSON legible; los formatos estructurados usan JSON.
func renderValue(cmd *cobra.Command, format render.Format, v any) error {
	switch format {
	case render.FormatJSON, render.FormatTable, render.FormatMarkdown, render.FormatConfluence:
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return fmt.Errorf("no se pudo serializar la salida: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
		return nil
	default:
		return fmt.Errorf("formato no soportado: %q", format)
	}
}
