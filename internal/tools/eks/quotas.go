package eks

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

// Quotas recolecta las ResourceQuotas de todos los namespaces con su uso vs.
// límite duro.
func Quotas(ctx context.Context, c *k8sx.Clients) ([]model.ResourceQuota, error) {
	list, err := c.Kube.CoreV1().ResourceQuotas("").List(ctx, listOptions())
	if err != nil {
		return nil, fmt.Errorf("no se pudieron listar las resource quotas: %w", err)
	}
	out := make([]model.ResourceQuota, 0, len(list.Items))
	for i := range list.Items {
		q := &list.Items[i]
		out = append(out, model.ResourceQuota{
			Namespace: q.Namespace,
			Name:      q.Name,
			Hard:      resourceListToStrings(q.Status.Hard),
			Used:      resourceListToStrings(q.Status.Used),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Namespace != out[j].Namespace {
			return out[i].Namespace < out[j].Namespace
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// resourceListToStrings convierte una ResourceList a un mapa nombre->cantidad.
func resourceListToStrings(rl corev1.ResourceList) map[string]string {
	if len(rl) == 0 {
		return nil
	}
	out := make(map[string]string, len(rl))
	for name, q := range rl {
		out[string(name)] = q.String()
	}
	return out
}
