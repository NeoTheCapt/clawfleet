package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
	"github.com/NeoTheCapt/clawfleet/internal/sync"
)

// handleBots handles /api/companies/{id}/bots and /api/companies/{id}/bots/{botId}
func (s *Server) handleBots(w http.ResponseWriter, r *http.Request, companyID, sub string) {
	botID := strings.TrimPrefix(sub, "/")

	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListBots(companyID)
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var b model.BotConfig
		if !decodeJSON(w, r, &b) {
			return
		}
		b.ID = generateID("bot")
		b.CompanyID = companyID
		b.CreatedAt = time.Now()
		b.UpdatedAt = time.Now()
		if err := s.store.CreateBot(&b); err != nil {
			writeServerError(w, err)
			return
		}
		s.syncCompanyBots(companyID, "bot_created")
		writeJSON(w, http.StatusCreated, b)
	case http.MethodPut:
		if botID == "" {
			writeBadRequest(w, "bot id required")
			return
		}
		var b model.BotConfig
		if !decodeJSON(w, r, &b) {
			return
		}
		b.ID = botID
		b.CompanyID = companyID
		b.UpdatedAt = time.Now()
		if err := s.store.UpdateBot(&b); err != nil {
			writeServerError(w, err)
			return
		}
		s.syncCompanyBots(companyID, "bot_updated")
		writeStatus(w, http.StatusOK, "updated")
	case http.MethodDelete:
		if botID == "" {
			writeBadRequest(w, "bot id required")
			return
		}
		if err := s.store.DeleteBot(botID); err != nil {
			writeServerError(w, err)
			return
		}
		s.syncCompanyBots(companyID, "bot_deleted")
		writeStatus(w, http.StatusOK, "deleted")
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// syncCompanyBots updates allowlist and bot token for all assignments in a company
func (s *Server) syncCompanyBots(companyID, trigger string) {
	if s.syncEngine == nil {
		return
	}
	go func() {
		assignments, _ := s.store.ListAssignments(companyID)
		for _, a := range assignments {
			_ = s.syncEngine.SyncToAgent(a.AgentID, a.PositionID, trigger)
			if agent, err := s.store.GetAgentInstance(a.AgentID); err == nil {
				s.redeployAgent(agent)
			}
		}
	}()
}

func (s *Server) redeployAgent(agent *model.AgentInstance) {
	agent.Status = model.AgentStatusCreating
	agent.UpdatedAt = time.Now()
	_ = s.store.UpdateAgentInstance(agent)
	go func() {
		s.stopAgentOnNode(agent)
		s.deployToNode(agent)
	}()
}

// ensure compiler uses sync package
var _ = sync.SyncOptions{}
