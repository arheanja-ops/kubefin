package eks

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kfake "k8s.io/client-go/kubernetes/fake"
	metricsapi "k8s.io/metrics/pkg/apis/metrics/v1beta1"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

// stubMetrics implementa k8sx.Metrics para tests, evitando el bug del fake
// clientset oficial (kubernetes/kubernetes#93853).
type stubMetrics struct {
	nodes *metricsapi.NodeMetricsList
	pods  *metricsapi.PodMetricsList
	err   error
}

func (s stubMetrics) NodeMetrics(context.Context) (*metricsapi.NodeMetricsList, error) {
	return s.nodes, s.err
}

func (s stubMetrics) PodMetrics(context.Context, string) (*metricsapi.PodMetricsList, error) {
	return s.pods, s.err
}

func newNode(name string, ready, schedulable bool, cpu, mem string) *corev1.Node {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"node.kubernetes.io/instance-type": "t3.medium",
				"topology.kubernetes.io/zone":      "us-east-1a",
			},
		},
		Spec: corev1.NodeSpec{Unschedulable: !schedulable},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{{Type: corev1.NodeReady, Status: status}},
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(mem),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(mem),
			},
		},
	}
}

func nodeMetric(name, cpu, mem string) metricsapi.NodeMetrics {
	return metricsapi.NodeMetrics{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Usage: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(cpu),
			corev1.ResourceMemory: resource.MustParse(mem),
		},
	}
}

func TestNodes_WithMetrics(t *testing.T) {
	kube := kfake.NewSimpleClientset(
		newNode("node-a", true, true, "2", "4Gi"),
		newNode("node-b", false, true, "2", "4Gi"),
	)
	metrics := stubMetrics{nodes: &metricsapi.NodeMetricsList{
		Items: []metricsapi.NodeMetrics{nodeMetric("node-a", "500m", "1Gi")},
	}}
	c := &k8sx.Clients{Kube: kube, Metrics: metrics, Context: "test"}

	nodes, err := Nodes(context.Background(), c)
	if err != nil {
		t.Fatalf("Nodes() error: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("esperaba 2 nodos, obtuve %d", len(nodes))
	}

	var a, b model.Node
	for _, n := range nodes {
		switch n.Name {
		case "node-a":
			a = n
		case "node-b":
			b = n
		}
	}
	if !a.Ready {
		t.Errorf("node-a debería estar Ready")
	}
	if a.Usage == nil {
		t.Fatalf("node-a debería tener uso de métricas")
	}
	if a.Usage.CPUMilli != 500 {
		t.Errorf("uso CPU node-a: esperaba 500m, obtuve %dm", a.Usage.CPUMilli)
	}
	if b.Ready {
		t.Errorf("node-b no debería estar Ready")
	}
	if b.Usage != nil {
		t.Errorf("node-b no tiene métricas; Usage debería ser nil")
	}
}

func TestNodes_NoMetrics(t *testing.T) {
	kube := kfake.NewSimpleClientset(newNode("node-a", true, true, "2", "4Gi"))
	c := &k8sx.Clients{Kube: kube, Metrics: nil, Context: "test"}

	nodes, err := Nodes(context.Background(), c)
	if err != nil {
		t.Fatalf("Nodes() error: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("esperaba 1 nodo, obtuve %d", len(nodes))
	}
	if nodes[0].Usage != nil {
		t.Errorf("sin metrics client, Usage debe ser nil (R1.4)")
	}
}
