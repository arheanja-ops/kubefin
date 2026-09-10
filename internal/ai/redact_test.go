package ai

import (
	"strings"
	"testing"
)

func TestRedactor_Disabled(t *testing.T) {
	r := NewRedactor(false)
	in := "cuenta 123456789012 arn:aws:iam::123456789012:role/x ip 10.0.0.5"
	if got := r.Apply(in); got != in {
		t.Errorf("con redacción deshabilitada no debería cambiar; obtuve %q", got)
	}
}

func TestRedactor_RedactsARNAccountIP(t *testing.T) {
	r := NewRedactor(true)
	in := "role arn:aws:iam::123456789012:role/admin en cuenta 210987654321, host 192.168.1.10"
	got := r.Apply(in)

	if strings.Contains(got, "123456789012") || strings.Contains(got, "210987654321") {
		t.Errorf("no debería quedar ningún account ID; obtuve %q", got)
	}
	if strings.Contains(got, "192.168.1.10") {
		t.Errorf("no debería quedar la IP; obtuve %q", got)
	}
	if !strings.Contains(got, "[IP]") || !strings.Contains(got, "[ACCOUNT_ID]") {
		t.Errorf("deberían aparecer los marcadores; obtuve %q", got)
	}
	if !strings.Contains(got, "arn:aws:REDACTED") {
		t.Errorf("el ARN debería redactarse; obtuve %q", got)
	}
}

func TestRedactor_Messages(t *testing.T) {
	r := NewRedactor(true)
	msgs := []Message{{Role: RoleUser, Content: "cuenta 123456789012"}}
	out := r.Messages(msgs)

	if strings.Contains(out[0].Content, "123456789012") {
		t.Errorf("el mensaje debería redactarse; obtuve %q", out[0].Content)
	}
	// El original no debe mutarse.
	if !strings.Contains(msgs[0].Content, "123456789012") {
		t.Error("el mensaje original no debería mutarse")
	}
}
