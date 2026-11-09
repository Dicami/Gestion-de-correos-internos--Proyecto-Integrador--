package services

import (
	"testing"
	"time"

	"maizap/internal/apperr"
	"maizap/internal/models"
	"maizap/internal/storage"
)

type fixture struct {
	users *UserService
	mail  *MailService
	auth  *LocalAuthenticator
	clock *time.Time
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	current := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	now := func() time.Time { return current }
	store := storage.NewMemoryStore()
	users := NewUserService(store, now)
	return fixture{
		users: users,
		mail:  NewMailService(store, users, now),
		auth:  NewLocalAuthenticator(users, 30*time.Minute, now),
		clock: &current,
	}
}

func (f fixture) register(t *testing.T, name, email, site string) *models.User {
	t.Helper()
	u, err := f.users.Register(RegisterInput{Name: name, Email: email, Site: site, Area: "Operaciones"})
	if err != nil {
		t.Fatalf("no se pudo registrar %s: %v", email, err)
	}
	return u
}

func TestRegisterRejectsDuplicates(t *testing.T) {
	f := newFixture(t)
	f.register(t, "Ana Torres", "ana@maizap.com", "A")
	_, err := f.users.Register(RegisterInput{Name: "Ana Dos", Email: "ANA@maizap.com", Site: "B", Area: "Ventas"})
	if !apperr.Is(err, apperr.Conflict) {
		t.Fatalf("se esperaba conflicto, se obtuvo %v", err)
	}
}

func TestSendValidatesRecipients(t *testing.T) {
	f := newFixture(t)
	ana := f.register(t, "Ana Torres", "ana@maizap.com", "A")
	_, err := f.mail.Send(ana, ComposeInput{To: []string{"nadie@maizap.com", "externo@gmail.com"}, Subject: "", Body: ""})
	if !apperr.Is(err, apperr.Validation) {
		t.Fatalf("se esperaba error de validación, se obtuvo %v", err)
	}
	if got := len(apperr.From(err).Details()); got < 4 {
		t.Fatalf("se esperaban al menos 4 detalles, se obtuvieron %d: %v", got, apperr.From(err).Details())
	}
}

func TestMailFlow(t *testing.T) {
	f := newFixture(t)
	ana := f.register(t, "Ana Torres", "ana@maizap.com", "A")
	luis := f.register(t, "Luis Paredes", "luis@maizap.com", "B")
	eva := f.register(t, "Eva Mora", "eva@maizap.com", "B")

	sent, err := f.mail.Send(ana, ComposeInput{To: []string{"luis@maizap.com"}, Subject: "Pedido", Body: "¿Sale hoy?"})
	if err != nil {
		t.Fatalf("error al enviar: %v", err)
	}
	id := sent.Message.ID()

	inbox, err := f.mail.Inbox(luis)
	if err != nil || len(inbox) != 1 || inbox[0].Entry.IsRead() {
		t.Fatalf("bandeja de entrada inesperada: %v %v", inbox, err)
	}
	outbox, err := f.mail.Sent(ana)
	if err != nil || len(outbox) != 1 {
		t.Fatalf("enviados inesperados: %v %v", outbox, err)
	}

	read, err := f.mail.Read(luis, id)
	if err != nil || !read.Entry.IsRead() || read.Message.Body() != "¿Sale hoy?" {
		t.Fatalf("lectura inesperada: %v %v", read, err)
	}

	if _, err := f.mail.Read(eva, id); !apperr.Is(err, apperr.Forbidden) {
		t.Fatalf("se esperaba acceso prohibido, se obtuvo %v", err)
	}
	if _, err := f.mail.Read(luis, "MSG-999999"); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("se esperaba mensaje no encontrado, se obtuvo %v", err)
	}

	if err := f.mail.Delete(luis, id); err != nil {
		t.Fatalf("error al eliminar: %v", err)
	}
	if inbox, _ := f.mail.Inbox(luis); len(inbox) != 0 {
		t.Fatalf("el mensaje eliminado sigue en la bandeja: %d", len(inbox))
	}
	if outbox, _ := f.mail.Sent(ana); len(outbox) != 1 {
		t.Fatal("eliminar en el buzón del destinatario no debe afectar al remitente")
	}
	if err := f.mail.Delete(luis, id); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("se esperaba no encontrado al eliminar dos veces, se obtuvo %v", err)
	}
}

func TestSessionExpiresAndRevokes(t *testing.T) {
	f := newFixture(t)
	f.register(t, "Ana Torres", "ana@maizap.com", "A")
	session, _, err := f.auth.Authenticate("ana@maizap.com")
	if err != nil {
		t.Fatalf("error al identificar: %v", err)
	}
	if _, err := f.auth.Validate(session.Token()); err != nil {
		t.Fatalf("la sesión debería ser válida: %v", err)
	}
	*f.clock = f.clock.Add(31 * time.Minute)
	if _, err := f.auth.Validate(session.Token()); !apperr.Is(err, apperr.Unauthorized) {
		t.Fatalf("se esperaba sesión expirada, se obtuvo %v", err)
	}
	if _, _, err := f.auth.Authenticate("nadie@maizap.com"); !apperr.Is(err, apperr.NotFound) {
		t.Fatalf("se esperaba usuario no encontrado, se obtuvo %v", err)
	}
}

func TestDeactivateRequiresAdmin(t *testing.T) {
	f := newFixture(t)
	ana := f.register(t, "Ana Torres", "ana@maizap.com", "A")
	luis := f.register(t, "Luis Paredes", "luis@maizap.com", "B")
	if _, err := f.users.Deactivate(ana, luis.ID()); !apperr.Is(err, apperr.Forbidden) {
		t.Fatalf("se esperaba operación no autorizada, se obtuvo %v", err)
	}
}
