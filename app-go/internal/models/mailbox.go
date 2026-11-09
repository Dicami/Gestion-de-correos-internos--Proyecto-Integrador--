package models

import (
	"strings"
	"time"

	"maizap/internal/apperr"
)

type Folder string

const (
	FolderInbox Folder = "inbox"
	FolderSent  Folder = "sent"
)

func (f Folder) Valid() bool {
	return f == FolderInbox || f == FolderSent
}

func (f Folder) Label() string {
	if f == FolderSent {
		return "Enviados"
	}
	return "Bandeja de entrada"
}

type MailboxEntry struct {
	owner     string
	messageID string
	folder    Folder
	readAt    time.Time
	deletedAt time.Time
}

func NewMailboxEntry(owner, messageID string, folder Folder, createdAt time.Time) (*MailboxEntry, error) {
	owner = strings.TrimSpace(owner)
	messageID = strings.TrimSpace(messageID)
	if owner == "" || messageID == "" {
		return nil, apperr.New(apperr.Validation, "la entrada de buzón requiere propietario y mensaje")
	}
	if !folder.Valid() {
		return nil, apperr.New(apperr.Validation, "la carpeta indicada no existe")
	}
	entry := &MailboxEntry{owner: owner, messageID: messageID, folder: folder}
	if folder == FolderSent {
		entry.readAt = createdAt
	}
	return entry, nil
}

func RestoreMailboxEntry(owner, messageID string, folder Folder, readAt, deletedAt time.Time) (*MailboxEntry, error) {
	entry, err := NewMailboxEntry(owner, messageID, folder, readAt)
	if err != nil {
		return nil, err
	}
	entry.readAt = readAt
	entry.deletedAt = deletedAt
	return entry, nil
}

func (e *MailboxEntry) Owner() string        { return e.owner }
func (e *MailboxEntry) MessageID() string    { return e.messageID }
func (e *MailboxEntry) Folder() Folder       { return e.folder }
func (e *MailboxEntry) IsRead() bool         { return !e.readAt.IsZero() }
func (e *MailboxEntry) ReadAt() time.Time    { return e.readAt }
func (e *MailboxEntry) IsDeleted() bool      { return !e.deletedAt.IsZero() }
func (e *MailboxEntry) DeletedAt() time.Time { return e.deletedAt }

func (e *MailboxEntry) MarkRead(now time.Time) bool {
	if e.IsRead() {
		return false
	}
	e.readAt = now
	return true
}

func (e *MailboxEntry) Delete(now time.Time) error {
	if e.IsDeleted() {
		return apperr.New(apperr.NotFound, "el mensaje ya fue eliminado")
	}
	e.deletedAt = now
	return nil
}
