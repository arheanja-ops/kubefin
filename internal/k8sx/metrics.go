package k8sx

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metricsapi "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Metrics abstrae el acceso a metrics.k8s.io que usan las tools. Se define como
// interfaz para poder mockearla en tests: el fake clientset oficial de metrics
// tiene un bug conocido (kubernetes/kubernetes#93853) por el que List devuelve
// vacío, así que los tests inyectan una implementación propia.
type Metrics interface {
	NodeMetrics(ctx context.Context) (*metricsapi.NodeMetricsList, error)
	PodMetrics(ctx context.Context, namespace string) (*metricsapi.PodMetricsList, error)
}

// realMetrics adapta el clientset oficial a la interfaz Metrics.
type realMetrics struct {
	client metricsv.Interface
}

func (m realMetrics) NodeMetrics(ctx context.Context) (*metricsapi.NodeMetricsList, error) {
	return m.client.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
}

func (m realMetrics) PodMetrics(ctx context.Context, namespace string) (*metricsapi.PodMetricsList, error) {
	return m.client.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
}
