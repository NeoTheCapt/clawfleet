package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// --- Registration Tokens ---

func (s *Server) handleTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tokens, err := s.store.ListTokens()
		if err != nil {
			writeServerError(w, err)
			return
		}
		if tokens == nil {
			tokens = []*model.Token{}
		}
		writeJSON(w, http.StatusOK, tokens)
	case http.MethodPost:
		var req struct {
			Name string `json:"name"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.Name == "" {
			req.Name = "token"
		}
		t := &model.Token{
			ID:        generateID("tok"),
			Token:     generateToken(),
			Name:      req.Name,
			Used:      false,
			CreatedAt: time.Now(),
		}
		if err := s.store.CreateToken(t); err != nil {
			writeServerError(w, err)
			return
		}
		// Convenience for UI: provide a one-click install command.
		// NOTE: Token itself is still returned (used for manual installs and auditing).
		host := r.Host
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		installCmd := fmt.Sprintf("curl -fsSL %s://%s/api/install.sh?token=%s | sudo bash", scheme, host, t.Token)
		writeJSON(w, http.StatusCreated, map[string]interface{}{
			"id":              t.ID,
			"token":           t.Token,
			"name":            t.Name,
			"used":            t.Used,
			"created_at":      t.CreatedAt,
			"install_command": installCmd,
		})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleTokenByID(w http.ResponseWriter, r *http.Request) {
	id, _ := splitSubpath(r.URL.Path, "/api/tokens/")
	if id == "" {
		writeBadRequest(w, "missing token id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		t, err := s.store.GetTokenByID(id)
		if err != nil {
			writeNotFound(w, "token not found")
			return
		}
		writeJSON(w, http.StatusOK, t)
	case http.MethodDelete:
		if err := s.store.DeleteToken(id); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, http.StatusOK, "deleted")
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
