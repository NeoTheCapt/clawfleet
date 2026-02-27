package api

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// --- Request types ---

type CreateAgentRequest struct {
	Name       string            `json:"name"`
	CompanyID  string            `json:"company_id,omitempty"`
	NodeID     string            `json:"node_id"`
	AgentType  model.AgentType   `json:"agent_type"`
	Role       string            `json:"role,omitempty"`
	DeployMode string            `json:"deploy_mode,omitempty"`
	Config     model.AgentConfig `json:"config"`
}

var debugCheckWhitelist = map[string]struct{}{
	"openclaw_status_deep":  {},
	"openclaw_plugins_list": {},
	"openclaw_config":       {},
	"openclaw_logs_tail":    {},
	"telegram_getme":        {},
	"env_keys":              {},
}

// --- Agent Instances ---

func newAgentInstanceFromRequest(req CreateAgentRequest, nodeID string) *model.AgentInstance {
	deployMode := req.DeployMode
	if deployMode == "" {
		deployMode = "docker"
	}
	if nodeID == "" {
		nodeID = req.NodeID
	}
	return &model.AgentInstance{
		ID:         generateID("agent"),
		Name:       req.Name,
		CompanyID:  req.CompanyID,
		NodeID:     nodeID,
		AgentType:  req.AgentType,
		Role:       req.Role,
		Status:     model.AgentStatusCreating,
		DeployMode: deployMode,
		Config:     req.Config,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		nodeID := r.URL.Query().Get("node_id")
		companyID := r.URL.Query().Get("company_id")
		var agents []*model.AgentInstance
		var err error
		if nodeID != "" {
			agents, err = s.store.ListAgentsByNode(nodeID)
		} else if companyID != "" {
			agents, err = s.store.ListAgentsByCompany(companyID)
		} else {
			agents, err = s.store.ListAgentInstances()
		}
		if err != nil {
			writeServerError(w, err)
			return
		}
		if agents == nil {
			agents = []*model.AgentInstance{}
		}
		writeJSON(w, http.StatusOK, sanitizeAgentsForResponse(agents))
	case http.MethodPost:
		var req CreateAgentRequest
		if !decodeJSON(w, r, &req) {
			return
		}
		agent := newAgentInstanceFromRequest(req, "")
		if err := s.store.CreateAgentInstance(agent); err != nil {
			writeServerError(w, err)
			return
		}

		// If deploy=true, send deploy request to node agent
		if r.URL.Query().Get("deploy") == "true" {
			go s.deployToNode(agent)
		}

		writeJSON(w, http.StatusCreated, sanitizeAgentForResponse(agent))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleAgentByID(w http.ResponseWriter, r *http.Request) {
	id, subRoute := splitSubpath(r.URL.Path, "/api/agents/")

	if id == "" {
		writeBadRequest(w, "missing agent id")
		return
	}

	switch subRoute {
	case "config":
		s.handleAgentConfig(w, r, id)
		return
	case "restart":
		s.handleAgentRestart(w, r, id)
		return
	case "redeploy":
		s.handleAgentRedeploy(w, r, id)
		return
	case "persona":
		s.handleAgentPersona(w, r, id)
		return
	case "persona-rollback", "persona-rollback/":
		s.handlePersonaRollbackList(w, r, id)
		return
	case "persona-snapshots":
		s.handlePersonaSnapshots(w, r, id)
		return
	case "debug":
		s.handleAgentDebug(w, r, id)
		return
	case "im-key":
		s.handleAgentIMKey(w, r, id)
		return
	case "im-key/rotate":
		s.handleAgentIMKeyRotate(w, r, id)
		return
	}
	if strings.HasPrefix(subRoute, "persona-rollback/") {
		snapshotID := strings.TrimSpace(strings.TrimPrefix(subRoute, "persona-rollback/"))
		if snapshotID == "" {
			s.handlePersonaRollbackList(w, r, id)
			return
		}
		if strings.Contains(snapshotID, "/") {
			writeBadRequest(w, "invalid snapshot id")
			return
		}
		s.handlePersonaRollback(w, r, id, snapshotID)
		return
	}

	switch r.Method {
	case http.MethodGet:
		agent, err := s.store.GetAgentInstance(id)
		if err != nil {
			writeNotFound(w, "agent not found")
			return
		}
		writeJSON(w, http.StatusOK, sanitizeAgentForResponse(agent))
	case http.MethodDelete:
		agent, err := s.store.GetAgentInstance(id)
		if err != nil {
			writeNotFound(w, "agent not found")
			return
		}
		// Mark as deleting and enqueue container/service removal.
		// The record is removed after node confirms task completion.
		_ = s.store.UpdateAgentStatus(id, model.AgentStatusDeleting, agent.ContainerID)
		s.removeAgentOnNode(agent)
		writeStatus(w, http.StatusOK, "deleting")
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleAgentConfig(w http.ResponseWriter, r *http.Request, agentID string) {
	agent, err := s.store.GetAgentInstance(agentID)
	if err != nil {
		writeNotFound(w, "agent not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, sanitizeAgentConfigForResponse(agent.Config))
	case http.MethodPut:
		var cfg model.AgentConfig
		if !decodeJSON(w, r, &cfg) {
			return
		}

		// Preserve/resolve Telegram token:
		// - UI may omit or blank out token (especially when managed by Position)
		// - Token must remain in agent config so redeploy can inject env vars
		if cfg.Channel != nil && cfg.Channel.Type == "telegram" {
			if strings.TrimSpace(cfg.Channel.Token) == "" {
				resolved := ""
				// 1) If agent is assigned to a position with a Telegram bot, use that bot's token.
				if assigns, err := s.store.ListAssignmentsByAgent(agentID); err == nil && len(assigns) > 0 {
					posID := assigns[len(assigns)-1].PositionID
					if pos, err := s.store.GetPosition(posID); err == nil {
						botID := strings.TrimSpace(pos.TelegramBotID)
						if botID != "" {
							if bot, err := s.store.GetBot(botID); err == nil {
								resolved = bot.Token
							}
						}
					}
				}
				// 2) Fallback: preserve existing agent token if present.
				if resolved == "" && agent.Config.Channel != nil {
					resolved = agent.Config.Channel.Token
				}
				cfg.Channel.Token = resolved
			}
		}
		// Preserve IM key unless explicitly changed by IM key rotation endpoint.
		if existing := strings.TrimSpace(agent.Config.EnvVars[imKeyEnvVar]); existing != "" {
			if cfg.EnvVars == nil {
				cfg.EnvVars = map[string]string{}
			}
			if strings.TrimSpace(cfg.EnvVars[imKeyEnvVar]) == "" {
				cfg.EnvVars[imKeyEnvVar] = existing
			}
		}

		agent.Config = cfg
		agent.UpdatedAt = time.Now()
		if err := s.store.UpdateAgentInstance(agent); err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, sanitizeAgentForResponse(agent))
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleAgentIMKey(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, err := s.store.GetAgentInstance(agentID); err != nil {
		writeNotFound(w, "agent not found")
		return
	}
	hasKey, err := s.store.HasIMKey(agentID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"has_key": hasKey})
}

func (s *Server) handleAgentIMKeyRotate(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if _, err := s.store.GetAgentInstance(agentID); err != nil {
		if err == sql.ErrNoRows {
			writeNotFound(w, "agent not found")
			return
		}
		writeServerError(w, err)
		return
	}
	imKey, err := s.store.RotateIMKey(agentID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"im_key": imKey})
}

func (s *Server) handleAgentRestart(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	agent, err := s.store.GetAgentInstance(agentID)
	if err != nil {
		writeNotFound(w, "agent not found")
		return
	}

	// Send restart task to node
	go s.restartAgentOnNode(agent)

	writeJSON(w, http.StatusOK, map[string]string{"status": "restarting"})
}

func (s *Server) handleAgentRedeploy(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	agent, err := s.store.GetAgentInstance(agentID)
	if err != nil {
		writeNotFound(w, "agent not found")
		return
	}

	// Stop old container + deploy with fresh config
	agent.Status = model.AgentStatusCreating
	agent.UpdatedAt = time.Now()
	if err := s.store.UpdateAgentInstance(agent); err != nil {
		writeServerError(w, err)
		return
	}

	go func() {
		s.stopAgentOnNode(agent)
		s.deployToNode(agent)
	}()

	writeJSON(w, http.StatusOK, map[string]string{"status": "redeploying"})
}

func (s *Server) handleAgentDebug(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	agent, err := s.store.GetAgentInstance(agentID)
	if err != nil {
		writeNotFound(w, "agent not found")
		return
	}

	var req struct {
		Checks []string `json:"checks"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Checks) == 0 {
		writeBadRequest(w, "checks required")
		return
	}
	for _, check := range req.Checks {
		if _, ok := debugCheckWhitelist[check]; !ok {
			writeBadRequest(w, "unsupported check: "+check)
			return
		}
	}

	task := &model.Task{
		ID:     generateID("task"),
		NodeID: agent.NodeID,
		Action: "debug_agent",
		Payload: map[string]interface{}{
			"agent_id":     agent.ID,
			"agent_name":   agent.Name,
			"container_id": agent.ContainerID,
			"checks":       req.Checks,
		},
		Status:    model.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := s.store.CreateTask(task); err != nil {
		writeServerError(w, err)
		return
	}

	if node, err := s.store.GetNode(agent.NodeID); err == nil && node.Address != "" {
		s.pushTaskToNode(node, task)
	}

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"task_id": task.ID,
		"status":  task.Status,
	})
}
