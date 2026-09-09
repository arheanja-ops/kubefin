package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
	"github.com/arheanja-ops/kubefin/internal/render"
	"github.com/arheanja-ops/kubefin/internal/tools/eks"
)

// newEKSCmd construye el grupo de comandos `kubefin eks ...`.
func newEKSCmd(gf *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eks",
		Short: "Diagnóstico de clústeres EKS/Kubernetes vía client-go",
	}
	cmd.AddCommand(
		newEKSReportCmd(gf),
		newEKSNodesCmd(gf),
		newEKSNamespacesCmd(gf),
		newEKSTopCmd(gf),
		newEKSRestartsCmd(gf),
		newEKSComponentsCmd(gf),
		newEKSKarpenterCmd(gf),
		newEKSDatadogCmd(gf),
	)
	return cmd
}

// withClients resuelve el formato y construye los clientes de Kubernetes,
// devolviendo un error claro si no hay contexto válido (R1.2).
func withClients(gf *globalFlags) (*k8sx.Clients, render.Format, error) {
	format, err := render.ParseFormat(gf.output)
	if err != nil {
		return nil, "", err
	}
	clients, err := k8sx.Load(gf.kubeconfig, gf.context)
	if err != nil {
		return nil, "", err
	}
	return clients, format, nil
}

func newEKSReportCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "Reporte de diagnóstico completo (nodos, namespaces, componentes, hallazgos)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clients, format, err := withClients(gf)
			if err != nil {
				return err
			}
			report, err := eks.BuildReport(context.Background(), clients)
			if err != nil {
				return err
			}
			out, err := render.Report(report, format)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out)
			return nil
		},
	}
}

func newEKSNodesCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "nodes",
		Short: "Estado, capacidad y uso de los nodos",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clients, format, err := withClients(gf)
			if err != nil {
				return err
			}
			nodes, err := eks.Nodes(context.Background(), clients)
			if err != nil {
				return err
			}
			return renderReportSection(cmd, format, &model.Report{Context: clients.Context, Nodes: nodes})
		},
	}
}

func newEKSNamespacesCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "namespaces",
		Short: "Uso de recursos agregado por namespace",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clients, format, err := withClients(gf)
			if err != nil {
				return err
			}
			nss, err := eks.Namespaces(context.Background(), clients)
			if err != nil {
				return err
			}
			return renderReportSection(cmd, format, &model.Report{Context: clients.Context, Namespaces: nss})
		},
	}
}

func newEKSTopCmd(gf *globalFlags) *cobra.Command {
	var n int
	var namespace string
	cmd := &cobra.Command{
		Use:   "top",
		Short: "Top consumidores de recursos (pods)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clients, format, err := withClients(gf)
			if err != nil {
				return err
			}
			pods, err := eks.TopPods(context.Background(), clients, namespace, n)
			if err != nil {
				return err
			}
			return renderValue(cmd, format, pods)
		},
	}
	cmd.Flags().IntVarP(&n, "limit", "n", 10, "número de pods a mostrar")
	cmd.Flags().StringVar(&namespace, "namespace", "", "namespace (por defecto: todos)")
	return cmd
}

func newEKSRestartsCmd(gf *globalFlags) *cobra.Command {
	var minRestarts int32
	cmd := &cobra.Command{
		Use:   "restarts",
		Short: "Pods con reinicios",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clients, format, err := withClients(gf)
			if err != nil {
				return err
			}
			restarts, err := eks.PodRestarts(context.Background(), clients, minRestarts)
			if err != nil {
				return err
			}
			return renderReportSection(cmd, format, &model.Report{Context: clients.Context, Restarts: restarts})
		},
	}
	cmd.Flags().Int32Var(&minRestarts, "min", 1, "reinicios mínimos para incluir un pod")
	return cmd
}

func newEKSComponentsCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "components",
		Short: "Estado de los componentes core del clúster",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clients, format, err := withClients(gf)
			if err != nil {
				return err
			}
			comps, err := eks.Components(context.Background(), clients)
			if err != nil {
				return err
			}
			return renderReportSection(cmd, format, &model.Report{Context: clients.Context, Components: comps})
		},
	}
}

func newEKSKarpenterCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "karpenter",
		Short: "Estado de Karpenter (NodePools, EC2NodeClasses, issues)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clients, format, err := withClients(gf)
			if err != nil {
				return err
			}
			status, err := eks.Karpenter(context.Background(), clients)
			if err != nil {
				return err
			}
			return renderValue(cmd, format, status)
		},
	}
}

func newEKSDatadogCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "datadog",
		Short: "Cobertura del Datadog Agent sobre los nodos",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clients, format, err := withClients(gf)
			if err != nil {
				return err
			}
			cov, err := eks.DatadogAgentCoverage(context.Background(), clients)
			if err != nil {
				return err
			}
			return renderValue(cmd, format, cov)
		},
	}
}
