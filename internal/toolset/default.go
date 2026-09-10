package toolset

// Default construye el registro con todas las tools core (eks, cost, optimize).
// Es la fuente única de verdad de las capacidades expuestas por MCP y el agente.
func Default() *Registry {
	r := New()
	registerEKS(r)
	registerCost(r)
	registerOptimize(r)
	return r
}
