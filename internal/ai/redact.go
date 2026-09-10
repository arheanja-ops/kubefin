package ai

import "regexp"

var (
	// arnRe captura ARNs de AWS (arn:partition:service:region:account:resource).
	arnRe = regexp.MustCompile(`arn:aws[a-z-]*:[^:\s]*:[^:\s]*:\d{12}:[^\s"']+`)
	// accountRe captura IDs de cuenta AWS (12 dígitos) no adyacentes a otros dígitos.
	accountRe = regexp.MustCompile(`\b\d{12}\b`)
	// ipRe captura direcciones IPv4.
	ipRe = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
)

// Redactor reemplaza datos sensibles por marcadores antes de enviarlos a un
// proveedor remoto (R5.5). El orden importa: ARNs antes que account IDs, para
// no romper el ARN al redactar su cuenta.
type Redactor struct {
	enabled bool
}

// NewRedactor crea un redactor. Si enabled es false, Apply devuelve el texto sin
// cambios (redacción opt-in).
func NewRedactor(enabled bool) *Redactor {
	return &Redactor{enabled: enabled}
}

// Apply redacta un texto si el redactor está habilitado.
func (r *Redactor) Apply(s string) string {
	if !r.enabled {
		return s
	}
	s = arnRe.ReplaceAllString(s, "arn:aws:REDACTED")
	s = accountRe.ReplaceAllString(s, "[ACCOUNT_ID]")
	s = ipRe.ReplaceAllString(s, "[IP]")
	return s
}

// Messages aplica la redacción al contenido de cada mensaje, devolviendo una
// copia. Los mensajes originales no se modifican.
func (r *Redactor) Messages(messages []Message) []Message {
	if !r.enabled {
		return messages
	}
	out := make([]Message, len(messages))
	for i, m := range messages {
		m.Content = r.Apply(m.Content)
		out[i] = m
	}
	return out
}
