package eks

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

// coreComponents son los Deployments/DaemonSets de kube-system que se consideran
// críticos para la salud de un clúster EKS.
var coreDeployments = []string{"coredns"}
var coreDaemonSets = []string{"kube-proxy", "aws-node"}

// Components recolecta el estado de los componentes core del clúster en
// kube-system (coredns, kube-proxy, aws-node). Los componentes ausentes se
// omiten sin error.
func Components(ctx context.Context, c *k8sx.Clients) ([]model.ComponentStatus, error) {
	out := []model.ComponentStatus{}

	for _, name := range coreDeployments {
		dep, err := c.Kube.AppsV1().Deployments("kube-system").Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("no se pudo leer el deployment %q: %w", name, err)
		}
		out = append(out, deploymentStatus(dep))
	}

	for _, name := range coreDaemonSets {
		ds, err := c.Kube.AppsV1().DaemonSets("kube-system").Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("no se pudo leer el daemonset %q: %w", name, err)
		}
		out = append(out, daemonSetStatus(ds))
	}

	return out, nil
}

// DatadogAgentCoverage detecta el DaemonSet del Datadog Agent y calcula su
// cobertura sobre los nodos del clúster.
func DatadogAgentCoverage(ctx context.Context, c *k8sx.Clients) (*model.DatadogCoverage, error) {
	nodeList, err := c.Kube.CoreV1().Nodes().List(ctx, listOptions())
	if err != nil {
		return nil, fmt.Errorf("no se pudieron listar los nodos: %w", err)
	}
	totalNodes := len(nodeList.Items)

	ds := findDatadogDaemonSet(ctx, c)
	if ds == nil {
		return &model.DatadogCoverage{Installed: false, TotalNodes: totalNodes}, nil
	}

	return &model.DatadogCoverage{
		Installed:    true,
		DesiredNodes: ds.Status.DesiredNumberScheduled,
		CoveredNodes: ds.Status.NumberReady,
		TotalNodes:   totalNodes,
	}, nil
}

// findDatadogDaemonSet busca en todos los namespaces un DaemonSet cuyo nombre
// contenga "datadog". Devuelve nil si no existe.
func findDatadogDaemonSet(ctx context.Context, c *k8sx.Clients) *appsv1.DaemonSet {
	list, err := c.Kube.AppsV1().DaemonSets("").List(ctx, listOptions())
	if err != nil {
		return nil
	}
	for i := range list.Items {
		if containsFold(list.Items[i].Name, "datadog") {
			return &list.Items[i]
		}
	}
	return nil
}

func deploymentStatus(dep *appsv1.Deployment) model.ComponentStatus {
	desired := int32(1)
	if dep.Spec.Replicas != nil {
		desired = *dep.Spec.Replicas
	}
	return model.ComponentStatus{
		Name:      dep.Name,
		Namespace: dep.Namespace,
		Kind:      "Deployment",
		Desired:   desired,
		Ready:     dep.Status.ReadyReplicas,
		Healthy:   dep.Status.ReadyReplicas >= desired && desired > 0,
	}
}

func daemonSetStatus(ds *appsv1.DaemonSet) model.ComponentStatus {
	desired := ds.Status.DesiredNumberScheduled
	return model.ComponentStatus{
		Name:      ds.Name,
		Namespace: ds.Namespace,
		Kind:      "DaemonSet",
		Desired:   desired,
		Ready:     ds.Status.NumberReady,
		Healthy:   ds.Status.NumberReady >= desired,
	}
}
