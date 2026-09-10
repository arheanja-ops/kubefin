package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/arheanja-ops/kubefin/internal/agent"
	"github.com/arheanja-ops/kubefin/internal/ai"
	"github.com/arheanja-ops/kubefin/internal/mcp"
	"github.com/arheanja-ops/kubefin/internal/render"
	"github.com/arheanja-ops/kubefin/internal/toolset"
)

func newMCPCmd(gf *globalFlags, version string) *cobra.Command {
	var region string
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Inicia el servidor MCP por stdio (para clientes LLM como Kiro o Claude Desktop)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps := toolset.Deps{
				Kubeconfig:  gf.kubeconfig,
				KubeContext: gf.context,
				Region:      region,
			}
			return mcp.Serve(cmd.Context(), version, toolset.Default(), deps)
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "región AWS para las tools de costo (por defecto: la del entorno)")
	return cmd
}

func newAskCmd(gf *globalFlags) *cobra.Command {
	var region string
	var redact bool
	cmd := &cobra.Command{
		Use:   "ask \"<pregunta>\"",
		Short: "Pregunta en lenguaje natural; el agente usa las tools y responde con hallazgos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, err := render.ParseFormat(gf.output)
			if err != nil {
				return err
			}
			provider, err := ai.NewGroq()
			if err != nil {
				return err
			}
			deps := toolset.Deps{
				Kubeconfig:  gf.kubeconfig,
				KubeContext: gf.context,
				Region:      region,
			}
			ag := agent.New(provider, toolset.Default(), ai.NewRedactor(redact), deps)

			findings, err := ag.Ask(context.Background(), args[0])
			if err != nil {
				return err
			}
			out, err := render.Findings(findings, format)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "región AWS para las tools de costo (por defecto: la del entorno)")
	cmd.Flags().BoolVar(&redact, "redact", false, "redactar account IDs, ARNs e IPs antes de enviar al proveedor de IA")
	return cmd
}
