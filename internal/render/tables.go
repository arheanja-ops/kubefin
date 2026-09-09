package render

import (
	"fmt"
	"strconv"

	"github.com/arheanja-ops/kubefin/internal/model"
)

func boolStr(b bool) string {
	if b {
		return "sí"
	}
	return "no"
}

// mem formatea bytes como MiB con una cifra compacta.
func mem(bytes int64) string {
	if bytes == 0 {
		return "0"
	}
	return strconv.FormatInt(bytes/(1024*1024), 10) + "Mi"
}

func cpu(milli int64) string { return strconv.FormatInt(milli, 10) + "m" }

func usageCell(u *model.Resource) (string, string) {
	if u == nil {
		return "-", "-"
	}
	return cpu(u.CPUMilli), mem(u.MemoryBytes)
}

func nodesTable(nodes []model.Node) *table {
	t := newTable("NODO", "READY", "SCHED", "TIPO", "ZONA", "CPU alloc", "MEM alloc", "CPU uso", "MEM uso")
	for _, n := range nodes {
		uc, um := usageCell(n.Usage)
		t.add(n.Name, boolStr(n.Ready), boolStr(n.Schedulable), n.InstanceType, n.Zone,
			cpu(n.Allocatable.CPUMilli), mem(n.Allocatable.MemoryBytes), uc, um)
	}
	return t
}

func namespacesTable(nss []model.NamespaceUsage) *table {
	t := newTable("NAMESPACE", "PODS", "CPU req", "MEM req", "CPU uso", "MEM uso")
	for _, ns := range nss {
		uc, um := usageCell(ns.Usage)
		t.add(ns.Namespace, strconv.Itoa(ns.PodCount),
			cpu(ns.Requests.CPUMilli), mem(ns.Requests.MemoryBytes), uc, um)
	}
	return t
}

func componentsTable(cs []model.ComponentStatus) *table {
	t := newTable("COMPONENTE", "NAMESPACE", "KIND", "LISTAS", "SANO")
	for _, c := range cs {
		t.add(c.Name, c.Namespace, c.Kind,
			fmt.Sprintf("%d/%d", c.Ready, c.Desired), boolStr(c.Healthy))
	}
	return t
}

func restartsTable(rs []model.PodRestart) *table {
	t := newTable("POD", "NAMESPACE", "REINICIOS", "MOTIVO")
	for _, r := range rs {
		t.add(r.Name, r.Namespace, strconv.FormatInt(int64(r.Restarts), 10), r.Reason)
	}
	return t
}

func findingsTable(fs []model.Finding) *table {
	t := newTable("SEVERIDAD", "TÍTULO", "EVIDENCIA", "RECOMENDACIÓN")
	for _, f := range fs {
		t.add(string(f.Severity), f.Title, f.Evidence, f.Recommendation)
	}
	return t
}
