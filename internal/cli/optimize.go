package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/arheanja-ops/kubefin/internal/awsx"
	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
	"github.com/arheanja-ops/kubefin/internal/render"
	"github.com/arheanja-ops/kubefin/internal/tools/eks"
	"github.com/arheanja-ops/kubefin/internal/tools/optimize"
)

func newOptimizeCmd(gf *globalFlags) *cobra.Command {
	var region string
	var correlate bool
	cmd := &cobra.Command{
		Use:   "optimize",
		Short: "Recomendaciones de rightsizing (Compute Optimizer + correlación con pods)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			format, err := render.ParseFormat(gf.output)
			if err != nil {
				return err
			}
			ctx := context.Background()

			clients, err := awsx.Load(ctx, region)
			if err != nil {
				return err
			}
			if _, err := clients.VerifyIdentity(ctx); err != nil {
				return err
			}

			findings, err := optimize.ComputeOptimizer(ctx, clients.ComputeOptimizer)
			if err != nil {
				return err
			}

			if correlate {
				corr, cErr := correlatePods(ctx, gf)
				if cErr != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "! no se pudo correlacionar con el clúster: %v\n", cErr)
				} else {
					findings = append(findings, corr...)
				}
			}

			out, err := render.Findings(findings, format)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out)
			return nil
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "región AWS (por defecto: la del entorno)")
	cmd.Flags().BoolVar(&correlate, "correlate", false, "correlacionar con el uso real de pods del clúster (requiere kubeconfig)")
	return cmd
}

// correlatePods obtiene el uso por namespace del clúster y deriva hallazgos de
// rightsizing de pods (R3.2).
func correlatePods(ctx context.Context, gf *globalFlags) ([]model.Finding, error) {
	clients, err := k8sx.Load(gf.kubeconfig, gf.context)
	if err != nil {
		return nil, err
	}
	nss, err := eks.Namespaces(ctx, clients)
	if err != nil {
		return nil, err
	}
	return optimize.CorrelatePodRightsizing(nss), nil
}
