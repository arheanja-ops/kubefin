// Package toolset expone las tools core (eks, cost, optimize) como un registro
// uniforme, invocable desde el servidor MCP y desde el agente sin duplicar
// lógica (D3). Cada tool declara nombre, descripción y esquema de parámetros, y
// se ejecuta con un conjunto de dependencias (clientes) resuelto perezosamente.
package toolset

import (
	"context"
	"encoding/json"

	"github.com/arheanja-ops/kubefin/internal/ai"
)

// Deps agrupa las dependencias que las tools pueden necesitar. Se resuelven de
// forma perezosa: una tool de EKS no fuerza credenciales de AWS y viceversa.
type Deps struct {
	// Kubeconfig y KubeContext seleccionan el clúster (vacío = actual).
	Kubeconfig  string
	KubeContext string
	// Region es la región AWS (vacío = la del entorno).
	Region string
	// UseCEAPI habilita la Cost Explorer API (opt-in, con costo). Las tools que
	// puedan incurrir en costo deben respetarlo (R4.4).
	UseCEAPI bool
}

// Handler ejecuta una tool con argumentos JSON y devuelve un resultado
// serializable. El resultado se renderiza igual en MCP y en el agente.
type Handler func(ctx context.Context, deps Deps, args json.RawMessage) (any, error)

// Tool describe una capacidad atómica reutilizable.
type Tool struct {
	Name        string
	Description string
	// Parameters es el JSON Schema de los argumentos de la tool.
	Parameters map[string]any
	Handler    Handler
}

// Spec devuelve la especificación de la tool para el proveedor de IA.
func (t Tool) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        t.Name,
		Description: t.Description,
		Parameters:  t.Parameters,
	}
}

// Registry es el conjunto de tools disponibles, indexado por nombre.
type Registry struct {
	tools map[string]Tool
	order []string
}

// New construye un registro vacío.
func New() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

// Register añade una tool al registro.
func (r *Registry) Register(t Tool) {
	if _, exists := r.tools[t.Name]; !exists {
		r.order = append(r.order, t.Name)
	}
	r.tools[t.Name] = t
}

// Get devuelve una tool por nombre.
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List devuelve las tools en orden de registro.
func (r *Registry) List() []Tool {
	out := make([]Tool, 0, len(r.order))
	for _, name := range r.order {
		out = append(out, r.tools[name])
	}
	return out
}

// Specs devuelve las especificaciones de todas las tools para el proveedor de IA.
func (r *Registry) Specs() []ai.ToolSpec {
	specs := make([]ai.ToolSpec, 0, len(r.order))
	for _, name := range r.order {
		specs = append(specs, r.tools[name].Spec())
	}
	return specs
}
