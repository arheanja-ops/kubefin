package toolset

import (
	"context"
	"encoding/json"
	"testing"
)

func noopHandler(context.Context, Deps, json.RawMessage) (any, error) { return nil, nil }

func TestRegistry_RegisterListOrder(t *testing.T) {
	r := New()
	r.Register(Tool{Name: "a", Handler: noopHandler})
	r.Register(Tool{Name: "b", Handler: noopHandler})
	r.Register(Tool{Name: "a", Description: "actualizada", Handler: noopHandler})

	list := r.List()
	if len(list) != 2 {
		t.Fatalf("esperaba 2 tools (a se actualiza, no duplica), obtuve %d", len(list))
	}
	if list[0].Name != "a" || list[1].Name != "b" {
		t.Errorf("orden de registro no preservado: %v", []string{list[0].Name, list[1].Name})
	}
	if list[0].Description != "actualizada" {
		t.Errorf("re-registrar debería actualizar la tool; desc=%q", list[0].Description)
	}
}

func TestRegistry_GetAndSpecs(t *testing.T) {
	r := New()
	r.Register(Tool{Name: "x", Description: "desc", Parameters: map[string]any{"type": "object"}, Handler: noopHandler})

	if _, ok := r.Get("x"); !ok {
		t.Fatal("Get('x') debería encontrar la tool")
	}
	if _, ok := r.Get("nope"); ok {
		t.Error("Get de tool inexistente debería devolver ok=false")
	}

	specs := r.Specs()
	if len(specs) != 1 || specs[0].Name != "x" || specs[0].Description != "desc" {
		t.Errorf("Specs mal formadas: %+v", specs)
	}
}

func TestDefault_RegistersCoreTools(t *testing.T) {
	r := Default()
	for _, name := range []string{"eks_report", "eks_nodes", "eks_namespaces", "cost_by_service", "optimize_compute"} {
		if _, ok := r.Get(name); !ok {
			t.Errorf("el registro por defecto debería incluir %q", name)
		}
	}
}
