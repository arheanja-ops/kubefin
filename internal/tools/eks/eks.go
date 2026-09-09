// Package eks contiene las tools de diagnóstico de clústeres EKS vía client-go.
//
// Las tools son funciones puras de recolección: reciben un *k8sx.Clients y
// devuelven structs de internal/model. No importan render, cli, mcp ni agent
// (regla estructural del diseño). Toleran la ausencia de Metrics Server sin
// abortar (R1.4).
package eks

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

// resourceFromList convierte una ResourceList de Kubernetes al modelo interno.
func resourceFromList(rl corev1.ResourceList) model.Resource {
	var r model.Resource
	if cpu, ok := rl[corev1.ResourceCPU]; ok {
		r.CPUMilli = cpu.MilliValue()
	}
	if mem, ok := rl[corev1.ResourceMemory]; ok {
		r.MemoryBytes = mem.Value()
	}
	return r
}

// quantityMilliCPU extrae millicores de una Quantity.
func quantityMilliCPU(q resource.Quantity) int64 { return q.MilliValue() }

// quantityBytes extrae bytes de una Quantity.
func quantityBytes(q resource.Quantity) int64 { return q.Value() }

// addResource suma b dentro de a (acumulador).
func addResource(a *model.Resource, b model.Resource) {
	a.CPUMilli += b.CPUMilli
	a.MemoryBytes += b.MemoryBytes
}

// listOptions es el ListOptions vacío estándar reutilizado por las tools.
func listOptions() metav1.ListOptions { return metav1.ListOptions{} }

// metricsAvailable indica si hay cliente de métricas y responde.
func metricsAvailable(ctx context.Context, c *k8sx.Clients) bool {
	if c.Metrics == nil {
		return false
	}
	_, err := c.Metrics.NodeMetrics(ctx)
	return err == nil
}
