package eks

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

// Nodes recolecta el estado, capacidad y (si hay métricas) el uso de cada nodo.
// Si Metrics Server no está disponible, Usage queda nil y no es un error (R1.4).
func Nodes(ctx context.Context, c *k8sx.Clients) ([]model.Node, error) {
	nodeList, err := c.Kube.CoreV1().Nodes().List(ctx, listOptions())
	if err != nil {
		return nil, fmt.Errorf("no se pudieron listar los nodos: %w", err)
	}

	usage := nodeUsage(ctx, c)

	nodes := make([]model.Node, 0, len(nodeList.Items))
	for i := range nodeList.Items {
		n := &nodeList.Items[i]
		node := model.Node{
			Name:         n.Name,
			Ready:        nodeReady(n),
			Schedulable:  !n.Spec.Unschedulable,
			InstanceType: n.Labels["node.kubernetes.io/instance-type"],
			Zone:         n.Labels["topology.kubernetes.io/zone"],
			Capacity:     resourceFromList(n.Status.Capacity),
			Allocatable:  resourceFromList(n.Status.Allocatable),
			Labels:       n.Labels,
		}
		if u, ok := usage[n.Name]; ok {
			node.Usage = &u
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// nodeReady devuelve true si la condición Ready del nodo es True.
func nodeReady(n *corev1.Node) bool {
	for _, cond := range n.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

// nodeUsage devuelve el uso por nombre de nodo. Vacío si no hay métricas (R1.4).
func nodeUsage(ctx context.Context, c *k8sx.Clients) map[string]model.Resource {
	out := map[string]model.Resource{}
	if c.Metrics == nil {
		return out
	}
	metrics, err := c.Metrics.NodeMetrics(ctx)
	if err != nil {
		return out
	}
	for i := range metrics.Items {
		m := &metrics.Items[i]
		out[m.Name] = model.Resource{
			CPUMilli:    quantityMilliCPU(m.Usage.Cpu().DeepCopy()),
			MemoryBytes: quantityBytes(m.Usage.Memory().DeepCopy()),
		}
	}
	return out
}
