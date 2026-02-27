package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
	"github.com/NeoTheCapt/clawfleet/internal/persona"
	"github.com/NeoTheCapt/clawfleet/internal/sync"
)

// ═══════════════════════════════════════════════════════════
// Persona Templates API
// ═══════════════════════════════════════════════════════════

func (s *Server) handlePersonaTemplates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := s.store.ListPersonaTemplates()
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var pt model.PersonaTemplate
		if !decodeJSON(w, r, &pt) {
			return
		}
		if pt.Name == "" {
			writeBadRequest(w, "name required")
			return
		}
		pt.ID = generateID("pt")
		pt.CreatedAt = time.Now()
		pt.UpdatedAt = time.Now()
		if err := s.store.CreatePersonaTemplate(&pt); err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, pt)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handlePersonaTemplateByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/persona-templates/")
	if id == "" {
		writeBadRequest(w, "missing template id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		pt, err := s.store.GetPersonaTemplate(id)
		if err != nil {
			writeNotFound(w, "template not found")
			return
		}
		writeJSON(w, http.StatusOK, pt)
	case http.MethodPut:
		var pt model.PersonaTemplate
		if !decodeJSON(w, r, &pt) {
			return
		}
		pt.ID = id
		pt.UpdatedAt = time.Now()
		if err := s.store.UpdatePersonaTemplate(&pt); err != nil {
			writeServerError(w, err)
			return
		}
		// 同步所有引用该模板的职位
		if s.syncEngine != nil {
			go func(templateID string) {
				positions, _ := s.store.ListPositionsByTemplate(templateID)
				for _, p := range positions {
					_ = s.syncEngine.SyncPositionPersonaToAllAgents(p.ID, &sync.SyncOptions{Trigger: "template_updated"})
				}
			}(id)
		}
		writeStatus(w, http.StatusOK, "updated")
	case http.MethodDelete:
		if err := s.store.DeletePersonaTemplate(id); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, http.StatusOK, "deleted")
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleSyncPersona handles persona synchronization endpoints
func (s *Server) handleSyncPersona(w http.ResponseWriter, r *http.Request) {
	// Extract position ID from URL: /api/positions/{id}/sync-persona
	positionID, sub := splitSubpath(r.URL.Path, "/api/positions/")
	if sub != "sync-persona" {
		writeBadRequest(w, "invalid path")
		return
	}
	// positionID extracted above

	switch r.Method {
	case http.MethodPost:
		// Sync persona to all agents assigned to this position
		result, err := s.SyncPositionPersonaToAllAgents(positionID)
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	case http.MethodGet:
		// Get sync history for this position
		history, err := s.store.ListPersonaSyncHistory(positionID, 50)
		if err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, history)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// SyncPersonaToAgent 将 position 的人设背景同步到 agent
func (s *Server) SyncPersonaToAgent(positionID, agentID string) error {
	// 获取 position 的人设配置
	systemPrompt, background, err := s.store.GetPersonaForPosition(positionID)
	if err != nil {
		return err
	}

	_, err = s.applyPersonaToAgent(agentID, systemPrompt, background, "assignment")
	return err
}

// applyPersonaToAgent applies persona configuration to an agent
func (s *Server) applyPersonaToAgent(agentID, systemPrompt, background, action string) (map[string]interface{}, error) {
	// 获取 agent
	agent, err := s.store.GetAgentInstance(agentID)
	if err != nil {
		return nil, err
	}

	// 获取 position (if available)
	var positionID string
	assignments, _ := s.store.ListAssignmentsByAgent(agentID)
	if len(assignments) > 0 {
		positionID = assignments[0].PositionID
	}

	// 记录旧的配置
	oldSystemPrompt := agent.Config.SystemPrompt
	oldBackground := agent.Config.Description

	// 更新 agent 配置
	if systemPrompt != "" {
		agent.Config.SystemPrompt = systemPrompt
	}
	if background != "" {
		agent.Config.Description = background
	}

	// 设置角色名称
	if agent.Config.PersonaName == "" {
		// 尝试从 position 获取标题，否则使用 agent 名称
		if positionID != "" {
			position, err := s.store.GetPosition(positionID)
			if err == nil && position.Title != "" {
				agent.Config.PersonaName = position.Title
			}
		}
		if agent.Config.PersonaName == "" {
			agent.Config.PersonaName = agent.Name
		}
	}

	agent.UpdatedAt = time.Now()
	if err := s.store.UpdateAgentInstance(agent); err != nil {
		// 记录失败
		if positionID != "" {
			if syncErr := s.recordPersonaSync(positionID, agentID, action, oldSystemPrompt, systemPrompt, oldBackground, background, "failed", err.Error()); syncErr != nil {
				return nil, err
			}
		}
		return nil, err
	}

	// 记录成功
	var syncErr error
	if positionID != "" {
		syncErr = s.recordPersonaSync(positionID, agentID, action, oldSystemPrompt, systemPrompt, oldBackground, background, "success", "")
	}

	// 如果 agent 正在运行，触发配置重载
	if agent.Status == model.AgentStatusRunning {
		go s.restartAgentOnNode(agent)
	}

	result := map[string]interface{}{
		"agent_id":              agentID,
		"position_id":           positionID,
		"system_prompt_updated": systemPrompt != "",
		"background_updated":    background != "",
		"action":                action,
		"timestamp":             time.Now(),
	}

	return result, syncErr
}

// SyncPositionPersonaToAllAgents 将 position 的人设同步到所有分配的 agents
func (s *Server) SyncPositionPersonaToAllAgents(positionID string) (map[string]interface{}, error) {
	// 使用 sync engine 同步
	if s.syncEngine != nil {
		err := s.syncEngine.SyncPositionPersonaToAllAgents(positionID, nil)
		if err != nil {
			return nil, err
		}

		// 获取同步结果
		assignments, _ := s.store.ListAssignmentsByPosition(positionID)
		systemPrompt, background, _ := s.store.GetPersonaForPosition(positionID)

		return map[string]interface{}{
			"position_id":   positionID,
			"total_agents":  len(assignments),
			"success_count": len(assignments),
			"failure_count": 0,
			"system_prompt": systemPrompt,
			"background":    background,
			"timestamp":     time.Now(),
		}, nil
	}

	// 获取该 position 的所有 assignments
	assignments, err := s.store.ListAssignmentsByPosition(positionID)
	if err != nil {
		return nil, err
	}

	// 获取 position 的人设配置
	systemPrompt, background, err := s.store.GetPersonaForPosition(positionID)
	if err != nil {
		return nil, err
	}

	results := []map[string]interface{}{}
	successCount := 0
	failureCount := 0

	for _, assignment := range assignments {
		result, err := s.applyPersonaToAgent(assignment.AgentID, systemPrompt, background, "sync")
		if err != nil {
			failureCount++
			result = map[string]interface{}{
				"agent_id": assignment.AgentID,
				"error":    err.Error(),
				"status":   "failed",
			}
		} else {
			successCount++
		}
		results = append(results, result)
	}

	summary := map[string]interface{}{
		"position_id":   positionID,
		"total_agents":  len(assignments),
		"success_count": successCount,
		"failure_count": failureCount,
		"system_prompt": systemPrompt,
		"background":    background,
		"results":       results,
		"timestamp":     time.Now(),
	}

	return summary, nil
}

// recordPersonaSync records a persona synchronization event
func (s *Server) recordPersonaSync(positionID, agentID, action, oldSystemPrompt, newSystemPrompt, oldBackground, newBackground, status, errorMsg string) error {
	record := &model.PersonaSyncRecord{
		ID:              generateID("psync"),
		PositionID:      positionID,
		AgentID:         agentID,
		Action:          action,
		OldSystemPrompt: oldSystemPrompt,
		NewSystemPrompt: newSystemPrompt,
		OldBackground:   oldBackground,
		NewBackground:   newBackground,
		Status:          status,
		Error:           errorMsg,
		SyncedAt:        time.Now(),
	}
	return s.store.CreatePersonaSyncRecord(record)
}

// (removed) ListAssignmentsByAgent was a thin wrapper over store.ListAssignmentsByAgent.

// ═══════════════════════════════════════════════════════════
// Persona Generation API
// ═══════════════════════════════════════════════════════════

func (s *Server) handlePersonaGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		PositionTitle    string              `json:"position_title"`
		Level            model.PositionLevel `json:"level"`
		Responsibilities string              `json:"responsibilities,omitempty"`
		ReportsTo        string              `json:"reports_to,omitempty"`
		DepartmentID     string              `json:"department_id,omitempty"`
		TemplateID       string              `json:"template_id,omitempty"`
	}

	if !decodeJSON(w, r, &req) {
		return
	}

	if req.PositionTitle == "" {
		writeBadRequest(w, "position_title required")
		return
	}

	if req.Level == "" {
		req.Level = model.PositionLevelStaff
	}

	// 创建临时position对象用于生成
	position := &model.Position{
		Title:             req.PositionTitle,
		Level:             req.Level,
		Responsibilities:  req.Responsibilities,
		ReportsTo:         req.ReportsTo,
		DepartmentID:      req.DepartmentID,
		PersonaTemplateID: req.TemplateID,
	}

	// 生成人设
	var profile *persona.PersonaProfile
	var err error

	if s.personaGen != nil {
		profile, err = s.personaGen.GenerateForPosition(position)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to generate persona: "+err.Error())
			return
		}
	} else {
		writeError(w, http.StatusInternalServerError, "persona generator not available")
		return
	}

	// 返回生成结果
	response := map[string]interface{}{
		"position_title":      req.PositionTitle,
		"level":               req.Level,
		"system_prompt":       profile.SystemPrompt,
		"background":          profile.Background,
		"identity":            profile.Identity,
		"has_workspace_files": len(profile.WorkspaceFiles) > 0,
		"workspace_files":     profile.WorkspaceFiles,
		"generated_at":        time.Now(),
	}

	writeJSON(w, http.StatusOK, response)
}

// ═══════════════════════════════════════════════════════════
// Agent Persona Management API
// ═══════════════════════════════════════════════════════════

func (s *Server) handleAgentPersona(w http.ResponseWriter, r *http.Request, agentID string) {
	switch r.Method {
	case http.MethodGet:
		// Get agent persona status
		agent, err := s.store.GetAgentInstance(agentID)
		if err != nil {
			writeNotFound(w, "agent not found")
			return
		}

		// Get assignments to find position
		assignments, _ := s.store.ListAssignmentsByAgent(agentID)
		var positionID string
		if len(assignments) > 0 {
			positionID = assignments[0].PositionID
		}

		response := map[string]interface{}{
			"agent_id":      agentID,
			"agent_name":    agent.Name,
			"position_id":   positionID,
			"system_prompt": agent.Config.SystemPrompt,
			"background":    agent.Config.Description,
			"persona_name":  agent.Config.PersonaName,
			"last_updated":  agent.UpdatedAt,
		}

		// If position exists, get position persona for comparison
		if positionID != "" {
			position, err := s.store.GetPosition(positionID)
			if err == nil {
				response["position_system_prompt"] = position.SystemPrompt
				response["position_background"] = position.Background
				response["position_title"] = position.Title

				// Check if in sync
				inSync := agent.Config.SystemPrompt == position.SystemPrompt &&
					agent.Config.Description == position.Background
				response["in_sync"] = inSync
			}
		}

		writeJSON(w, http.StatusOK, response)

	case http.MethodPost:
		// Apply persona to agent
		var req struct {
			PositionID   string `json:"position_id,omitempty"`
			SystemPrompt string `json:"system_prompt,omitempty"`
			Background   string `json:"background,omitempty"`
			Force        bool   `json:"force,omitempty"`
		}

		if !decodeJSON(w, r, &req) {
			return
		}

		// 使用现有的applyPersonaToAgent方法
		var systemPrompt, background string
		if req.PositionID != "" {
			var err error
			systemPrompt, background, err = s.store.GetPersonaForPosition(req.PositionID)
			if err != nil {
				writeError(w, http.StatusBadRequest, "failed to get persona from position: "+err.Error())
				return
			}
		} else if req.SystemPrompt != "" || req.Background != "" {
			systemPrompt = req.SystemPrompt
			background = req.Background
		} else {
			writeBadRequest(w, "must provide position_id or system_prompt/background")
			return
		}

		result, err := s.applyPersonaToAgent(agentID, systemPrompt, background, "manual_api")
		if err != nil {
			writeServerError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, result)

	case http.MethodPut:
		// Reload persona (trigger re-initialization)
		// This would trigger the node to reload persona files
		_, err := s.store.GetAgentInstance(agentID)
		if err != nil {
			writeNotFound(w, "agent not found")
			return
		}

		// TODO: Implement actual reload logic
		// For now, just return a placeholder response
		response := map[string]interface{}{
			"agent_id":  agentID,
			"action":    "reload_triggered",
			"status":    "pending",
			"message":   "Reload requested, implementation needed",
			"timestamp": time.Now(),
		}

		writeJSON(w, http.StatusOK, response)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// ═══════════════════════════════════════════════════════════
// Persona Rollback API
// ═══════════════════════════════════════════════════════════

// handlePersonaRollback handles persona rollback to a specific snapshot
func (s *Server) handlePersonaRollback(w http.ResponseWriter, r *http.Request, agentID, snapshotID string) {
	switch r.Method {
	case http.MethodPost:
		// 执行回滚
		agent, err := s.store.RollbackPersonaSnapshot(agentID, snapshotID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "rollback failed: "+err.Error())
			return
		}

		response := map[string]interface{}{
			"agent_id":      agentID,
			"snapshot_id":   snapshotID,
			"status":        "rolled_back",
			"system_prompt": agent.Config.SystemPrompt,
			"background":    agent.Config.Description,
			"timestamp":     time.Now(),
		}
		writeJSON(w, http.StatusOK, response)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handlePersonaRollbackList handles listing rollbackable snapshots
func (s *Server) handlePersonaRollbackList(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// 获取可回滚的快照列表
	snapshots, err := s.store.ListPersonaSnapshots(agentID, 10)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list snapshots: "+err.Error())
		return
	}

	type snapshotInfo struct {
		ID           string    `json:"id"`
		Version      int       `json:"version"`
		Reason       string    `json:"reason"`
		CreatedAt    time.Time `json:"created_at"`
		SystemPrompt string    `json:"system_prompt_preview,omitempty"`
	}

	var snapshotList []snapshotInfo
	for _, snap := range snapshots {
		preview := snap.SystemPrompt
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		snapshotList = append(snapshotList, snapshotInfo{
			ID:           snap.ID,
			Version:      snap.Version,
			Reason:       snap.Reason,
			CreatedAt:    snap.CreatedAt,
			SystemPrompt: preview,
		})
	}

	response := map[string]interface{}{
		"agent_id":  agentID,
		"snapshots": snapshotList,
	}
	writeJSON(w, http.StatusOK, response)
}

// handlePersonaSnapshots lists snapshots for an agent
func (s *Server) handlePersonaSnapshots(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	snapshots, err := s.store.ListPersonaSnapshots(agentID, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list snapshots: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, snapshots)
}
