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
)

func newPod(name, ns, node string, cpuReq string, restarts int32) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns},
		Spec: corev1.PodSpec{
			NodeName: node,
			Containers: []corev1.Container{{
				Name: "main",
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpuReq)},
				},
			}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:         "main",
				RestartCount: restarts,
				LastTerminationState: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"},
				},
			}},
		},
	}
}

func TestPods_RequestsAndRestarts(t *testing.T) {
	kube := kfake.NewSimpleClientset(
		newPod("p1", "default", "node-a", "250m", 0),
		newPod("p2", "kube-system", "node-a", "100m", 7),
	)
	c := &k8sx.Clients{Kube: kube, Context: "test"}

	pods, err := Pods(context.Background(), c, "")
	if err != nil {
		t.Fatalf("Pods() error: %v", err)
	}
	if len(pods) != 2 {
		t.Fatalf("esperaba 2 pods, obtuve %d", len(pods))
	}
	for _, p := range pods {
		if p.Name == "p1" && p.Requests.CPUMilli != 250 {
			t.Errorf("p1 CPU req: esperaba 250, obtuve %d", p.Requests.CPUMilli)
		}
		if p.Name == "p2" && p.Restarts != 7 {
			t.Errorf("p2 reinicios: esperaba 7, obtuve %d", p.Restarts)
		}
	}
}

func TestPodRestarts_FiltersAndSorts(t *testing.T) {
	kube := kfake.NewSimpleClientset(
		newPod("p1", "default", "node-a", "250m", 2),
		newPod("p2", "default", "node-a", "100m", 9),
		newPod("p3", "default", "node-a", "100m", 0),
	)
	c := &k8sx.Clients{Kube: kube, Context: "test"}

	rs, err := PodRestarts(context.Background(), c, 1)
	if err != nil {
		t.Fatalf("PodRestarts() error: %v", err)
	}
	if len(rs) != 2 {
		t.Fatalf("esperaba 2 pods con >=1 reinicio, obtuve %d", len(rs))
	}
	if rs[0].Restarts != 9 || rs[1].Restarts != 2 {
		t.Errorf("orden incorrecto: %+v", rs)
	}
	if rs[0].Reason != "OOMKilled" {
		t.Errorf("motivo esperado OOMKilled, obtuve %q", rs[0].Reason)
	}
}

func TestNamespaces_AggregatesWithUsage(t *testing.T) {
	kube := kfake.NewSimpleClientset(
		newPod("p1", "default", "node-a", "250m", 0),
		newPod("p2", "default", "node-a", "250m", 0),
		newPod("p3", "kube-system", "node-a", "100m", 0),
	)
	metrics := stubMetrics{pods: &metricsapi.PodMetricsList{
		Items: []metricsapi.PodMetrics{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "p1", Namespace: "default"},
				Containers: []metricsapi.ContainerMetrics{{
					Name:  "main",
					Usage: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m")},
				}},
			},
		},
	}}
	c := &k8sx.Clients{Kube: kube, Metrics: metrics, Context: "test"}

	nss, err := Namespaces(context.Background(), c)
	if err != nil {
		t.Fatalf("Namespaces() error: %v", err)
	}
	if len(nss) != 2 {
		t.Fatalf("esperaba 2 namespaces, obtuve %d", len(nss))
	}
	// default (ordenado alfabéticamente) va primero.
	if nss[0].Namespace != "default" {
		t.Fatalf("esperaba 'default' primero, obtuve %q", nss[0].Namespace)
	}
	if nss[0].PodCount != 2 {
		t.Errorf("default PodCount: esperaba 2, obtuve %d", nss[0].PodCount)
	}
	if nss[0].Requests.CPUMilli != 500 {
		t.Errorf("default CPU req: esperaba 500, obtuve %d", nss[0].Requests.CPUMilli)
	}
	if nss[0].Usage == nil || nss[0].Usage.CPUMilli != 50 {
		t.Errorf("default uso: esperaba 50m, obtuve %+v", nss[0].Usage)
	}
}

func TestTopPods_LimitAndOrder(t *testing.T) {
	kube := kfake.NewSimpleClientset(
		newPod("small", "default", "node-a", "100m", 0),
		newPod("big", "default", "node-a", "800m", 0),
		newPod("mid", "default", "node-a", "400m", 0),
	)
	c := &k8sx.Clients{Kube: kube, Context: "test"}

	top, err := TopPods(context.Background(), c, "", 2)
	if err != nil {
		t.Fatalf("TopPods() error: %v", err)
	}
	if len(top) != 2 {
		t.Fatalf("esperaba 2 pods, obtuve %d", len(top))
	}
	if top[0].Name != "big" || top[1].Name != "mid" {
		t.Errorf("orden esperado [big mid], obtuve [%s %s]", top[0].Name, top[1].Name)
	}
}
