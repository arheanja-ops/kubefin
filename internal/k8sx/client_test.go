package k8sx

import (
	"os"
	"path/filepath"
	"testing"
)

// writeKubeconfig crea un kubeconfig temporal con el contenido dado y devuelve su ruta.
func writeKubeconfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("no se pudo escribir kubeconfig: %v", err)
	}
	return path
}

func TestLoad_NoCurrentContext(t *testing.T) {
	// Kubeconfig sin current-context ni contextos: debe fallar con mensaje claro (R1.2).
	path := writeKubeconfig(t, "apiVersion: v1\nkind: Config\nclusters: []\ncontexts: []\nusers: []\n")

	_, err := Load(path, "")
	if err == nil {
		t.Fatal("esperaba error cuando no hay contexto activo")
	}
	if !contains(err.Error(), "contexto") {
		t.Errorf("el mensaje de error debería mencionar el contexto, obtuve: %v", err)
	}
}

func TestLoad_UnknownContext(t *testing.T) {
	path := writeKubeconfig(t, "apiVersion: v1\nkind: Config\nclusters: []\ncontexts: []\nusers: []\ncurrent-context: \"\"\n")

	_, err := Load(path, "no-existe")
	if err == nil {
		t.Fatal("esperaba error cuando el contexto pedido no existe")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
