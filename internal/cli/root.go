// Package cli define los comandos cobra (superficie humana) sobre las tools core.
package cli

import (
	"github.com/spf13/cobra"
)

// globalFlags reúne las banderas compartidas por los subcomandos.
type globalFlags struct {
	kubeconfig string
	context    string
	output     string
}

// NewRootCmd construye el comando raíz de kubefin con sus subcomandos.
// version se inyecta desde main.
func NewRootCmd(version string) *cobra.Command {
	gf := &globalFlags{}

	root := &cobra.Command{
		Use:   "kubefin",
		Short: "Diagnóstico de EKS y análisis de costos AWS (FinOps) con IA",
		Long: "kubefin unifica el diagnóstico de clústeres EKS con el análisis de costos AWS,\n" +
			"exponiendo su lógica vía CLI, servidor MCP y un agente de IA.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: false,
	}

	pf := root.PersistentFlags()
	pf.StringVar(&gf.kubeconfig, "kubeconfig", "", "ruta al kubeconfig (por defecto: KUBECONFIG o ~/.kube/config)")
	pf.StringVar(&gf.context, "context", "", "contexto de kubectl a usar (por defecto: el actual)")
	pf.StringVarP(&gf.output, "output", "o", "table", "formato de salida: table, json, markdown, confluence")

	root.AddCommand(newEKSCmd(gf))
	root.AddCommand(newCostCmd(gf))
	root.AddCommand(newOptimizeCmd(gf))
	return root
}
