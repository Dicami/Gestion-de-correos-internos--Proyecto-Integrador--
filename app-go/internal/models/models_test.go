package models

import (
	"testing"
	"time"

	"maizap/internal/apperr"
)

func TestNewUserRejectsInvalidData(t *testing.T) {
	_, err := NewUser("USR-0001", "", "persona@gmail.com", Site("C"), "", RoleUser, time.Now())
	if !apperr.Is(err, apperr.Validation) {
		t.Fatalf("se esperaba error de validación, se obtuvo %v", err)
	}
	if got := len(apperr.From(err).Details()); got != 4 {
		t.Fatalf("se esperaban 4 detalles de validación, se obtuvieron %d", got)
	}
}

func TestNewUserNormalizesEmail(t *testing.T) {
	u, err := NewUser("USR-0001", "Ana Torres", "  Ana.Torres@MaiZap.com ", SiteA, "Compras", RoleUser, time.Now())
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if u.Email() != "ana.torres@maizap.com" {
		t.Fatalf("correo no normalizado: %s", u.Email())
	}
	if err := u.Deactivate(); err != nil {
		t.Fatalf("error al desactivar: %v", err)
	}
	if err := u.Deactivate(); !apperr.Is(err, apperr.Conflict) {
		t.Fatalf("se esperaba conflicto al desactivar dos veces, se obtuvo %v", err)
	}
}

func TestEnvelopeValidation(t *testing.T) {
	if _, err := NewEnvelope("USR-0001", nil, "", time.Now()); !apperr.Is(err, apperr.Validation) {
		t.Fatalf("se esperaba error de validación, se obtuvo %v", err)
	}
	if _, err := NewEnvelope("USR-0001", []string{"USR-0001"}, "Hola", time.Now()); !apperr.Is(err, apperr.Validation) {
		t.Fatalf("se esperaba rechazo del autoenvío, se obtuvo %v", err)
	}
	env, err := NewEnvelope("USR-0001", []string{"USR-0002", "USR-0002", " "}, "Hola", time.Now())
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(env.To()) != 1 || !env.Involves("USR-0002") || env.Involves("USR-0009") {
		t.Fatalf("destinatarios incorrectos: %v", env.To())
	}
}

func TestMailboxEntryLifecycle(t *testing.T) {
	now := time.Now()
	entry, err := NewMailboxEntry("USR-0002", "MSG-000001", FolderInbox, now)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if entry.IsRead() {
		t.Fatal("una entrada nueva de bandeja no debe estar leída")
	}
	if !entry.MarkRead(now) || entry.MarkRead(now) {
		t.Fatal("MarkRead debe cambiar el estado solo la primera vez")
	}
	if err := entry.Delete(now); err != nil {
		t.Fatalf("error al eliminar: %v", err)
	}
	if err := entry.Delete(now); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("se esperaba error al eliminar dos veces, se obtuvo %v", err)
	}
}
