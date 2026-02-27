package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ═══════════════════════════════════════════════════════════
// Company CRUD
// ═══════════════════════════════════════════════════════════

func (s *Server) handleCompanies(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListCompanies()
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, 200, list)
	case http.MethodPost:
		var req struct {
			Name    string `json:"name"`
			Vision  string `json:"vision"`
			Mission string `json:"mission"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.Name == "" {
			writeBadRequest(w, "name required")
			return
		}
		c := &model.Company{
			ID: generateID("co"), Name: req.Name, Vision: req.Vision,
			Mission: req.Mission, Status: "active",
			CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := s.store.CreateCompany(c); err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, 201, c)
	default:
		writeError(w, 405, "method not allowed")
	}
}

func (s *Server) handleCompanyByID(w http.ResponseWriter, r *http.Request) {
	id, sub := splitSubpath(r.URL.Path, "/api/companies/")
	if sub != "" {
		switch {
		case sub == "goals" || strings.HasPrefix(sub, "goals/"):
			s.handleGoals(w, r, id, strings.TrimPrefix(sub, "goals"))
		case sub == "departments" || strings.HasPrefix(sub, "departments/"):
			s.handleDepartments(w, r, id, strings.TrimPrefix(sub, "departments"))
		case sub == "positions":
			s.handleAllPositions(w, r, id)
		case sub == "assignments" || strings.HasPrefix(sub, "assignments/"):
			s.handleAssignments(w, r, id, strings.TrimPrefix(sub, "assignments"))
		case sub == "review-cycles" || strings.HasPrefix(sub, "review-cycles/"):
			s.handleReviewCycles(w, r, id, strings.TrimPrefix(sub, "review-cycles"))
		case sub == "kpi-reviews" || strings.HasPrefix(sub, "kpi-reviews/"):
			s.handleKPIReviews(w, r, id, strings.TrimPrefix(sub, "kpi-reviews"))
		case sub == "im-groups" || strings.HasPrefix(sub, "im-groups/"):
			s.handleIMGroups(w, r, id, strings.TrimPrefix(sub, "im-groups"))
		case sub == "workflows" || strings.HasPrefix(sub, "workflows/"):
			s.handleWorkflows(w, r, id, strings.TrimPrefix(sub, "workflows"))
		case sub == "governance-logs":
			s.handleGovernanceLogs(w, r, id)
		case sub == "bots" || strings.HasPrefix(sub, "bots/"):
			s.handleBots(w, r, id, strings.TrimPrefix(sub, "bots"))
		case sub == "org-chart":
			s.handleOrgChart(w, r, id)
		default:
			writeNotFound(w, "not found")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		c, err := s.store.GetCompany(id)
		if err != nil {
			writeNotFound(w, "company not found")
			return
		}
		writeJSON(w, 200, c)
	case http.MethodPut:
		var req model.Company
		if !decodeJSON(w, r, &req) {
			return
		}
		req.ID = id
		if err := s.store.UpdateCompany(&req); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "updated")
	case http.MethodDelete:
		if err := s.store.DeleteCompany(id); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "deleted")
	default:
		writeError(w, 405, "method not allowed")
	}
}
