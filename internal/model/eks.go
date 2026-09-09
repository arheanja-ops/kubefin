package model

// Resource captura una pareja CPU/memoria. CPU en millicores (m), memoria en
// bytes. Se usa tanto para capacidad/asignable de nodos como para
// requests/limits/uso de pods.
type Resource struct {
	CPUMilli    int64 `json:"cpuMilli"`
	MemoryBytes int64 `json:"memoryBytes"`
}

// Node describe el estado y capacidad de un nodo del clúster.
type Node struct {
	Name         string            `json:"name"`
	Ready        bool              `json:"ready"`
	Schedulable  bool              `json:"schedulable"`
	InstanceType string            `json:"instanceType,omitempty"`
	Zone         string            `json:"zone,omitempty"`
	Capacity     Resource          `json:"capacity"`
	Allocatable  Resource          `json:"allocatable"`
	Labels       map[string]string `json:"labels,omitempty"`
	// Usage es nil si Metrics Server no está disponible.
	Usage *Resource `json:"usage,omitempty"`
}

// Pod describe un pod y su consumo/solicitud de recursos.
type Pod struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Node      string    `json:"node,omitempty"`
	Phase     string    `json:"phase"`
	Restarts  int32     `json:"restarts"`
	Requests  Resource  `json:"requests"`
	Limits    Resource  `json:"limits"`
	Usage     *Resource `json:"usage,omitempty"`
}

// NamespaceUsage agrega solicitudes, límites y uso por namespace.
type NamespaceUsage struct {
	Namespace string    `json:"namespace"`
	PodCount  int       `json:"podCount"`
	Requests  Resource  `json:"requests"`
	Limits    Resource  `json:"limits"`
	Usage     *Resource `json:"usage,omitempty"`
}

// ResourceQuota refleja una ResourceQuota de un namespace: uso vs. límite duro.
type ResourceQuota struct {
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Hard      map[string]string `json:"hard,omitempty"`
	Used      map[string]string `json:"used,omitempty"`
}

// PodRestart resume un pod con reinicios, para el diagnóstico de estabilidad.
type PodRestart struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Restarts  int32  `json:"restarts"`
	Reason    string `json:"reason,omitempty"`
}

// ComponentStatus refleja el estado de un componente core del clúster
// (p. ej. un Deployment de kube-system como coredns).
type ComponentStatus struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"`
	Desired   int32  `json:"desired"`
	Ready     int32  `json:"ready"`
	Healthy   bool   `json:"healthy"`
}

// KarpenterStatus resume la presencia y salud de Karpenter en el clúster.
type KarpenterStatus struct {
	Installed      bool     `json:"installed"`
	NodePools      int      `json:"nodePools"`
	EC2NodeClasses int      `json:"ec2NodeClasses"`
	Issues         []string `json:"issues,omitempty"`
}

// DatadogCoverage resume la cobertura del Datadog Agent (DaemonSet) sobre los
// nodos del clúster.
type DatadogCoverage struct {
	Installed    bool  `json:"installed"`
	DesiredNodes int32 `json:"desiredNodes"`
	CoveredNodes int32 `json:"coveredNodes"`
	TotalNodes   int   `json:"totalNodes"`
}

// Report agrega el resultado completo de un diagnóstico EKS. Los campos de tipo
// slice/puntero nil indican secciones no recolectadas (p. ej. por falta de
// Metrics Server).
type Report struct {
	Context          string            `json:"context"`
	MetricsAvailable bool              `json:"metricsAvailable"`
	Nodes            []Node            `json:"nodes"`
	Namespaces       []NamespaceUsage  `json:"namespaces"`
	TopPods          []Pod             `json:"topPods,omitempty"`
	Quotas           []ResourceQuota   `json:"quotas,omitempty"`
	Restarts         []PodRestart      `json:"restarts,omitempty"`
	Components       []ComponentStatus `json:"components,omitempty"`
	Karpenter        *KarpenterStatus  `json:"karpenter,omitempty"`
	Datadog          *DatadogCoverage  `json:"datadog,omitempty"`
	Findings         []Finding         `json:"findings"`
	// Warnings recoge problemas no fatales durante la recolección
	// (p. ej. "metrics server no disponible").
	Warnings []string `json:"warnings,omitempty"`
}
