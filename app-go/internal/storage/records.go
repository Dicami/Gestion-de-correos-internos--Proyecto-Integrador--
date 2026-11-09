package storage

import (
	"time"

	"maizap/internal/models"
)

type UserRecord struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Site      string    `json:"site"`
	Area      string    `json:"area"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type MessageRecord struct {
	ID      string    `json:"id"`
	From    string    `json:"from"`
	To      []string  `json:"to"`
	Subject string    `json:"subject"`
	Body    string    `json:"body"`
	SentAt  time.Time `json:"sent_at"`
}

type EntryRecord struct {
	Owner     string     `json:"owner"`
	MessageID string     `json:"message_id"`
	Folder    string     `json:"folder"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type snapshot struct {
	NextUserSeq    int             `json:"next_user_seq"`
	NextMessageSeq int             `json:"next_message_seq"`
	Users          []UserRecord    `json:"users"`
	Messages       []MessageRecord `json:"messages"`
	Entries        []EntryRecord   `json:"entries"`
}

func userToRecord(u *models.User) UserRecord {
	return UserRecord{
		ID:        u.ID(),
		Name:      u.Name(),
		Email:     u.Email(),
		Site:      string(u.Site()),
		Area:      u.Area(),
		Role:      string(u.Role()),
		Active:    u.IsActive(),
		CreatedAt: u.CreatedAt(),
	}
}

func (r UserRecord) toModel() (*models.User, error) {
	u, err := models.NewUser(r.ID, r.Name, r.Email, models.Site(r.Site), r.Area, models.Role(r.Role), r.CreatedAt)
	if err != nil {
		return nil, err
	}
	if !r.Active {
		if err := u.Deactivate(); err != nil {
			return nil, err
		}
	}
	return u, nil
}

func messageToRecord(m *models.Message) MessageRecord {
	return MessageRecord{
		ID:      m.ID(),
		From:    m.From(),
		To:      m.To(),
		Subject: m.Subject(),
		Body:    m.Body(),
		SentAt:  m.SentAt(),
	}
}

func (r MessageRecord) toModel() (*models.Message, error) {
	envelope, err := models.NewEnvelope(r.From, r.To, r.Subject, r.SentAt)
	if err != nil {
		return nil, err
	}
	return models.NewMessage(r.ID, envelope, r.Body)
}

func entryToRecord(e *models.MailboxEntry) EntryRecord {
	record := EntryRecord{Owner: e.Owner(), MessageID: e.MessageID(), Folder: string(e.Folder())}
	if e.IsRead() {
		readAt := e.ReadAt()
		record.ReadAt = &readAt
	}
	if e.IsDeleted() {
		deletedAt := e.DeletedAt()
		record.DeletedAt = &deletedAt
	}
	return record
}

func (r EntryRecord) toModel() (*models.MailboxEntry, error) {
	var readAt, deletedAt time.Time
	if r.ReadAt != nil {
		readAt = *r.ReadAt
	}
	if r.DeletedAt != nil {
		deletedAt = *r.DeletedAt
	}
	return models.RestoreMailboxEntry(r.Owner, r.MessageID, models.Folder(r.Folder), readAt, deletedAt)
}
