package api

import (
	"crypto/rand"
	"net/http"
	"time"
)

// handleAdminCredentials handles updating admin username/password.
// Route: PUT /api/admin/credentials (JWT required)
func (s *Server) handleAdminCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewUsername     string `json:"new_username"`
		NewPassword     string `json:"new_password"`
		RotateJWT       *bool  `json:"rotate_jwt,omitempty"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.NewUsername == "" {
		writeBadRequest(w, "new_username required")
		return
	}
	if req.NewPassword == "" {
		writeBadRequest(w, "new_password required")
		return
	}
	// Verify current password (defense in depth; JWT alone might be stolen)
	if req.CurrentPassword == "" || !s.auth.CheckPassword(s.auth.AdminUser, req.CurrentPassword) {
		writeError(w, http.StatusUnauthorized, "invalid current password")
		return
	}

	// Update auth config (in-memory)
	if err := s.auth.UpdateCredentials(req.NewUsername, req.NewPassword); err != nil {
		writeServerError(w, err)
		return
	}

	// Optionally rotate JWT secret to invalidate existing tokens
	rotate := true
	if req.RotateJWT != nil {
		rotate = *req.RotateJWT
	}
	if rotate {
		secret := make([]byte, 32)
		_, _ = rand.Read(secret)
		_ = s.auth.SetSecret(secret)
	}

	// Persist to DB (use snapshot for consistency)
	u, ph, sec := s.auth.Snapshot()
	if err := s.store.UpsertAdminCredentials(u, ph, sec); err != nil {
		writeServerError(w, err)
		return
	}

	// Best-effort: set a short-lived cookie to force relogin if JWT rotated
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(0, 0),
	})

	writeStatus(w, http.StatusOK, "updated")
}
