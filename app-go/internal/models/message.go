package models

import (
	"strings"
	"time"
	"unicode/utf8"

	"maizap/internal/apperr"
)

const (
	MaxSubjectLength = 150
	MaxBodyLength    = 5000
	MaxRecipients    = 20
)

type Envelope struct {
	from    string
	to      []string
	subject string
	sentAt  time.Time
}

func NewEnvelope(from string, to []string, subject string, sentAt time.Time) (Envelope, error) {
	var problems []string
	from = strings.TrimSpace(from)
	subject = strings.TrimSpace(subject)
	recipients := uniqueNonEmpty(to)

	if from == "" {
		problems = append(problems, "el remitente es obligatorio")
	}
	if len(recipients) == 0 {
		problems = append(problems, "debe indicar al menos un destinatario")
	}
	if len(recipients) > MaxRecipients {
		problems = append(problems, "no se permiten más de 20 destinatarios")
	}
	for _, r := range recipients {
		if r == from {
			problems = append(problems, "no puede enviarse un mensaje a sí mismo")
			break
		}
	}
	if subject == "" {
		problems = append(problems, "el asunto es obligatorio")
	}
	if utf8.RuneCountInString(subject) > MaxSubjectLength {
		problems = append(problems, "el asunto no puede superar 150 caracteres")
	}
	if strings.ContainsAny(subject, "\r\n") {
		problems = append(problems, "el asunto debe ocupar una sola línea")
	}
	if sentAt.IsZero() {
		sentAt = time.Now()
	}
	if len(problems) > 0 {
		return Envelope{}, apperr.New(apperr.Validation, "datos del mensaje inválidos").WithDetails(problems...)
	}
	return Envelope{from: from, to: recipients, subject: subject, sentAt: sentAt}, nil
}

func (e Envelope) From() string      { return e.from }
func (e Envelope) Subject() string   { return e.subject }
func (e Envelope) SentAt() time.Time { return e.sentAt }

func (e Envelope) To() []string {
	out := make([]string, len(e.to))
	copy(out, e.to)
	return out
}

func (e Envelope) HasRecipient(userID string) bool {
	for _, r := range e.to {
		if r == userID {
			return true
		}
	}
	return false
}

func (e Envelope) Involves(userID string) bool {
	return e.from == userID || e.HasRecipient(userID)
}

type Message struct {
	Envelope
	id   string
	body string
}

func NewMessage(id string, envelope Envelope, body string) (*Message, error) {
	id = strings.TrimSpace(id)
	body = strings.TrimSpace(body)
	var problems []string
	if id == "" {
		problems = append(problems, "el identificador del mensaje es obligatorio")
	}
	if envelope.from == "" {
		problems = append(problems, "el mensaje requiere un sobre válido")
	}
	if body == "" {
		problems = append(problems, "el cuerpo del mensaje es obligatorio")
	}
	if utf8.RuneCountInString(body) > MaxBodyLength {
		problems = append(problems, "el cuerpo no puede superar 5000 caracteres")
	}
	if len(problems) > 0 {
		return nil, apperr.New(apperr.Validation, "datos del mensaje inválidos").WithDetails(problems...)
	}
	return &Message{Envelope: envelope, id: id, body: body}, nil
}

func (m *Message) ID() string   { return m.id }
func (m *Message) Body() string { return m.body }

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}
