// Package ai define la interfaz Provider para modelos de lenguaje y su
// implementación Groq (API OpenAI-compatible). El proveedor es intercambiable
// (D4): Groq es el default, pero cualquier backend OpenAI-compatible encaja.
package ai

import "context"

// Role identifica el emisor de un mensaje en la conversación.
type Role string

const (
	// RoleSystem es el mensaje de sistema (instrucciones).
	RoleSystem Role = "system"
	// RoleUser es un mensaje del usuario.
	RoleUser Role = "user"
	// RoleAssistant es un mensaje del modelo.
	RoleAssistant Role = "assistant"
	// RoleTool es el resultado de la ejecución de una tool.
	RoleTool Role = "tool"
)

// Message es un turno de la conversación. Para RoleTool, ToolCallID referencia
// la llamada que produjo el contenido. Para RoleAssistant, ToolCalls contiene
// las llamadas solicitadas por el modelo.
type Message struct {
	Role       Role
	Content    string
	ToolCalls  []ToolCall
	ToolCallID string
}

// ToolCall es una invocación de tool solicitada por el modelo.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON con los argumentos.
}

// ToolSpec describe una tool disponible para el modelo (nombre, descripción y
// esquema JSON de parámetros).
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema de los parámetros.
}

// Response es la respuesta del modelo a una llamada Chat.
type Response struct {
	Content   string
	ToolCalls []ToolCall
}

// Provider abstrae un modelo de lenguaje con soporte de tool-use. Es la única
// dependencia del agente hacia la IA, lo que permite mockearlo en tests (R5.6).
type Provider interface {
	// Chat envía la conversación y las tools disponibles y devuelve la respuesta
	// del modelo (texto y/o llamadas a tools).
	Chat(ctx context.Context, messages []Message, tools []ToolSpec) (*Response, error)
}
