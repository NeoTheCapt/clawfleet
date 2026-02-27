package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ═══════════════════════════════════════════════════════════
// Persona Snapshots
// ═══════════════════════════════════════════════════════════

// CreatePersonaSnapshot 创建人设快照
func (s *Store) CreatePersonaSnapshot(p *model.PersonaSnapshot) error {
	_, err := s.db.Exec(`INSERT INTO persona_snapshots 
		(id, agent_id, position_id, system_prompt, background, persona_name, files_json, version, reason, created_at) 
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.AgentID, p.PositionID, p.SystemPrompt, p.Background,
		p.PersonaName, p.FilesJSON, p.Version, p.Reason, p.CreatedAt)
	return err
}

// GetPersonaSnapshot 获取人设快照
func (s *Store) GetPersonaSnapshot(id string) (*model.PersonaSnapshot, error) {
	var p model.PersonaSnapshot
	err := s.db.QueryRow(`SELECT id, agent_id, position_id, system_prompt, background, 
		persona_name, files_json, version, reason, created_at 
		FROM persona_snapshots WHERE id=?`, id).
		Scan(&p.ID, &p.AgentID, &p.PositionID, &p.SystemPrompt, &p.Background,
			&p.PersonaName, &p.FilesJSON, &p.Version, &p.Reason, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListPersonaSnapshots 获取 Agent 的所有快照
func (s *Store) ListPersonaSnapshots(agentID string, limit int) ([]*model.PersonaSnapshot, error) {
	query := `SELECT id, agent_id, position_id, system_prompt, background, 
		persona_name, files_json, version, reason, created_at 
		FROM persona_snapshots WHERE agent_id=? ORDER BY version DESC`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	return queryRows(s.db, query, []interface{}{agentID}, func(r *sql.Rows) (*model.PersonaSnapshot, error) {
		var p model.PersonaSnapshot
		if err := r.Scan(&p.ID, &p.AgentID, &p.PositionID, &p.SystemPrompt, &p.Background,
			&p.PersonaName, &p.FilesJSON, &p.Version, &p.Reason, &p.CreatedAt); err != nil {
			return nil, err
		}
		return &p, nil
	})
}

// GetLatestPersonaSnapshot 获取最新的快照
func (s *Store) GetLatestPersonaSnapshot(agentID string) (*model.PersonaSnapshot, error) {
	var p model.PersonaSnapshot
	err := s.db.QueryRow(`SELECT id, agent_id, position_id, system_prompt, background, 
		persona_name, files_json, version, reason, created_at 
		FROM persona_snapshots WHERE agent_id=? ORDER BY version DESC LIMIT 1`, agentID).
		Scan(&p.ID, &p.AgentID, &p.PositionID, &p.SystemPrompt, &p.Background,
			&p.PersonaName, &p.FilesJSON, &p.Version, &p.Reason, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetNextSnapshotVersion 获取下一个版本号
func (s *Store) GetNextSnapshotVersion(agentID string) (int, error) {
	var version int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) + 1 FROM persona_snapshots WHERE agent_id=?`, agentID).
		Scan(&version)
	return version, err
}

// DeletePersonaSnapshotsOlderThan 删除旧快照（保留最近 N 个版本）
func (s *Store) DeletePersonaSnapshotsOlderThan(agentID string, keepCount int) error {
	_, err := s.db.Exec(`DELETE FROM persona_snapshots 
		WHERE agent_id=? AND id NOT IN (
			SELECT id FROM persona_snapshots WHERE agent_id=? 
			ORDER BY version DESC LIMIT ?
		)`, agentID, agentID, keepCount)
	return err
}

// ═══════════════════════════════════════════════════════════
// Agent Persona Status
// ═══════════════════════════════════════════════════════════

// UpsertAgentPersonaStatus 更新或创建 Agent 人设状态
func (s *Store) UpsertAgentPersonaStatus(status *model.AgentPersonaStatus) error {
	status.UpdatedAt = time.Now()
	_, err := s.db.Exec(`INSERT OR REPLACE INTO agent_persona_status 
		(agent_id, position_id, current_snapshot_id, last_sync_at, sync_status, 
		custom_modified, conflict_resolution, created_at, updated_at) 
		VALUES (?,?,?,?,?,?,?,?,?)`,
		status.AgentID, status.PositionID, status.CurrentSnapshotID, status.LastSyncAt,
		status.SyncStatus, status.CustomModified, status.ConflictResolution,
		status.CreatedAt, status.UpdatedAt)
	return err
}

// GetAgentPersonaStatus 获取 Agent 人设状态
func (s *Store) GetAgentPersonaStatus(agentID string) (*model.AgentPersonaStatus, error) {
	var status model.AgentPersonaStatus
	err := s.db.QueryRow(`SELECT agent_id, position_id, current_snapshot_id, last_sync_at, 
		sync_status, custom_modified, conflict_resolution, created_at, updated_at 
		FROM agent_persona_status WHERE agent_id=?`, agentID).
		Scan(&status.AgentID, &status.PositionID, &status.CurrentSnapshotID, &status.LastSyncAt,
			&status.SyncStatus, &status.CustomModified, &status.ConflictResolution,
			&status.CreatedAt, &status.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

// UpdateAgentPersonaSyncStatus 更新同步状态
func (s *Store) UpdateAgentPersonaSyncStatus(agentID, syncStatus string) error {
	_, err := s.db.Exec(`UPDATE agent_persona_status 
		SET sync_status=?, last_sync_at=?, updated_at=? 
		WHERE agent_id=?`, syncStatus, time.Now(), time.Now(), agentID)
	return err
}

// SetAgentPersonaCustomModified 标记 Agent 人设已自定义修改
func (s *Store) SetAgentPersonaCustomModified(agentID string, modified bool) error {
	_, err := s.db.Exec(`UPDATE agent_persona_status 
		SET custom_modified=?, updated_at=? 
		WHERE agent_id=?`, modified, time.Now(), agentID)
	return err
}

// SetAgentPersonaConflictResolution 设置冲突解决策略
func (s *Store) SetAgentPersonaConflictResolution(agentID, resolution string) error {
	_, err := s.db.Exec(`UPDATE agent_persona_status 
		SET conflict_resolution=?, updated_at=? 
		WHERE agent_id=?`, resolution, time.Now(), agentID)
	return err
}

// ═══════════════════════════════════════════════════════════
// Helper Functions
// ═══════════════════════════════════════════════════════════

// CreatePersonaSnapshotForAgent 为 Agent 创建快照
func (s *Store) CreatePersonaSnapshotForAgent(agent *model.AgentInstance, positionID, reason string) (*model.PersonaSnapshot, error) {
	version, err := s.GetNextSnapshotVersion(agent.ID)
	if err != nil {
		return nil, err
	}

	// 获取现有文件（如果有）
	filesMap := map[string]string{}
	if agent.Config.SystemPrompt != "" {
		filesMap["SOUL.md"] = agent.Config.SystemPrompt
	}
	if agent.Config.Description != "" {
		filesMap["BACKGROUND.md"] = agent.Config.Description
	}
	filesJSONBytes, _ := json.Marshal(filesMap)
	filesJSON := string(filesJSONBytes)

	snapshot := &model.PersonaSnapshot{
		ID:           fmt.Sprintf("psnap_%s_v%d", agent.ID, version),
		AgentID:      agent.ID,
		PositionID:   positionID,
		SystemPrompt: agent.Config.SystemPrompt,
		Background:   agent.Config.Description,
		PersonaName:  agent.Config.PersonaName,
		FilesJSON:    filesJSON,
		Version:      version,
		Reason:       reason,
		CreatedAt:    time.Now(),
	}

	if err := s.CreatePersonaSnapshot(snapshot); err != nil {
		return nil, err
	}

	// 更新当前快照 ID
	status, err := s.GetAgentPersonaStatus(agent.ID)
	if err == nil {
		status.CurrentSnapshotID = snapshot.ID
		status.LastSyncAt = time.Now()
		status.SyncStatus = "synced"
		if err := s.UpsertAgentPersonaStatus(status); err != nil {
			return nil, fmt.Errorf("update agent persona status: %w", err)
		}
	} else {
		// 创建新状态
		newStatus := &model.AgentPersonaStatus{
			AgentID:           agent.ID,
			PositionID:        positionID,
			CurrentSnapshotID: snapshot.ID,
			LastSyncAt:        time.Now(),
			SyncStatus:        "synced",
			CustomModified:    false,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}
		if err := s.UpsertAgentPersonaStatus(newStatus); err != nil {
			return nil, fmt.Errorf("create agent persona status: %w", err)
		}
	}

	return snapshot, nil
}

// RollbackPersonaSnapshot 回滚到指定快照
func (s *Store) RollbackPersonaSnapshot(agentID, snapshotID string) (*model.AgentInstance, error) {
	// 获取快照
	snapshot, err := s.GetPersonaSnapshot(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}

	// 获取 Agent
	agent, err := s.GetAgentInstance(agentID)
	if err != nil {
		return nil, err
	}

	// 保存当前状态为新快照（以便保留回滚点）
	if _, err := s.CreatePersonaSnapshotForAgent(agent, snapshot.PositionID, "rollback_before"); err != nil {
		return nil, fmt.Errorf("create rollback snapshot: %w", err)
	}

	// 应用快照内容到 Agent
	agent.Config.SystemPrompt = snapshot.SystemPrompt
	agent.Config.Description = snapshot.Background
	if snapshot.PersonaName != "" {
		agent.Config.PersonaName = snapshot.PersonaName
	}
	agent.UpdatedAt = time.Now()

	// 更新 Agent
	if err := s.UpdateAgentInstance(agent); err != nil {
		return nil, err
	}

	// 更新状态
	status, err := s.GetAgentPersonaStatus(agentID)
	if err == nil {
		status.CurrentSnapshotID = snapshotID
		status.SyncStatus = "synced"
		status.CustomModified = false
		status.ConflictResolution = ""
		status.LastSyncAt = time.Now()
		status.UpdatedAt = time.Now()
		if err := s.UpsertAgentPersonaStatus(status); err != nil {
			return nil, fmt.Errorf("update agent persona status after rollback: %w", err)
		}
	}

	return agent, nil
}
