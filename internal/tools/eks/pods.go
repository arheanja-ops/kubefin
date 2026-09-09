package eks

import (
	"context"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

// Pods recolecta todos los pods del clúster con sus requests, limits, reinicios
// y (si hay métricas) su uso real. namespace vacío significa todos los namespaces.
func Pods(ctx context.Context, c *k8sx.Clients, namespace string) ([]model.Pod, error) {
	podList, err := c.Kube.CoreV1().Pods(namespace).List(ctx, listOptions())
	if err != nil {
		return nil, fmt.Errorf("no se pudieron listar los pods: %w", err)
	}

	usage := podUsage(ctx, c, namespace)

	pods := make([]model.Pod, 0, len(podList.Items))
	for i := range podList.Items {
		p := &podList.Items[i]
		pod := model.Pod{
			Name:      p.Name,
			Namespace: p.Namespace,
			Node:      p.Spec.NodeName,
			Phase:     string(p.Status.Phase),
			Restarts:  podRestarts(p),
			Requests:  containersResources(p, requestsSelector),
			Limits:    containersResources(p, limitsSelector),
		}
		if u, ok := usage[p.Namespace+"/"+p.Name]; ok {
			pod.Usage = &u
		}
		pods = append(pods, pod)
	}
	return pods, nil
}

// TopPods devuelve los n pods con mayor uso de CPU (requiere métricas). Si no hay
// métricas, ordena por CPU solicitada como aproximación. n<=0 devuelve todos.
func TopPods(ctx context.Context, c *k8sx.Clients, namespace string, n int) ([]model.Pod, error) {
	pods, err := Pods(ctx, c, namespace)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(pods, func(i, j int) bool {
		return podSortKey(pods[i]) > podSortKey(pods[j])
	})
	if n > 0 && n < len(pods) {
		pods = pods[:n]
	}
	return pods, nil
}

// Namespaces agrega requests, limits, uso y conteo de pods por namespace.
func Namespaces(ctx context.Context, c *k8sx.Clients) ([]model.NamespaceUsage, error) {
	pods, err := Pods(ctx, c, "")
	if err != nil {
		return nil, err
	}

	byNS := map[string]*model.NamespaceUsage{}
	order := []string{}
	for _, p := range pods {
		nu, ok := byNS[p.Namespace]
		if !ok {
			nu = &model.NamespaceUsage{Namespace: p.Namespace}
			byNS[p.Namespace] = nu
			order = append(order, p.Namespace)
		}
		nu.PodCount++
		addResource(&nu.Requests, p.Requests)
		addResource(&nu.Limits, p.Limits)
		if p.Usage != nil {
			if nu.Usage == nil {
				nu.Usage = &model.Resource{}
			}
			addResource(nu.Usage, *p.Usage)
		}
	}

	sort.Strings(order)
	out := make([]model.NamespaceUsage, 0, len(order))
	for _, ns := range order {
		out = append(out, *byNS[ns])
	}
	return out, nil
}

// PodRestarts devuelve los pods con al menos minRestarts reinicios, ordenados
// de mayor a menor.
func PodRestarts(ctx context.Context, c *k8sx.Clients, minRestarts int32) ([]model.PodRestart, error) {
	podList, err := c.Kube.CoreV1().Pods("").List(ctx, listOptions())
	if err != nil {
		return nil, fmt.Errorf("no se pudieron listar los pods: %w", err)
	}
	out := []model.PodRestart{}
	for i := range podList.Items {
		p := &podList.Items[i]
		r := podRestarts(p)
		if r < minRestarts {
			continue
		}
		out = append(out, model.PodRestart{
			Name:      p.Name,
			Namespace: p.Namespace,
			Restarts:  r,
			Reason:    lastTerminationReason(p),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Restarts > out[j].Restarts })
	return out, nil
}

// --- helpers internos ---

type resourceSelector func(corev1.Container) corev1.ResourceList

func requestsSelector(ct corev1.Container) corev1.ResourceList { return ct.Resources.Requests }
func limitsSelector(ct corev1.Container) corev1.ResourceList   { return ct.Resources.Limits }

// containersResources suma requests o limits de todos los contenedores del pod.
func containersResources(p *corev1.Pod, sel resourceSelector) model.Resource {
	var total model.Resource
	for i := range p.Spec.Containers {
		addResource(&total, resourceFromList(sel(p.Spec.Containers[i])))
	}
	return total
}

// podRestarts suma los reinicios de todos los contenedores del pod.
func podRestarts(p *corev1.Pod) int32 {
	var total int32
	for i := range p.Status.ContainerStatuses {
		total += p.Status.ContainerStatuses[i].RestartCount
	}
	return total
}

// lastTerminationReason devuelve el motivo de la última terminación de un
// contenedor, si existe.
func lastTerminationReason(p *corev1.Pod) string {
	for i := range p.Status.ContainerStatuses {
		if term := p.Status.ContainerStatuses[i].LastTerminationState.Terminated; term != nil {
			return term.Reason
		}
	}
	return ""
}

// podSortKey usa el uso de CPU si existe, o la CPU solicitada como aproximación.
func podSortKey(p model.Pod) int64 {
	if p.Usage != nil {
		return p.Usage.CPUMilli
	}
	return p.Requests.CPUMilli
}

// podUsage devuelve el uso por "ns/name". Vacío si no hay métricas (R1.4).
func podUsage(ctx context.Context, c *k8sx.Clients, namespace string) map[string]model.Resource {
	out := map[string]model.Resource{}
	if c.Metrics == nil {
		return out
	}
	metrics, err := c.Metrics.PodMetrics(ctx, namespace)
	if err != nil {
		return out
	}
	for i := range metrics.Items {
		m := &metrics.Items[i]
		var agg model.Resource
		for j := range m.Containers {
			cont := &m.Containers[j]
			agg.CPUMilli += quantityMilliCPU(cont.Usage.Cpu().DeepCopy())
			agg.MemoryBytes += quantityBytes(cont.Usage.Memory().DeepCopy())
		}
		out[m.Namespace+"/"+m.Name] = agg
	}
	return out
}
