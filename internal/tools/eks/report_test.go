package eks

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kfake "k8s.io/client-go/kubernetes/fake"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

func ptr32(v int32) *int32 { return &v }

func TestComponents_HealthAndAbsence(t *testing.T) {
	kube := kfake.NewSimpleClientset(
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "coredns", Namespace: "kube-system"},
			Spec:       appsv1.DeploymentSpec{Replicas: ptr32(2)},
			Status:     appsv1.DeploymentStatus{ReadyReplicas: 1},
		},
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "aws-node", Namespace: "kube-system"},
			Status:     appsv1.DaemonSetStatus{DesiredNumberScheduled: 3, NumberReady: 3},
		},
		// kube-proxy ausente a propósito: debe omitirse sin error.
	)
	c := &k8sx.Clients{Kube: kube, Context: "test"}

	comps, err := Components(context.Background(), c)
	if err != nil {
		t.Fatalf("Components() error: %v", err)
	}
	if len(comps) != 2 {
		t.Fatalf("esperaba 2 componentes (coredns, aws-node), obtuve %d", len(comps))
	}
	for _, comp := range comps {
		switch comp.Name {
		case "coredns":
			if comp.Healthy {
				t.Errorf("coredns con 1/2 réplicas no debería ser Healthy")
			}
		case "aws-node":
			if !comp.Healthy {
				t.Errorf("aws-node con 3/3 debería ser Healthy")
			}
		}
	}
}

func TestDatadogCoverage_PartialAndAbsent(t *testing.T) {
	// Con Datadog presente cubriendo 2 de 3 nodos.
	kube := kfake.NewSimpleClientset(
		newNode("n1", true, true, "2", "4Gi"),
		newNode("n2", true, true, "2", "4Gi"),
		newNode("n3", true, true, "2", "4Gi"),
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{Name: "datadog-agent", Namespace: "monitoring"},
			Status:     appsv1.DaemonSetStatus{DesiredNumberScheduled: 3, NumberReady: 2},
		},
	)
	c := &k8sx.Clients{Kube: kube, Context: "test"}

	cov, err := DatadogAgentCoverage(context.Background(), c)
	if err != nil {
		t.Fatalf("DatadogAgentCoverage() error: %v", err)
	}
	if !cov.Installed {
		t.Fatalf("esperaba Datadog instalado")
	}
	if cov.CoveredNodes != 2 || cov.TotalNodes != 3 {
		t.Errorf("cobertura esperada 2/3, obtuve %d/%d", cov.CoveredNodes, cov.TotalNodes)
	}

	// Sin Datadog.
	kube2 := kfake.NewSimpleClientset(newNode("n1", true, true, "2", "4Gi"))
	cov2, err := DatadogAgentCoverage(context.Background(), &k8sx.Clients{Kube: kube2, Context: "test"})
	if err != nil {
		t.Fatalf("DatadogAgentCoverage() sin agente error: %v", err)
	}
	if cov2.Installed {
		t.Errorf("no debería reportar Datadog instalado")
	}
}

func TestRecommendations_DerivesFindings(t *testing.T) {
	r := &model.Report{
		Context:          "test",
		MetricsAvailable: true,
		Nodes: []model.Node{
			{Name: "bad", Ready: false, Schedulable: true},
			{Name: "good", Ready: true, Schedulable: true},
		},
		Namespaces: []model.NamespaceUsage{{
			Namespace: "over",
			Requests:  model.Resource{CPUMilli: 1000},
			Usage:     &model.Resource{CPUMilli: 100},
		}},
		Restarts: []model.PodRestart{{Name: "flaky", Namespace: "default", Restarts: 8, Reason: "OOMKilled"}},
		Components: []model.ComponentStatus{
			{Name: "coredns", Namespace: "kube-system", Kind: "Deployment", Desired: 2, Ready: 1, Healthy: false},
		},
		Datadog:   &model.DatadogCoverage{Installed: true, CoveredNodes: 1, TotalNodes: 3},
		Karpenter: &model.KarpenterStatus{Installed: true, NodePools: 0},
	}

	findings := Recommendations(r)
	if len(findings) == 0 {
		t.Fatal("esperaba hallazgos, obtuve 0")
	}

	var hasCriticalNode, hasOverprovision bool
	for _, f := range findings {
		if f.Severity == model.SeverityCritical && contains(f.Title, "bad") {
			hasCriticalNode = true
		}
		if contains(f.Title, "sobredimensionado") {
			hasOverprovision = true
		}
	}
	if !hasCriticalNode {
		t.Error("esperaba hallazgo crítico por nodo no Ready")
	}
	if !hasOverprovision {
		t.Error("esperaba hallazgo de sobredimensionamiento de CPU")
	}
}

func contains(s, sub string) bool { return containsFold(s, sub) }
