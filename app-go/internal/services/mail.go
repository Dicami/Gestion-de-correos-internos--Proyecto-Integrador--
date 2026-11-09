package services

import (
	"sort"
	"strings"
	"time"

	"maizap/internal/apperr"
	"maizap/internal/models"
	"maizap/internal/storage"
)

type ComposeInput struct {
	To      []string
	Subject string
	Body    string
}

type MailView struct {
	Message    *models.Message
	Entry      *models.MailboxEntry
	Sender     *models.User
	Recipients []*models.User
}

type MailService struct {
	store *storage.JSONStore
	users *UserService
	now   func() time.Time
}

func NewMailService(store *storage.JSONStore, users *UserService, now func() time.Time) *MailService {
	if now == nil {
		now = time.Now
	}
	return &MailService{store: store, users: users, now: now}
}

func (s *MailService) Send(sender *models.User, in ComposeInput) (*MailView, error) {
	recipients, recipientErr := s.users.ResolveRecipients(in.To)
	recipientIDs := make([]string, 0, len(recipients))
	for _, r := range recipients {
		recipientIDs = append(recipientIDs, r.ID())
	}
	if recipientErr != nil {
		recipientIDs = []string{"pendiente"}
	}

	now := s.now()
	envelope, envErr := models.NewEnvelope(sender.ID(), recipientIDs, in.Subject, now)
	bodyErr := validateBody(in.Body)
	if err := combineValidation(recipientErr, envErr, bodyErr); err != nil {
		return nil, err
	}

	message, err := models.NewMessage(s.store.NextMessageID(), envelope, in.Body)
	if err != nil {
		return nil, err
	}
	sentEntry, err := models.NewMailboxEntry(sender.ID(), message.ID(), models.FolderSent, now)
	if err != nil {
		return nil, err
	}
	entries := []*models.MailboxEntry{sentEntry}
	for _, r := range recipients {
		inbox, err := models.NewMailboxEntry(r.ID(), message.ID(), models.FolderInbox, now)
		if err != nil {
			return nil, err
		}
		entries = append(entries, inbox)
	}
	if err := s.store.CreateMessage(message, entries); err != nil {
		return nil, err
	}
	return &MailView{Message: message, Entry: sentEntry, Sender: sender, Recipients: recipients}, nil
}

func (s *MailService) Inbox(user *models.User) ([]*MailView, error) {
	return s.folder(user, models.FolderInbox)
}

func (s *MailService) Sent(user *models.User) ([]*MailView, error) {
	return s.folder(user, models.FolderSent)
}

func (s *MailService) Read(user *models.User, messageID string) (*MailView, error) {
	entry, err := s.ownedEntry(user, messageID)
	if err != nil {
		return nil, err
	}
	if entry.Folder() == models.FolderInbox && entry.MarkRead(s.now()) {
		if err := s.store.UpdateEntry(entry); err != nil {
			return nil, err
		}
	}
	return s.view(entry)
}

func (s *MailService) Delete(user *models.User, messageID string) error {
	entry, err := s.ownedEntry(user, messageID)
	if err != nil {
		return err
	}
	if err := entry.Delete(s.now()); err != nil {
		return err
	}
	return s.store.UpdateEntry(entry)
}

func (s *MailService) folder(user *models.User, folder models.Folder) ([]*MailView, error) {
	entries, err := s.store.EntriesByOwner(user.ID(), folder)
	if err != nil {
		return nil, err
	}
	views := make([]*MailView, 0, len(entries))
	for _, e := range entries {
		if e.IsDeleted() {
			continue
		}
		v, err := s.view(e)
		if err != nil {
			return nil, err
		}
		views = append(views, v)
	}
	sort.Slice(views, func(i, j int) bool {
		return views[i].Message.SentAt().After(views[j].Message.SentAt())
	})
	return views, nil
}

func (s *MailService) ownedEntry(user *models.User, messageID string) (*models.MailboxEntry, error) {
	messageID = strings.ToUpper(strings.TrimSpace(messageID))
	if messageID == "" {
		return nil, apperr.New(apperr.Validation, "el identificador del mensaje es obligatorio")
	}
	message, err := s.store.MessageByID(messageID)
	if err != nil {
		return nil, err
	}
	if !message.Involves(user.ID()) {
		return nil, apperr.New(apperr.Forbidden, "no tiene permiso para acceder al mensaje "+messageID)
	}
	entry, err := s.store.Entry(user.ID(), messageID)
	if err != nil {
		return nil, err
	}
	if entry.IsDeleted() {
		return nil, apperr.New(apperr.NotFound, "el mensaje "+messageID+" fue eliminado de su buzón")
	}
	return entry, nil
}

func (s *MailService) view(entry *models.MailboxEntry) (*MailView, error) {
	message, err := s.store.MessageByID(entry.MessageID())
	if err != nil {
		return nil, err
	}
	sender, err := s.users.ByID(message.From())
	if err != nil {
		return nil, err
	}
	recipients := make([]*models.User, 0, len(message.To()))
	for _, id := range message.To() {
		r, err := s.users.ByID(id)
		if err != nil {
			return nil, err
		}
		recipients = append(recipients, r)
	}
	return &MailView{Message: message, Entry: entry, Sender: sender, Recipients: recipients}, nil
}

func validateBody(body string) error {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return apperr.New(apperr.Validation, "datos del mensaje inválidos").WithDetails("el cuerpo del mensaje es obligatorio")
	}
	if len([]rune(trimmed)) > models.MaxBodyLength {
		return apperr.New(apperr.Validation, "datos del mensaje inválidos").WithDetails("el cuerpo no puede superar 5000 caracteres")
	}
	return nil
}

func combineValidation(errs ...error) error {
	var details []string
	for _, err := range errs {
		if err == nil {
			continue
		}
		appErr := apperr.From(err)
		if appErr.Kind() != apperr.Validation {
			return err
		}
		if d := appErr.Details(); len(d) > 0 {
			details = append(details, d...)
		} else {
			details = append(details, appErr.Message())
		}
	}
	if len(details) == 0 {
		return nil
	}
	return apperr.New(apperr.Validation, "no se pudo enviar el mensaje").WithDetails(details...)
}
