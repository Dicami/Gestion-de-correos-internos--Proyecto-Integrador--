package client

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"maizap/internal/protocol"
)

var errQuit = errors.New("salir")

type menuOption struct {
	key    string
	label  string
	action func() error
}

type Console struct {
	api  *APIClient
	in   *bufio.Reader
	out  io.Writer
	user *protocol.UserDTO
}

func NewConsole(api *APIClient, in io.Reader, out io.Writer) *Console {
	return &Console{api: api, in: bufio.NewReader(in), out: out}
}

func (c *Console) Run() error {
	c.printf("\n=== MaiZap · Correo interno ===\n")
	c.checkServer()
	for {
		options := c.guestMenu()
		title := "Menú principal"
		if c.user != nil {
			options = c.userMenu()
			title = fmt.Sprintf("Sesión: %s <%s> · Sede %s", c.user.Name, c.user.Email, c.user.Site)
		}
		err := c.runMenu(title, options)
		if errors.Is(err, errQuit) || errors.Is(err, io.EOF) {
			if c.user != nil && c.api.HasSession() {
				_ = c.api.Logout()
			}
			c.printf("\nHasta pronto.\n")
			return nil
		}
		if err != nil {
			c.showError(err)
		}
		if c.user != nil && !c.api.HasSession() {
			c.user = nil
			c.printf("Su sesión terminó. Identifíquese nuevamente.\n")
		}
	}
}

func (c *Console) guestMenu() []menuOption {
	return []menuOption{
		{"1", "Registrar usuario", c.register},
		{"2", "Identificarse", c.login},
		{"3", "Probar conexión con el servidor", func() error { c.checkServer(); return nil }},
		{"0", "Salir", func() error { return errQuit }},
	}
}

func (c *Console) userMenu() []menuOption {
	return []menuOption{
		{"1", "Redactar y enviar mensaje", c.compose},
		{"2", "Bandeja de entrada", func() error { return c.listFolder("inbox") }},
		{"3", "Mensajes enviados", func() error { return c.listFolder("sent") }},
		{"4", "Leer mensaje", c.read},
		{"5", "Eliminar mensaje", c.remove},
		{"6", "Directorio de usuarios", c.directory},
		{"7", "Desactivar usuario (solo administrador)", c.deactivate},
		{"8", "Cerrar sesión", c.logout},
		{"0", "Salir", func() error { return errQuit }},
	}
}

func (c *Console) runMenu(title string, options []menuOption) error {
	c.printf("\n--- %s ---\n", title)
	for _, o := range options {
		c.printf("  %s) %s\n", o.key, o.label)
	}
	choice, err := c.ask("Seleccione una opción")
	if err != nil {
		return err
	}
	for _, o := range options {
		if o.key == choice {
			return o.action()
		}
	}
	if choice == "" {
		return fmt.Errorf("debe escribir el número de una opción")
	}
	return fmt.Errorf("la opción %q no existe", choice)
}

func (c *Console) checkServer() {
	health, err := c.api.Health()
	if err != nil {
		c.showError(err)
		return
	}
	c.printf("Conectado a %s (estado: %s)\n", c.api.Address(), health.Status)
}

func (c *Console) register() error {
	name, err := c.askRequired("Nombre completo")
	if err != nil {
		return err
	}
	email, err := c.askRequired("Correo corporativo (@maizap.com)")
	if err != nil {
		return err
	}
	site, err := c.askRequired("Sede (A o B)")
	if err != nil {
		return err
	}
	area, err := c.askRequired("Área o departamento")
	if err != nil {
		return err
	}
	user, err := c.api.Register(protocol.RegisterRequest{Name: name, Email: email, Site: site, Area: area})
	if err != nil {
		return err
	}
	c.printf("Usuario registrado: %s <%s> (%s)\n", user.Name, user.Email, user.ID)
	return nil
}

func (c *Console) login() error {
	email, err := c.askRequired("Correo corporativo")
	if err != nil {
		return err
	}
	resp, err := c.api.Login(email)
	if err != nil {
		return err
	}
	c.user = &resp.User
	c.printf("Bienvenido/a, %s. Rol: %s. La sesión expira tras inactividad (%s).\n",
		resp.User.Name, resp.User.Role, formatTime(resp.ExpiresAt))
	return nil
}

func (c *Console) logout() error {
	err := c.api.Logout()
	c.user = nil
	if err != nil {
		return err
	}
	c.printf("Sesión cerrada correctamente.\n")
	return nil
}

func (c *Console) compose() error {
	to, err := c.ask("Para (correos separados por coma)")
	if err != nil {
		return err
	}
	subject, err := c.ask("Asunto")
	if err != nil {
		return err
	}
	c.printf("Cuerpo del mensaje (escriba una línea con solo un punto '.' para terminar):\n")
	body, err := c.readMultiline()
	if err != nil {
		return err
	}
	msg, err := c.api.Send(protocol.SendRequest{To: splitList(to), Subject: subject, Body: body})
	if err != nil {
		return err
	}
	c.printf("Mensaje %s enviado a %s.\n", msg.ID, joinRefs(msg.To))
	return nil
}

func (c *Console) listFolder(folder string) error {
	box, err := c.api.Mailbox(folder)
	if err != nil {
		return err
	}
	label := "Bandeja de entrada"
	if folder == "sent" {
		label = "Mensajes enviados"
	}
	c.printf("\n%s: %d mensaje(s)", label, box.Total)
	if folder == "inbox" {
		c.printf(", %d sin leer", box.Unread)
	}
	c.printf("\n")
	if box.Total == 0 {
		c.printf("  (vacía)\n")
		return nil
	}
	for _, m := range box.Messages {
		mark := " "
		if !m.Read {
			mark = "*"
		}
		who := "De: " + m.From.Email
		if folder == "sent" {
			who = "Para: " + joinRefs(m.To)
		}
		c.printf(" %s %s  %s  %-32s  %s\n", mark, m.ID, formatTime(m.SentAt), truncate(who, 32), truncate(m.Subject, 40))
	}
	if folder == "inbox" && box.Unread > 0 {
		c.printf("  (* = no leído)\n")
	}
	return nil
}

func (c *Console) read() error {
	id, err := c.askRequired("ID del mensaje (ej. MSG-000001)")
	if err != nil {
		return err
	}
	msg, err := c.api.Message(id)
	if err != nil {
		return err
	}
	c.printf("\n----------------------------------------\n")
	c.printf("ID:      %s\n", msg.ID)
	c.printf("De:      %s <%s>\n", msg.From.Name, msg.From.Email)
	c.printf("Para:    %s\n", joinRefs(msg.To))
	c.printf("Fecha:   %s\n", formatTime(msg.SentAt))
	c.printf("Asunto:  %s\n", msg.Subject)
	c.printf("----------------------------------------\n%s\n", msg.Body)
	c.printf("----------------------------------------\n")
	return nil
}

func (c *Console) remove() error {
	id, err := c.askRequired("ID del mensaje a eliminar")
	if err != nil {
		return err
	}
	confirm, err := c.ask(fmt.Sprintf("¿Eliminar %s de su buzón? (s/n)", id))
	if err != nil {
		return err
	}
	if !strings.EqualFold(confirm, "s") {
		c.printf("Operación cancelada.\n")
		return nil
	}
	if err := c.api.DeleteMessage(id); err != nil {
		return err
	}
	c.printf("Mensaje %s eliminado de su buzón.\n", strings.ToUpper(id))
	return nil
}

func (c *Console) directory() error {
	users, err := c.api.Users()
	if err != nil {
		return err
	}
	c.printf("\n%-9s %-26s %-28s %-5s %-16s %s\n", "ID", "Nombre", "Correo", "Sede", "Área", "Estado")
	for _, u := range users {
		state := "activo"
		if !u.Active {
			state = "desactivado"
		}
		if u.Role == "administrador" {
			state += " (admin)"
		}
		c.printf("%-9s %-26s %-28s %-5s %-16s %s\n", u.ID, truncate(u.Name, 26), truncate(u.Email, 28), u.Site, truncate(u.Area, 16), state)
	}
	return nil
}

func (c *Console) deactivate() error {
	id, err := c.askRequired("ID del usuario a desactivar (ej. USR-0003)")
	if err != nil {
		return err
	}
	user, err := c.api.DeactivateUser(strings.ToUpper(id))
	if err != nil {
		return err
	}
	c.printf("Usuario %s <%s> desactivado.\n", user.Name, user.Email)
	return nil
}

func (c *Console) ask(label string) (string, error) {
	c.printf("%s: ", label)
	line, err := c.in.ReadString('\n')
	if err != nil && (line == "" || !errors.Is(err, io.EOF)) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (c *Console) askRequired(label string) (string, error) {
	value, err := c.ask(label)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("el campo %q es obligatorio", label)
	}
	return value, nil
}

func (c *Console) readMultiline() (string, error) {
	var lines []string
	for {
		line, err := c.in.ReadString('\n')
		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "." {
			break
		}
		if trimmed != "" || line != "" {
			lines = append(lines, trimmed)
		}
		if err != nil {
			if errors.Is(err, io.EOF) && len(lines) > 0 {
				break
			}
			return "", err
		}
	}
	return strings.Join(lines, "\n"), nil
}

func (c *Console) showError(err error) {
	var apiErr *APIError
	var connErr *ConnectionError
	switch {
	case errors.As(err, &connErr):
		c.printf("[Error de conexión] %s\n", connErr.Error())
	case errors.As(err, &apiErr):
		c.printf("[Error %d] %s\n", apiErr.Status, apiErr.Message)
		for _, d := range apiErr.Details {
			c.printf("   - %s\n", d)
		}
	default:
		c.printf("[Error] %s\n", err.Error())
	}
}

func (c *Console) printf(format string, args ...any) {
	fmt.Fprintf(c.out, format, args...)
}

func splitList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ';' })
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func joinRefs(refs []protocol.UserRef) string {
	emails := make([]string, 0, len(refs))
	for _, r := range refs {
		emails = append(emails, r.Email)
	}
	return strings.Join(emails, ", ")
}

func formatTime(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04")
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
