package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"maizap/internal/apperr"
	"maizap/internal/models"
)

type entryKey struct {
	owner     string
	messageID string
}

type JSONStore struct {
	mu         sync.RWMutex
	path       string
	userSeq    int
	messageSeq int
	users      map[string]UserRecord
	emailIndex map[string]string
	messages   map[string]MessageRecord
	entries    map[entryKey]EntryRecord
}

func NewMemoryStore() *JSONStore {
	return &JSONStore{
		users:      make(map[string]UserRecord),
		emailIndex: make(map[string]string),
		messages:   make(map[string]MessageRecord),
		entries:    make(map[entryKey]EntryRecord),
	}
}

func OpenJSONStore(path, seedPath string) (*JSONStore, error) {
	if path == "" {
		return nil, apperr.New(apperr.Validation, "la ruta del archivo de datos es obligatoria")
	}
	store := NewMemoryStore()
	store.path = path

	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := store.load(data, path); err != nil {
			return nil, err
		}
		return store, nil
	case !errors.Is(err, fs.ErrNotExist):
		return nil, apperr.Wrap(apperr.Storage, "no se pudo leer el archivo de datos "+path, err)
	}

	if seedPath != "" {
		seed, err := os.ReadFile(seedPath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, apperr.Wrap(apperr.Storage, "no se pudo leer el archivo semilla "+seedPath, err)
		}
		if err == nil {
			if err := store.load(seed, seedPath); err != nil {
				return nil, err
			}
		}
	}
	if err := store.persistLocked(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *JSONStore) Path() string {
	return s.path
}

func (s *JSONStore) load(data []byte, source string) error {
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return apperr.Wrap(apperr.Storage, "el archivo "+source+" no contiene JSON válido", err)
	}
	for _, r := range snap.Users {
		if _, err := r.toModel(); err != nil {
			return apperr.Wrap(apperr.Storage, fmt.Sprintf("usuario %q inválido en %s", r.ID, source), err)
		}
		if _, dup := s.emailIndex[r.Email]; dup {
			return apperr.New(apperr.Storage, fmt.Sprintf("correo duplicado %q en %s", r.Email, source))
		}
		s.users[r.ID] = r
		s.emailIndex[r.Email] = r.ID
	}
	for _, r := range snap.Messages {
		if _, err := r.toModel(); err != nil {
			return apperr.Wrap(apperr.Storage, fmt.Sprintf("mensaje %q inválido en %s", r.ID, source), err)
		}
		s.messages[r.ID] = r
	}
	for _, r := range snap.Entries {
		if _, err := r.toModel(); err != nil {
			return apperr.Wrap(apperr.Storage, fmt.Sprintf("entrada de buzón inválida en %s", source), err)
		}
		if _, ok := s.messages[r.MessageID]; !ok {
			return apperr.New(apperr.Storage, fmt.Sprintf("la entrada de buzón referencia el mensaje inexistente %q", r.MessageID))
		}
		s.entries[entryKey{r.Owner, r.MessageID}] = r
	}
	s.userSeq = max(snap.NextUserSeq, len(s.users))
	s.messageSeq = max(snap.NextMessageSeq, len(s.messages))
	return nil
}

func (s *JSONStore) persistLocked() error {
	if s.path == "" {
		return nil
	}
	snap := snapshot{
		NextUserSeq:    s.userSeq,
		NextMessageSeq: s.messageSeq,
		Users:          make([]UserRecord, 0, len(s.users)),
		Messages:       make([]MessageRecord, 0, len(s.messages)),
		Entries:        make([]EntryRecord, 0, len(s.entries)),
	}
	for _, r := range s.users {
		snap.Users = append(snap.Users, r)
	}
	for _, r := range s.messages {
		snap.Messages = append(snap.Messages, r)
	}
	for _, r := range s.entries {
		snap.Entries = append(snap.Entries, r)
	}
	sort.Slice(snap.Users, func(i, j int) bool { return snap.Users[i].ID < snap.Users[j].ID })
	sort.Slice(snap.Messages, func(i, j int) bool { return snap.Messages[i].ID < snap.Messages[j].ID })
	sort.Slice(snap.Entries, func(i, j int) bool {
		if snap.Entries[i].MessageID != snap.Entries[j].MessageID {
			return snap.Entries[i].MessageID < snap.Entries[j].MessageID
		}
		return snap.Entries[i].Owner < snap.Entries[j].Owner
	})

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return apperr.Wrap(apperr.Storage, "no se pudieron serializar los datos", err)
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return apperr.Wrap(apperr.Storage, "no se pudo crear el directorio de datos", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return apperr.Wrap(apperr.Storage, "no se pudo escribir el archivo de datos", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		return apperr.Wrap(apperr.Storage, "no se pudo reemplazar el archivo de datos", err)
	}
	return nil
}

func (s *JSONStore) NextUserID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.userSeq++
	return fmt.Sprintf("USR-%04d", s.userSeq)
}

func (s *JSONStore) NextMessageID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageSeq++
	return fmt.Sprintf("MSG-%06d", s.messageSeq)
}

func (s *JSONStore) CreateUser(u *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.emailIndex[u.Email()]; exists {
		return apperr.New(apperr.Conflict, "ya existe un usuario registrado con el correo "+u.Email())
	}
	if _, exists := s.users[u.ID()]; exists {
		return apperr.New(apperr.Conflict, "ya existe un usuario con el identificador "+u.ID())
	}
	s.users[u.ID()] = userToRecord(u)
	s.emailIndex[u.Email()] = u.ID()
	if err := s.persistLocked(); err != nil {
		delete(s.users, u.ID())
		delete(s.emailIndex, u.Email())
		return err
	}
	return nil
}

func (s *JSONStore) UpdateUser(u *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	previous, exists := s.users[u.ID()]
	if !exists {
		return apperr.New(apperr.NotFound, "no existe el usuario "+u.ID())
	}
	s.users[u.ID()] = userToRecord(u)
	if err := s.persistLocked(); err != nil {
		s.users[u.ID()] = previous
		return err
	}
	return nil
}

func (s *JSONStore) UserByID(id string) (*models.User, error) {
	s.mu.RLock()
	record, ok := s.users[id]
	s.mu.RUnlock()
	if !ok {
		return nil, apperr.New(apperr.NotFound, "no existe el usuario "+id)
	}
	return record.toModel()
}

func (s *JSONStore) UserByEmail(email string) (*models.User, error) {
	s.mu.RLock()
	id, ok := s.emailIndex[email]
	record := s.users[id]
	s.mu.RUnlock()
	if !ok {
		return nil, apperr.New(apperr.NotFound, "no existe un usuario con el correo "+email)
	}
	return record.toModel()
}

func (s *JSONStore) ListUsers() ([]*models.User, error) {
	s.mu.RLock()
	records := make([]UserRecord, 0, len(s.users))
	for _, r := range s.users {
		records = append(records, r)
	}
	s.mu.RUnlock()
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	users := make([]*models.User, 0, len(records))
	for _, r := range records {
		u, err := r.toModel()
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *JSONStore) CreateMessage(m *models.Message, entries []*models.MailboxEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.messages[m.ID()]; exists {
		return apperr.New(apperr.Conflict, "ya existe el mensaje "+m.ID())
	}
	s.messages[m.ID()] = messageToRecord(m)
	keys := make([]entryKey, 0, len(entries))
	for _, e := range entries {
		key := entryKey{e.Owner(), e.MessageID()}
		s.entries[key] = entryToRecord(e)
		keys = append(keys, key)
	}
	if err := s.persistLocked(); err != nil {
		delete(s.messages, m.ID())
		for _, key := range keys {
			delete(s.entries, key)
		}
		return err
	}
	return nil
}

func (s *JSONStore) MessageByID(id string) (*models.Message, error) {
	s.mu.RLock()
	record, ok := s.messages[id]
	s.mu.RUnlock()
	if !ok {
		return nil, apperr.New(apperr.NotFound, "no existe el mensaje "+id)
	}
	return record.toModel()
}

func (s *JSONStore) Entry(owner, messageID string) (*models.MailboxEntry, error) {
	s.mu.RLock()
	record, ok := s.entries[entryKey{owner, messageID}]
	s.mu.RUnlock()
	if !ok {
		return nil, apperr.New(apperr.NotFound, "el mensaje "+messageID+" no está en su buzón")
	}
	return record.toModel()
}

func (s *JSONStore) EntriesByOwner(owner string, folder models.Folder) ([]*models.MailboxEntry, error) {
	s.mu.RLock()
	records := make([]EntryRecord, 0)
	for key, r := range s.entries {
		if key.owner == owner && r.Folder == string(folder) {
			records = append(records, r)
		}
	}
	s.mu.RUnlock()
	entries := make([]*models.MailboxEntry, 0, len(records))
	for _, r := range records {
		e, err := r.toModel()
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *JSONStore) UpdateEntry(e *models.MailboxEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := entryKey{e.Owner(), e.MessageID()}
	previous, exists := s.entries[key]
	if !exists {
		return apperr.New(apperr.NotFound, "el mensaje "+e.MessageID()+" no está en su buzón")
	}
	s.entries[key] = entryToRecord(e)
	if err := s.persistLocked(); err != nil {
		s.entries[key] = previous
		return err
	}
	return nil
}
