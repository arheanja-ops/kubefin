package cli

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/arheanja-ops/kubefin/internal/awsx"
	"github.com/arheanja-ops/kubefin/internal/model"
	"github.com/arheanja-ops/kubefin/internal/render"
	"github.com/arheanja-ops/kubefin/internal/tools/cost"
)

// costFlags reúne las banderas de los comandos de costo.
type costFlags struct {
	bucket   string
	prefix   string
	tag      string
	last     string
	region   string
	useCEAPI bool
}

func newCostCmd(gf *globalFlags) *cobra.Command {
	cf := &costFlags{}
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "Análisis de costos AWS (CUR gratis por defecto; CE API opt-in)",
	}
	pf := cmd.PersistentFlags()
	pf.StringVar(&cf.bucket, "bucket", "", "bucket S3 del CUR")
	pf.StringVar(&cf.prefix, "prefix", "cur", "prefijo del export del CUR en el bucket")
	pf.StringVar(&cf.last, "last", "30d", "ventana temporal: Nd (días) o Nm (meses)")
	pf.StringVar(&cf.region, "region", "", "región AWS (por defecto: la del entorno)")
	pf.BoolVar(&cf.useCEAPI, "use-ce-api", false, "usar la Cost Explorer API ($0.01/request); por defecto usa el CUR gratis")

	cmd.AddCommand(newCostByServiceCmd(gf, cf), newCostByTagCmd(gf, cf))
	return cmd
}

func newCostByServiceCmd(gf *globalFlags, cf *costFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "by-service",
		Short: "Costo agrupado por servicio de AWS",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCost(cmd, gf, cf, model.GroupByService)
		},
	}
}

func newCostByTagCmd(gf *globalFlags, cf *costFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "by-tag",
		Short: "Costo agrupado por el valor de un tag de asignación de costos",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCost(cmd, gf, cf, model.GroupByTag)
		},
	}
	cmd.Flags().StringVar(&cf.tag, "tag", "", "clave del tag de asignación de costos")
	return cmd
}

// runCost ejecuta el análisis de costo por CUR (gratis) o por CE API (opt-in).
func runCost(cmd *cobra.Command, gf *globalFlags, cf *costFlags, group model.GroupBy) error {
	format, err := render.ParseFormat(gf.output)
	if err != nil {
		return err
	}
	days, err := parseWindowDays(cf.last)
	if err != nil {
		return err
	}

	ctx := context.Background()
	clients, err := awsx.Load(ctx, cf.region)
	if err != nil {
		return err
	}
	if _, err := clients.VerifyIdentity(ctx); err != nil {
		return err
	}

	rows, err := collectCost(ctx, clients, cf, group, days, cmd)
	if err != nil {
		if errors.Is(err, cost.ErrNoCUR) {
			fmt.Fprintln(cmd.OutOrStdout(), cost.NoCURGuidance)
			return nil
		}
		return err
	}

	out, err := render.CostRows(rows, format)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), out)
	return nil
}

// collectCost decide la fuente: CE API si hay opt-in, CUR en S3 en otro caso.
func collectCost(ctx context.Context, clients *awsx.Clients, cf *costFlags,
	group model.GroupBy, days int, cmd *cobra.Command) ([]model.CostRow, error) {

	if cf.useCEAPI {
		warn := func(m string) { fmt.Fprintln(cmd.ErrOrStderr(), m) }
		if err := cost.EnsureCEOptIn(true, warn); err != nil {
			return nil, err
		}
		if group == model.GroupByTag {
			return nil, errors.New("la agrupación por tag vía CE API no está soportada; usa el CUR")
		}
		ce := clients.CostExplorer()
		return cost.ByServiceCE(ctx, ce, days)
	}

	src := cost.Source{Bucket: cf.bucket, Prefix: cf.prefix, TagKey: cf.tag}
	if group == model.GroupByTag {
		return cost.ByTag(ctx, clients.S3, src, days)
	}
	return cost.ByService(ctx, clients.S3, src, days)
}

// parseWindowDays convierte "30d" o "3m" en un número de días.
func parseWindowDays(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 30, nil
	}
	unit := s[len(s)-1]
	numStr := s[:len(s)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("ventana inválida %q (usa Nd o Nm, p. ej. 30d)", s)
	}
	switch unit {
	case 'd':
		return n, nil
	case 'm':
		return n * 30, nil
	default:
		return 0, fmt.Errorf("unidad de ventana inválida en %q (usa d o m)", s)
	}
}
