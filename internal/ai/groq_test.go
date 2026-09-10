package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestGroq construye un Groq apuntando a un servidor de prueba, sin dormir.
func newTestGroq(url string) *Groq {
	return &Groq{
		apiKey:     "test-key",
		model:      "test-model",
		httpClient: &http.Client{Timeout: 5 * time.Second},
		baseURL:    url,
		sleep:      func(time.Duration) {}, // no dormir en tests
	}
}

func TestGroq_RetriesOn429ThenSucceeds(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"hola"}}]}`))
	}))
	defer srv.Close()

	g := newTestGroq(srv.URL)
	resp, err := g.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil)
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("esperaba 2 intentos (1 tras 429), hubo %d", attempts)
	}
	if resp.Content != "hola" {
		t.Errorf("contenido esperado 'hola', obtuve %q", resp.Content)
	}
}

func TestGroq_ParsesToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"","tool_calls":[
			{"id":"call_1","type":"function","function":{"name":"eks_report","arguments":"{}"}}
		]}}]}`))
	}))
	defer srv.Close()

	g := newTestGroq(srv.URL)
	resp, err := g.Chat(context.Background(), []Message{{Role: RoleUser, Content: "diagnostica"}}, nil)
	if err != nil {
		t.Fatalf("Chat error: %v", err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "eks_report" {
		t.Fatalf("esperaba una tool call a eks_report, obtuve %+v", resp.ToolCalls)
	}
	if resp.ToolCalls[0].ID != "call_1" {
		t.Errorf("ID esperado call_1, obtuve %q", resp.ToolCalls[0].ID)
	}
}

func TestGroq_GivesUpAfterMaxRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	g := newTestGroq(srv.URL)
	_, err := g.Chat(context.Background(), []Message{{Role: RoleUser, Content: "hi"}}, nil)
	if err == nil {
		t.Fatal("esperaba error tras agotar los reintentos por 429")
	}
}
