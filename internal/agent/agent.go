// Package agent implementa el loop de tool-use: dada una pregunta en lenguaje
// natural, invoca las tools core iterativamente vía el Provider hasta producir
// hallazgos estructurados (R5.1, R5.3). Nunca ejecuta operaciones de escritura
// (R5.4): solo usa las tools read-only del registro.
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arheanja-ops/kubefin/internal/ai"
	"github.com/arheanja-ops/kubefin/internal/model"
	"github.com/arheanja-ops/kubefin/internal/toolset"
)

// maxSteps limita las iteraciones de tool-use para acotar costo y latencia.
const maxSteps = 8

const systemPrompt = `Eres kubefin, un asistente de diagnóstico de EKS y FinOps de AWS.
Respondes preguntas usando EXCLUSIVAMENTE las tools disponibles (son read-only).
Nunca inventes datos: si necesitas información, llama a una tool.
Cuando tengas suficiente evidencia, responde con un JSON que cumpla exactamente este esquema:
{"findings":[{"severity":"info|warning|critical","title":"...","evidence":"...","recommendation":"...","command":"...","estimatedSaving":{"amount":0,"currency":"USD"}}]}
"recommendation", "command" y "estimatedSaving" son opcionales. No incluyas texto fuera del JSON.`

// Agent orquesta el Provider y el registro de tools.
type Agent struct {
	provider ai.Provider
	registry *toolset.Registry
	redactor *ai.Redactor
	deps     toolset.Deps
}

// New construye un agente.
func New(provider ai.Provider, registry *toolset.Registry, redactor *ai.Redactor, deps toolset.Deps) *Agent {
	return &Agent{provider: provider, registry: registry, redactor: redactor, deps: deps}
}

// Ask ejecuta el loop de tool-use para responder la pregunta y devuelve los
// hallazgos estructurados.
func (a *Agent) Ask(ctx context.Context, question string) ([]model.Finding, error) {
	messages := []ai.Message{
		{Role: ai.RoleSystem, Content: systemPrompt},
		{Role: ai.RoleUser, Content: question},
	}
	specs := a.registry.Specs()

	for step := 0; step < maxSteps; step++ {
		resp, err := a.provider.Chat(ctx, a.redactor.Messages(messages), specs)
		if err != nil {
			return nil, err
		}

		if len(resp.ToolCalls) == 0 {
			return parseFindings(resp.Content)
		}

		messages = append(messages, assistantMessage(resp))
		for _, call := range resp.ToolCalls {
			messages = append(messages, a.runToolCall(ctx, call))
		}
	}
	return nil, fmt.Errorf("el agente no convergió en %d pasos", maxSteps)
}

// runToolCall ejecuta una tool solicitada por el modelo y devuelve el mensaje
// de resultado (o de error, sin abortar el loop).
func (a *Agent) runToolCall(ctx context.Context, call ai.ToolCall) ai.Message {
	tool, ok := a.registry.Get(call.Name)
	if !ok {
		return toolResult(call.ID, fmt.Sprintf("error: tool desconocida %q", call.Name))
	}
	out, err := tool.Handler(ctx, a.deps, json.RawMessage(call.Arguments))
	if err != nil {
		return toolResult(call.ID, "error: "+err.Error())
	}
	data, err := json.Marshal(out)
	if err != nil {
		return toolResult(call.ID, "error: no se pudo serializar el resultado: "+err.Error())
	}
	return toolResult(call.ID, string(data))
}

func assistantMessage(resp *ai.Response) ai.Message {
	return ai.Message{Role: ai.RoleAssistant, Content: resp.Content, ToolCalls: resp.ToolCalls}
}

func toolResult(id, content string) ai.Message {
	return ai.Message{Role: ai.RoleTool, ToolCallID: id, Content: content}
}

// findingsEnvelope es el JSON que el modelo debe producir.
type findingsEnvelope struct {
	Findings []model.Finding `json:"findings"`
}

// parseFindings extrae los hallazgos del contenido del modelo, tolerando texto
// alrededor del bloque JSON.
func parseFindings(content string) ([]model.Finding, error) {
	jsonStr, err := extractJSON(content)
	if err != nil {
		return nil, err
	}
	var env findingsEnvelope
	if err := json.Unmarshal([]byte(jsonStr), &env); err != nil {
		return nil, fmt.Errorf("la respuesta del modelo no es un JSON de findings válido: %w", err)
	}
	return env.Findings, nil
}

// extractJSON devuelve el primer objeto JSON balanceado dentro de s.
func extractJSON(s string) (string, error) {
	start := -1
	depth := 0
	for i, r := range s {
		switch r {
		case '{':
			if depth == 0 {
				start = i
			}
			depth++
		case '}':
			depth--
			if depth == 0 && start >= 0 {
				return s[start : i+1], nil
			}
		}
	}
	return "", fmt.Errorf("no se encontró un objeto JSON en la respuesta del modelo")
}
