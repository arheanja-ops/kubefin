package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	// groqBaseURL es el endpoint OpenAI-compatible de Groq.
	groqBaseURL = "https://api.groq.com/openai/v1/chat/completions"
	// defaultModel es el modelo por defecto (free tier), configurable por env.
	defaultModel = "llama-3.3-70b-versatile"
	// maxRetries es el número de reintentos ante 429 (R6.4).
	maxRetries = 4
)

// ErrNoAPIKey indica que falta GROQ_API_KEY (R5.2).
var ErrNoAPIKey = errors.New(
	"GROQ_API_KEY no está definida: obtén una gratis en https://console.groq.com y expórtala " +
		"(o ponla en .env.local)")

// Groq implementa Provider sobre la API OpenAI-compatible de Groq.
type Groq struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
	// sleep permite inyectar el retardo en tests; por defecto time.Sleep.
	sleep func(time.Duration)
}

// NewGroq construye el provider leyendo GROQ_API_KEY del entorno y, si existe,
// GROQ_MODEL. Falla con guía si la key falta (R5.2).
func NewGroq() (*Groq, error) {
	key := os.Getenv("GROQ_API_KEY")
	if key == "" {
		return nil, ErrNoAPIKey
	}
	model := os.Getenv("GROQ_MODEL")
	if model == "" {
		model = defaultModel
	}
	return &Groq{
		apiKey:     key,
		model:      model,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		baseURL:    groqBaseURL,
		sleep:      time.Sleep,
	}, nil
}

// Chat implementa Provider. Reintenta ante 429 respetando Retry-After con
// backoff exponencial de reserva (R6.4).
func (g *Groq) Chat(ctx context.Context, messages []Message, tools []ToolSpec) (*Response, error) {
	body, err := json.Marshal(buildRequest(g.model, messages, tools))
	if err != nil {
		return nil, fmt.Errorf("no se pudo serializar la petición: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, rErr := g.doRequest(ctx, body)
		if rErr != nil {
			return nil, rErr
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			if attempt == maxRetries {
				lastErr = fmt.Errorf("Groq devolvió 429 tras %d reintentos (límite de rate del free tier)", maxRetries)
				break
			}
			wait := retryAfter(resp.Header, attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			g.sleep(wait)
			continue
		}

		return parseResponse(resp)
	}
	return nil, lastErr
}

func (g *Groq) doRequest(ctx context.Context, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("no se pudo construir la petición HTTP: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("la petición a Groq falló: %w", err)
	}
	return resp, nil
}

// retryAfter calcula la espera ante un 429: respeta el header Retry-After si
// está presente, o usa backoff exponencial (1s, 2s, 4s, ...).
func retryAfter(h http.Header, attempt int) time.Duration {
	if v := h.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(1<<attempt) * time.Second
}

// parseResponse decodifica la respuesta de la API y la mapea a Response.
func parseResponse(resp *http.Response) (*Response, error) {
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer la respuesta: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Groq devolvió %d: %s", resp.StatusCode, string(data))
	}

	var parsed chatResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, fmt.Errorf("respuesta de Groq no parseable: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, errors.New("Groq devolvió una respuesta sin choices")
	}

	msg := parsed.Choices[0].Message
	out := &Response{Content: msg.Content}
	for _, tc := range msg.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return out, nil
}
