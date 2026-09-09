package eks

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/arheanja-ops/kubefin/internal/k8sx"
	"github.com/arheanja-ops/kubefin/internal/model"
)

// Karpenter GVRs (API estable v1, karpenter.sh / karpenter.k8s.aws).
var (
	nodePoolGVR = schema.GroupVersionResource{
		Group: "karpenter.sh", Version: "v1", Resource: "nodepools",
	}
	nodeClaimGVR = schema.GroupVersionResource{
		Group: "karpenter.sh", Version: "v1", Resource: "nodeclaims",
	}
	ec2NodeClassGVR = schema.GroupVersionResource{
		Group: "karpenter.k8s.aws", Version: "v1", Resource: "ec2nodeclasses",
	}
)

// Karpenter detecta la presencia de Karpenter (por sus CRDs), cuenta NodePools y
// EC2NodeClasses, y reporta NodeClaims con condiciones no listas como issues.
// Si no hay cliente dinámico o los CRDs no existen, reporta Installed=false.
func Karpenter(ctx context.Context, c *k8sx.Clients) (*model.KarpenterStatus, error) {
	status := &model.KarpenterStatus{}
	if c.Dynamic == nil {
		return status, nil
	}

	nodePools, err := c.Dynamic.Resource(nodePoolGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		// CRD ausente: Karpenter no instalado. No es un error fatal.
		return status, nil
	}
	status.Installed = true
	status.NodePools = len(nodePools.Items)

	if classes, cErr := c.Dynamic.Resource(ec2NodeClassGVR).List(ctx, metav1.ListOptions{}); cErr == nil {
		status.EC2NodeClasses = len(classes.Items)
	}

	if claims, clErr := c.Dynamic.Resource(nodeClaimGVR).List(ctx, metav1.ListOptions{}); clErr == nil {
		for i := range claims.Items {
			if issue := unreadyNodeClaimIssue(&claims.Items[i]); issue != "" {
				status.Issues = append(status.Issues, issue)
			}
		}
	}

	return status, nil
}

// unreadyNodeClaimIssue devuelve una descripción si el NodeClaim tiene su
// condición Ready en un estado distinto de True.
func unreadyNodeClaimIssue(claim interface{ UnstructuredContent() map[string]any }) string {
	content := claim.UnstructuredContent()
	name, _ := nestedString(content, "metadata", "name")

	conditions, ok := nestedSlice(content, "status", "conditions")
	if !ok {
		return ""
	}
	for _, raw := range conditions {
		cond, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := cond["type"].(string); t == "Ready" {
			if s, _ := cond["status"].(string); s != "True" {
				reason, _ := cond["reason"].(string)
				return "NodeClaim " + name + " no está Ready (status=" + s + ", reason=" + reason + ")"
			}
		}
	}
	return ""
}

// containsFold indica si s contiene substr sin distinguir mayúsculas.
func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// nestedString navega un mapa unstructured y devuelve el string en la ruta dada.
func nestedString(obj map[string]any, path ...string) (string, bool) {
	v, ok := nestedField(obj, path...)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// nestedSlice navega un mapa unstructured y devuelve el slice en la ruta dada.
func nestedSlice(obj map[string]any, path ...string) ([]any, bool) {
	v, ok := nestedField(obj, path...)
	if !ok {
		return nil, false
	}
	s, ok := v.([]any)
	return s, ok
}

func nestedField(obj map[string]any, path ...string) (any, bool) {
	cur := any(obj)
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[key]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}
