package api

import (
	"net/http"

	"github.com/NeoTheCapt/clawfleet/internal/version"
)

// handleVersion returns backend build info.
// Public endpoint: GET /api/version
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, version.Get())
}
