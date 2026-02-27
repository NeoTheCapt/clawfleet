package api

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
	"github.com/NeoTheCapt/clawfleet/internal/sync"
)

// ═══════════════════════════════════════════════════════════
// Goals
// ═══════════════════════════════════════════════════════════

func (s *Server) handleGoals(w http.ResponseWriter, r *http.Request, companyID string, sub string) {
	sub = strings.TrimPrefix(sub, "/")
	if sub == "" {
		switch r.Method {
		case http.MethodGet:
			list, err := s.store.ListGoals(companyID)
			if err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 200, list)
		case http.MethodPost:
			var g model.CompanyGoal
			if !decodeJSON(w, r, &g) {
				return
			}
			g.ID = generateID("goal")
			g.CompanyID = companyID
			g.CreatedAt = time.Now()
			g.UpdatedAt = time.Now()
			if g.Status == "" {
				g.Status = model.GoalStatusPlanning
			}
			if g.Priority == "" {
				g.Priority = "medium"
			}
			if err := s.store.CreateGoal(&g); err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 201, g)
		default:
			writeError(w, 405, "method not allowed")
		}
		return
	}
	// PUT/DELETE /goals/{id}
	goalID := sub
	switch r.Method {
	case http.MethodPut:
		var g model.CompanyGoal
		if !decodeJSON(w, r, &g) {
			return
		}
		g.ID = goalID
		if err := s.store.UpdateGoal(&g); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "updated")
	case http.MethodDelete:
		if err := s.store.DeleteGoal(goalID); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "deleted")
	}
}

// ═══════════════════════════════════════════════════════════
// Departments
// ═══════════════════════════════════════════════════════════

func (s *Server) handleDepartments(w http.ResponseWriter, r *http.Request, companyID string, sub string) {
	sub = strings.TrimPrefix(sub, "/")
	if sub == "" {
		switch r.Method {
		case http.MethodGet:
			list, err := s.store.ListDepartments(companyID)
			if err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 200, list)
		case http.MethodPost:
			var d model.Department
			if !decodeJSON(w, r, &d) {
				return
			}
			d.ID = generateID("dept")
			d.CompanyID = companyID
			d.CreatedAt = time.Now()
			d.UpdatedAt = time.Now()
			if err := s.store.CreateDepartment(&d); err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, 201, d)
		default:
			writeError(w, 405, "method not allowed")
		}
		return
	}

	// Check for /departments/{id}/positions or /departments/{id}/positions/{posId}
	parts := strings.SplitN(sub, "/", 2)
	deptID := parts[0]
	if len(parts) > 1 && strings.HasPrefix(parts[1], "positions") {
		posParts := strings.SplitN(parts[1], "/", 2)
		if len(posParts) > 1 && posParts[1] != "" {
			// /departments/{id}/positions/{posId}
			s.handlePositionByID(w, r, deptID, posParts[1])
		} else {
			// /departments/{id}/positions
			s.handlePositions(w, r, deptID)
		}
		return
	}

	switch r.Method {
	case http.MethodPut:
		var d model.Department
		if !decodeJSON(w, r, &d) {
			return
		}
		d.ID = deptID
		if err := s.store.UpdateDepartment(&d); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "updated")
	case http.MethodDelete:
		if err := s.store.DeleteDepartment(deptID); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "deleted")
	}
}

// ═══════════════════════════════════════════════════════════
// Positions
// ═══════════════════════════════════════════════════════════

func (s *Server) handlePositions(w http.ResponseWriter, r *http.Request, deptID string) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListPositions(deptID)
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, 200, list)
	case http.MethodPost:
		var p model.Position
		if !decodeJSON(w, r, &p) {
			return
		}

		// 🔄 新增：自动生成人设背景（如果未提供）
		if p.SystemPrompt == "" && p.Background == "" && s.personaGen != nil {
			profile, err := s.personaGen.GenerateForPosition(&p)
			if err == nil {
				p.SystemPrompt = profile.SystemPrompt
				p.Background = profile.Background
			} else {
				log.Printf("[persona] failed to generate persona for position %s: %v", p.Title, err)
			}
		}

		p.ID = generateID("pos")
		p.DepartmentID = deptID
		p.CreatedAt = time.Now()
		p.UpdatedAt = time.Now()
		if p.Level == "" {
			p.Level = model.PositionLevelStaff
		}
		if err := s.store.CreatePosition(&p); err != nil {
			writeServerError(w, err)
			return
		}

		// 🔄 新增：记录人设创建日志
		log.Printf("[persona] position created: %s (ID: %s) with system_prompt: %d chars, background: %d chars",
			p.Title, p.ID, len(p.SystemPrompt), len(p.Background))

		writeJSON(w, 201, p)
	default:
		writeError(w, 405, "method not allowed")
	}
}

func (s *Server) handlePositionByID(w http.ResponseWriter, r *http.Request, deptID string, posID string) {
	switch r.Method {
	case http.MethodPut:
		var p model.Position
		if !decodeJSON(w, r, &p) {
			return
		}

		// 获取现有职位信息以检测变更
		existingPos, err := s.store.GetPosition(posID)
		if err != nil {
			writeNotFound(w, "position not found")
			return
		}

		// 如果人设字段为空，尝试生成
		if (p.SystemPrompt == "" || p.Background == "") && s.personaGen != nil {
			// 合并现有信息
			if p.Title == "" {
				p.Title = existingPos.Title
			}
			if p.Level == "" {
				p.Level = existingPos.Level
			}
			if p.Responsibilities == "" {
				p.Responsibilities = existingPos.Responsibilities
			}
			if p.ReportsTo == "" {
				p.ReportsTo = existingPos.ReportsTo
			}

			profile, err := s.personaGen.GenerateForPosition(&p)
			if err == nil {
				if p.SystemPrompt == "" {
					p.SystemPrompt = profile.SystemPrompt
				}
				if p.Background == "" {
					p.Background = profile.Background
				}
			} else {
				log.Printf("[persona] failed to generate persona for position update %s: %v", posID, err)
			}
		}

		p.ID = posID
		p.DepartmentID = deptID
		p.UpdatedAt = time.Now()
		if err := s.store.UpdatePosition(&p); err != nil {
			writeServerError(w, err)
			return
		}

		// 检测人设变更，如果需要同步到agents
		if existingPos.SystemPrompt != p.SystemPrompt || existingPos.Background != p.Background {
			log.Printf("[persona] position %s persona updated, system_prompt changed: %v, background changed: %v",
				posID, existingPos.SystemPrompt != p.SystemPrompt, existingPos.Background != p.Background)
			if s.syncEngine != nil {
				go func() {
					_ = s.syncEngine.SyncPositionPersonaToAllAgents(posID, &sync.SyncOptions{Trigger: "position_updated"})
				}()
			}
		}

		writeJSON(w, 200, p)
	case http.MethodDelete:
		if err := s.store.DeletePosition(posID); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "deleted")
	default:
		writeError(w, 405, "method not allowed")
	}
}

func (s *Server) handleAllPositions(w http.ResponseWriter, r *http.Request, companyID string) {
	list, err := s.store.ListAllPositions(companyID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, 200, list)
}

// handlePositionWithSubpath handles subpaths under /api/positions/{id}
func (s *Server) handlePositionWithSubpath(w http.ResponseWriter, r *http.Request) {
	id, subRoute := splitSubpath(r.URL.Path, "/api/positions/")

	if id == "" {
		writeBadRequest(w, "missing position id")
		return
	}

	switch subRoute {
	case "sync-persona":
		s.handleSyncPersona(w, r)
		return
		// 可以添加其他人设相关子路径
	}

	// 如果没有匹配的子路径，返回404
	writeNotFound(w, "subroute not found")
}

// ═══════════════════════════════════════════════════════════
// Agent Assignments
// ═══════════════════════════════════════════════════════════

func (s *Server) handleAssignments(w http.ResponseWriter, r *http.Request, companyID string, sub string) {
	assignmentID := ""
	if strings.HasPrefix(sub, "/") {
		assignmentID = strings.TrimPrefix(sub, "/")
	}

	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListAssignments(companyID)
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, 200, list)
	case http.MethodPost:
		var req struct {
			PositionID string `json:"position_id"`
			AgentID    string `json:"agent_id,omitempty"`    // optional, if not provided, create new agent
			NodeID     string `json:"node_id,omitempty"`     // optional, if not provided, pick an online node
			AgentType  string `json:"agent_type,omitempty"`  // optional, default "openclaw"
			DeployMode string `json:"deploy_mode,omitempty"` // optional, default "docker"
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.PositionID == "" {
			writeBadRequest(w, "position_id required")
			return
		}

		// Get position details
		position, err := s.store.GetPosition(req.PositionID)
		if err != nil {
			writeNotFound(w, "position not found")
			return
		}

		var agentID string
		if req.AgentID != "" {
			// Use existing agent
			agentID = req.AgentID
		} else {
			// Create new agent instance based on position
			agentID, err = s.createAgentForPosition(position, req.NodeID, req.AgentType, req.DeployMode)
			if err != nil {
				writeError(w, 500, "failed to create agent: "+err.Error())
				return
			}
		}

		// Create assignment
		a := model.AgentAssignment{
			ID:         generateID("assign"),
			AgentID:    agentID,
			PositionID: req.PositionID,
			Status:     "active",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		if err := s.store.CreateAssignment(&a); err != nil {
			writeServerError(w, err)
			return
		}
		if s.syncEngine != nil {
			go func() {
				_ = s.syncEngine.SyncToAgent(agentID, req.PositionID, "assignment")
			}()
		}
		writeJSON(w, 201, a)
	case http.MethodPut:
		var a model.AgentAssignment
		if !decodeJSON(w, r, &a) {
			return
		}
		if err := s.store.UpdateAssignment(&a); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "updated")
	case http.MethodDelete:
		id := assignmentID
		if id == "" {
			id = r.URL.Query().Get("id")
		}
		if id == "" {
			writeBadRequest(w, "id required")
			return
		}
		if err := s.store.DeleteAssignment(id); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, 200, "deleted")
	default:
		writeError(w, 405, "method not allowed")
	}
}

// createAgentForPosition creates a new agent instance for the given position
func (s *Server) createAgentForPosition(position *model.Position, nodeID, agentType, deployMode string) (string, error) {
	// Determine node
	node, err := s.pickNode(nodeID)
	if err != nil {
		return "", err
	}

	// Default agent type
	if agentType == "" {
		agentType = "openclaw"
	}

	// Default deploy mode
	if deployMode == "" {
		deployMode = "docker"
	}

	// Get company ID for this position
	companyID, _ := s.store.GetCompanyByPositionID(position.ID)

	// Build agent config from position
	config := model.AgentConfig{
		SystemPrompt: position.SystemPrompt,
		PersonaName:  position.Title,
		Language:     "zh", // default Chinese
		Resources: model.ResourceLimit{
			CPULimit:    "2",
			MemoryLimit: "4Gi",
		},
	}

	// Create agent instance
	agent := &model.AgentInstance{
		ID:         generateID("agent"),
		Name:       position.Title + "_" + randomSuffix(),
		CompanyID:  companyID,
		NodeID:     node.ID,
		AgentType:  model.AgentType(agentType),
		Role:       position.Title,
		Status:     model.AgentStatusCreating,
		DeployMode: deployMode,
		Config:     config,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := s.store.CreateAgentInstance(agent); err != nil {
		return "", err
	}

	// Initialize persona snapshot/status before deploy
	if s.syncEngine != nil {
		_ = s.syncEngine.SyncToAgent(agent.ID, position.ID, "init")
	}

	// Trigger deployment
	go s.deployToNode(agent)

	return agent.ID, nil
}

// pickNode selects a node for the agent. If nodeID is provided, returns that node.
// Otherwise picks the first online node.
func (s *Server) pickNode(nodeID string) (*model.Node, error) {
	if nodeID != "" {
		return s.store.GetNode(nodeID)
	}

	// Get all nodes and pick first online
	nodes, err := s.store.ListNodes()
	if err != nil {
		return nil, err
	}
	for _, n := range nodes {
		if n.Status == model.NodeStatusOnline {
			return n, nil
		}
	}
	return nil, fmt.Errorf("no online nodes available")
}

// ═══════════════════════════════════════════════════════════
// Org Chart (read-only aggregate view)
// ═══════════════════════════════════════════════════════════

func (s *Server) handleOrgChart(w http.ResponseWriter, r *http.Request, companyID string) {
	company, err := s.store.GetCompany(companyID)
	if err != nil {
		writeNotFound(w, "company not found")
		return
	}
	depts, _ := s.store.ListDepartments(companyID)
	positions, _ := s.store.ListAllPositions(companyID)
	assignments, _ := s.store.ListAssignments(companyID)
	goals, _ := s.store.ListGoals(companyID)

	writeJSON(w, 200, map[string]interface{}{
		"company":     company,
		"departments": depts,
		"positions":   positions,
		"assignments": assignments,
		"goals":       goals,
	})
}
