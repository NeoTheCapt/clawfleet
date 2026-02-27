package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ═══════════════════════════════════════════════════════════
// IM Groups
// ═══════════════════════════════════════════════════════════

func (s *Server) handleIMGroups(w http.ResponseWriter, r *http.Request, companyID string, sub string) {
	sub = strings.TrimPrefix(sub, "/")
	if sub == "" {
		switch r.Method {
		case http.MethodGet:
			list, err := s.store.ListIMGroups(companyID)
			if err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 200, list)
		case http.MethodPost:
			var g model.IMGroup
			if !decodeJSON(w, r, &g) {
		return
	}
			g.ID = generateID("im")
			g.CompanyID = companyID
			g.CreatedAt = time.Now()
			g.UpdatedAt = time.Now()
			if err := s.store.CreateIMGroup(&g); err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 201, g)
		default:
			writeError(w, 405, "method not allowed")
		}
		return
	}
	groupID := sub
	switch r.Method {
	case http.MethodPut:
		var g model.IMGroup
		if !decodeJSON(w, r, &g) {
		return
	}
		g.ID = groupID
		if err := s.store.UpdateIMGroup(&g); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "updated")
	case http.MethodDelete:
		if err := s.store.DeleteIMGroup(groupID); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "deleted")
	}
}

// ═══════════════════════════════════════════════════════════
// Workflows
// ═══════════════════════════════════════════════════════════

func (s *Server) handleWorkflows(w http.ResponseWriter, r *http.Request, companyID string, sub string) {
	sub = strings.TrimPrefix(sub, "/")
	if sub == "" {
		switch r.Method {
		case http.MethodGet:
			list, err := s.store.ListWorkflows(companyID)
			if err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 200, list)
		case http.MethodPost:
			var wf model.Workflow
			if !decodeJSON(w, r, &wf) {
		return
	}
			wf.ID = generateID("wf")
			wf.CompanyID = companyID
			wf.CreatedAt = time.Now()
			wf.UpdatedAt = time.Now()
			if err := s.store.CreateWorkflow(&wf); err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 201, wf)
		default:
			writeError(w, 405, "method not allowed")
		}
		return
	}
	wfID := sub
	switch r.Method {
	case http.MethodPut:
		var wf model.Workflow
		if !decodeJSON(w, r, &wf) {
		return
	}
		wf.ID = wfID
		if err := s.store.UpdateWorkflow(&wf); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "updated")
	case http.MethodDelete:
		if err := s.store.DeleteWorkflow(wfID); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "deleted")
	}
}

// ═══════════════════════════════════════════════════════════
// Governance Logs
// ═══════════════════════════════════════════════════════════

func (s *Server) handleGovernanceLogs(w http.ResponseWriter, r *http.Request, companyID string) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListGovernanceLogs(companyID, 100)
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, 200, list)
	case http.MethodPost:
		var g model.GovernanceLog
		if !decodeJSON(w, r, &g) {
		return
	}
		g.ID = generateID("gov")
		g.CompanyID = companyID
		g.Timestamp = time.Now()
		if err := s.store.CreateGovernanceLog(&g); err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, 201, g)
	default:
		writeError(w, 405, "method not allowed")
	}
}
