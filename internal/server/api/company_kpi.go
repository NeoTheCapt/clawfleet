package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ═══════════════════════════════════════════════════════════
// Review Cycles
// ═══════════════════════════════════════════════════════════

func (s *Server) handleReviewCycles(w http.ResponseWriter, r *http.Request, companyID string, sub string) {
	sub = strings.TrimPrefix(sub, "/")
	if sub == "" {
		switch r.Method {
		case http.MethodGet:
			list, err := s.store.ListReviewCycles(companyID)
			if err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 200, list)
		case http.MethodPost:
			var rc model.ReviewCycle
			if !decodeJSON(w, r, &rc) {
				return
			}
			rc.ID = generateID("cycle")
			rc.CompanyID = companyID
			rc.CreatedAt = time.Now()
			rc.UpdatedAt = time.Now()
			if rc.Frequency == "" {
				rc.Frequency = "weekly"
			}
			if err := s.store.CreateReviewCycle(&rc); err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 201, rc)
		default:
			writeError(w, 405, "method not allowed")
		}
		return
	}
	cycleID := sub
	switch r.Method {
	case http.MethodPut:
		var rc model.ReviewCycle
		if !decodeJSON(w, r, &rc) {
			return
		}
		rc.ID = cycleID
		if err := s.store.UpdateReviewCycle(&rc); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "updated")
	case http.MethodDelete:
		if err := s.store.DeleteReviewCycle(cycleID); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "deleted")
	}
}

// ═══════════════════════════════════════════════════════════
// KPI Reviews
// ═══════════════════════════════════════════════════════════

func (s *Server) handleKPIReviews(w http.ResponseWriter, r *http.Request, companyID string, sub string) {
	switch r.Method {
	case http.MethodGet:
		agentID := r.URL.Query().Get("agent_id")
		cycleID := r.URL.Query().Get("cycle_id")
		period := r.URL.Query().Get("period")
		if agentID != "" {
			list, err := s.store.ListKPIReviews(agentID)
			if err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 200, list)
		} else if cycleID != "" && period == "" {
			// List all reviews for a cycle (any period)
			list, err := s.store.ListKPIReviewsByCycle(cycleID, "")
			if err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 200, list)
		} else if cycleID != "" && period != "" {
			list, err := s.store.ListKPIReviewsByCycle(cycleID, period)
			if err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 200, list)
		} else if cycleID == "" && period != "" {
			list, err := s.store.ListKPIReviewsByCompanyPeriod(companyID, period)
			if err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 200, list)
		} else {
			writeBadRequest(w, "agent_id or period or (cycle_id + period) required")
		}
	case http.MethodPost:
		var k model.KPIReview
		if !decodeJSON(w, r, &k) {
			return
		}
		k.ID = generateID("kpi")
		k.CreatedAt = time.Now()
		k.UpdatedAt = time.Now()
		if k.Status == "" {
			k.Status = "pending"
		}
		if err := s.store.CreateKPIReview(&k); err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, 201, k)
	case http.MethodPut:
		var k model.KPIReview
		if !decodeJSON(w, r, &k) {
			return
		}
		if err := s.store.UpdateKPIReview(&k); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "updated")
	default:
		writeError(w, 405, "method not allowed")
	}
}
