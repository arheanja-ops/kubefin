package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/arheanja-ops/kubefin/internal/ai"
	"github.com/arheanja-ops/kubefin/internal/model"
	"github.com/arheanja-ops/kubefin/internal/toolset"
)

// mockProvider devuelve respuestas predefinidas en secuencia, registrando los
// mensajes recibidos para aserciones.
type mockProvider struct {
	responses []ai.Response
	calls     int
	seen      [][]ai.Message
}

func (m *mockProvider) Chat(_ context.Context, messages []ai.Message, _ []ai.ToolSpec) (*ai.Response, error) {
	m.seen = append(m.seen, messages)
	r := m.responses[m.calls]
	m.calls++
	return &r, nil
}

// buildRegistry crea un registro con una tool determinista para tests.
func buildRegistry(result any, err error) *toolset.Registry {
	r := toolset.New()
	r.Register(toolset.Tool{
		Name:        "test_tool",
		Description: "tool de prueba",
		Parameters:  map[string]any{"type": "object"},
		Handler: func(_ context.Context, _ toolset.Deps, _ json.RawMessage) (any, error) {
			return result, err
		},
	})
	return r
}

func TestAgent_ToolUseThenFindings(t *testing.T) {
	// Paso 1: el modelo pide llamar test_tool. Paso 2: devuelve findings.
	provider := &mockProvider{responses: []ai.Response{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "test_tool", Arguments: "{}"}}},
		{Content: `{"findings":[{"severity":"warning","title":"CPU alta","evidence":"90%"}]}`},
	}}
	reg := buildRegistry(map[string]any{"cpu": 90}, nil)
	ag := New(provider, reg, ai.NewRedactor(false), toolset.Deps{})

	findings, err := ag.Ask(context.Background(), "¿cómo está el clúster?")
	if err != nil {
		t.Fatalf("Ask error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("esperaba 1 finding, obtuve %d", len(findings))
	}
	if findings[0].Severity != model.SeverityWarning || findings[0].Title != "CPU alta" {
		t.Errorf("finding inesperado: %+v", findings[0])
	}
	if provider.calls != 2 {
		t.Errorf("esperaba 2 llamadas al provider, hubo %d", provider.calls)
	}

	// En la 2ª llamada, el resultado de la tool debe haberse añadido como
	// mensaje de rol tool.
	second := provider.seen[1]
	var foundToolMsg bool
	for _, m := range second {
		if m.Role == ai.RoleTool && m.ToolCallID == "c1" {
			foundToolMsg = true
		}
	}
	if !foundToolMsg {
		t.Error("el resultado de la tool debería inyectarse como mensaje rol=tool")
	}
}

func TestAgent_DirectAnswer(t *testing.T) {
	provider := &mockProvider{responses: []ai.Response{
		{Content: `Aquí tienes: {"findings":[]} listo`},
	}}
	ag := New(provider, toolset.New(), ai.NewRedactor(false), toolset.Deps{})

	findings, err := ag.Ask(context.Background(), "hola")
	if err != nil {
		t.Fatalf("Ask error: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("esperaba 0 findings, obtuve %d", len(findings))
	}
}

func TestAgent_ToolErrorDoesNotAbort(t *testing.T) {
	provider := &mockProvider{responses: []ai.Response{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "test_tool", Arguments: "{}"}}},
		{Content: `{"findings":[{"severity":"info","title":"ok","evidence":"tras error de tool"}]}`},
	}}
	reg := buildRegistry(nil, context.DeadlineExceeded)
	ag := New(provider, reg, ai.NewRedactor(false), toolset.Deps{})

	findings, err := ag.Ask(context.Background(), "diagnostica")
	if err != nil {
		t.Fatalf("un error de tool no debería abortar el loop: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("esperaba 1 finding tras recuperarse del error, obtuve %d", len(findings))
	}
}

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{`{"a":1}`, `{"a":1}`, true},
		{"prefijo {\"a\":{\"b\":2}} sufijo", `{"a":{"b":2}}`, true},
		{"sin json aquí", "", false},
	}
	for _, c := range cases {
		got, err := extractJSON(c.in)
		if c.ok && (err != nil || got != c.want) {
			t.Errorf("extractJSON(%q) = %q, %v; esperaba %q", c.in, got, err, c.want)
		}
		if !c.ok && err == nil {
			t.Errorf("extractJSON(%q) esperaba error", c.in)
		}
	}
}
