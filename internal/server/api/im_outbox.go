package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"github.com/NeoTheCapt/clawfleet/internal/model"
	"github.com/NeoTheCapt/clawfleet/internal/store"
)

// /api/im/outbox (agent-only)
// GET: list pending outbox items for this agent
// POST /api/im/outbox/{id}/ack
// POST /api/im/outbox/{id}/fail
func (s *Server) handleIMOutbox(w http.ResponseWriter, r *http.Request) {
	pr, ok := getIMPrincipal(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if pr.SenderType != string(model.IMMemberTypeAgent) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	// subroute
	id, sub := splitSubpath(r.URL.Path, "/api/im/outbox/")
	if id != "" {
		s.handleIMOutboxByID(w, r, pr, id, sub)
		return
	}

	if r.URL.Path != "/api/im/outbox" {
		writeNotFound(w, "not found")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "pending"
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}

	items, err := s.store.ListIMOutbox(pr.SenderID, status, limit)
	if err != nil {
		writeServerError(w, err)
		return
	}
	if items == nil {
		items = []*store.IMOutboxItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) handleIMOutboxByID(w http.ResponseWriter, r *http.Request, pr imPrincipal, id, sub string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if sub != "ack" && sub != "fail" {
		writeNotFound(w, "subroute not found")
		return
	}

	item, err := s.store.GetIMOutboxItem(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeNotFound(w, "outbox item not found")
			return
		}
		writeServerError(w, err)
		return
	}
	if item.ToAgentID != pr.SenderID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	if sub == "ack" {
		if err := s.store.AckIMOutbox(id); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, http.StatusOK, "ok")
		return
	}

	// fail
	if err := s.store.FailIMOutbox(id); err != nil {
		writeServerError(w, err)
		return
	}
	writeStatus(w, http.StatusOK, "ok")
}
