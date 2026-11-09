package handlers

import (
	"net/http"

	"maizap/internal/apperr"
	"maizap/internal/models"
	"maizap/internal/protocol"
	"maizap/internal/services"
)

func (a *API) sendMessage(w http.ResponseWriter, r *http.Request, user *models.User, _ string) {
	var req protocol.SendRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	view, err := a.mail.Send(user, services.ComposeInput{To: req.To, Subject: req.Subject, Body: req.Body})
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusCreated, toDetail(view))
}

func (a *API) mailbox(w http.ResponseWriter, r *http.Request, user *models.User, _ string) {
	folder := models.Folder(r.PathValue("folder"))
	var (
		views []*services.MailView
		err   error
	)
	switch folder {
	case models.FolderInbox:
		views, err = a.mail.Inbox(user)
	case models.FolderSent:
		views, err = a.mail.Sent(user)
	default:
		err = apperr.New(apperr.NotFound, "la carpeta "+string(folder)+" no existe; use inbox o sent")
	}
	if err != nil {
		writeError(w, err)
		return
	}
	resp := protocol.MailboxResponse{Folder: string(folder), Messages: make([]protocol.MessageSummary, 0, len(views))}
	for _, v := range views {
		summary := toSummary(v)
		if !summary.Read {
			resp.Unread++
		}
		resp.Messages = append(resp.Messages, summary)
	}
	resp.Total = len(resp.Messages)
	writeData(w, http.StatusOK, resp)
}

func (a *API) readMessage(w http.ResponseWriter, r *http.Request, user *models.User, _ string) {
	view, err := a.mail.Read(user, r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, toDetail(view))
}

func (a *API) deleteMessage(w http.ResponseWriter, r *http.Request, user *models.User, _ string) {
	id := r.PathValue("id")
	if err := a.mail.Delete(user, id); err != nil {
		writeError(w, err)
		return
	}
	writeData(w, http.StatusOK, map[string]string{"message": "mensaje " + id + " eliminado de su buzón"})
}

func toSummary(v *services.MailView) protocol.MessageSummary {
	to := make([]protocol.UserRef, 0, len(v.Recipients))
	for _, r := range v.Recipients {
		to = append(to, toUserRef(r))
	}
	return protocol.MessageSummary{
		ID:      v.Message.ID(),
		Folder:  string(v.Entry.Folder()),
		From:    toUserRef(v.Sender),
		To:      to,
		Subject: v.Message.Subject(),
		SentAt:  v.Message.SentAt(),
		Read:    v.Entry.IsRead(),
	}
}

func toDetail(v *services.MailView) protocol.MessageDetail {
	detail := protocol.MessageDetail{MessageSummary: toSummary(v), Body: v.Message.Body()}
	if v.Entry.IsRead() {
		readAt := v.Entry.ReadAt()
		detail.ReadAt = &readAt
	}
	return detail
}
