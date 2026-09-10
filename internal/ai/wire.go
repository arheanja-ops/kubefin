package ai

// Tipos del payload OpenAI-compatible (chat completions con tool-use).

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []wireMessage `json:"messages"`
	Tools    []wireTool    `json:"tools,omitempty"`
}

type wireMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type wireTool struct {
	Type     string       `json:"type"`
	Function wireFunction `json:"function"`
}

type wireFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

type wireToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function wireToolCallFunc `json:"function"`
}

type wireToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content   string         `json:"content"`
			ToolCalls []wireToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

// buildRequest mapea los tipos del dominio al payload de la API.
func buildRequest(model string, messages []Message, tools []ToolSpec) chatRequest {
	req := chatRequest{Model: model}
	for _, m := range messages {
		wm := wireMessage{
			Role:       string(m.Role),
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		for _, tc := range m.ToolCalls {
			wm.ToolCalls = append(wm.ToolCalls, wireToolCall{
				ID:   tc.ID,
				Type: "function",
				Function: wireToolCallFunc{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
		req.Messages = append(req.Messages, wm)
	}
	for _, t := range tools {
		// wireFunction se construye explícitamente para mantenerlo desacoplado de
		// ai.ToolSpec (tipo de dominio): sus campos coinciden hoy, pero no deben
		// acoplarse por conversión.
		fn := wireFunction{ //nolint:staticcheck // wire type desacoplado del tipo de dominio a propósito
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		}
		req.Tools = append(req.Tools, wireTool{Type: "function", Function: fn})
	}
	return req
}
