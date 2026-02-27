package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ═══════════════════════════════════════════════════════════
// Agent Instance CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateAgentInstance(a *model.AgentInstance) error {
	config, _ := json.Marshal(a.Config)
	_, err := s.db.Exec(
		`INSERT INTO agent_instances (id,name,company_id,node_id,agent_type,role,status,container_id,deploy_mode,config,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.Name, a.CompanyID, a.NodeID, a.AgentType, a.Role, a.Status, a.ContainerID, a.DeployMode, string(config), a.CreatedAt, a.UpdatedAt)
	return err
}

func (s *Store) GetAgentInstance(id string) (*model.AgentInstance, error) {
	return scanAgent(s.db.QueryRow(`SELECT id,name,company_id,node_id,agent_type,role,status,container_id,deploy_mode,config,created_at,updated_at FROM agent_instances WHERE id=?`, id))
}

func (s *Store) ListAgentInstances() ([]*model.AgentInstance, error) {
	return queryAgents(s.db, `SELECT id,name,company_id,node_id,agent_type,role,status,container_id,deploy_mode,config,created_at,updated_at FROM agent_instances ORDER BY created_at DESC`)
}

func (s *Store) ListAgentsByNode(nodeID string) ([]*model.AgentInstance, error) {
	return queryAgents(s.db, `SELECT id,name,company_id,node_id,agent_type,role,status,container_id,deploy_mode,config,created_at,updated_at FROM agent_instances WHERE node_id=? ORDER BY created_at DESC`, nodeID)
}

func (s *Store) ListAgentsByCompany(companyID string) ([]*model.AgentInstance, error) {
	return queryAgents(s.db, `SELECT id,name,company_id,node_id,agent_type,role,status,container_id,deploy_mode,config,created_at,updated_at FROM agent_instances WHERE company_id=? ORDER BY created_at DESC`, companyID)
}

func (s *Store) UpdateAgentStatus(id string, status model.AgentStatus, containerID string) error {
	_, err := s.db.Exec(`UPDATE agent_instances SET status=?,container_id=?,updated_at=? WHERE id=?`, status, containerID, time.Now(), id)
	return err
}

func (s *Store) UpdateAgentInstance(a *model.AgentInstance) error {
	config, _ := json.Marshal(a.Config)
	_, err := s.db.Exec(`UPDATE agent_instances SET name=?,role=?,config=?,status=?,updated_at=? WHERE id=?`,
		a.Name, a.Role, string(config), a.Status, a.UpdatedAt, a.ID)
	return err
}

func (s *Store) DeleteAgentInstance(id string) error {
	// 级联删除相关的 agent 记录
	if _, err := s.db.Exec(`DELETE FROM agent_assignments WHERE agent_id=?`, id); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM persona_sync_history WHERE agent_id=?`, id); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM persona_snapshots WHERE agent_id=?`, id); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM agent_persona_status WHERE agent_id=?`, id); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM kpi_reviews WHERE agent_id=?`, id); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM agent_instances WHERE id=?`, id)
	return err
}

func scanAgent(row scannable) (*model.AgentInstance, error) {
	var a model.AgentInstance
	var config string
	if err := row.Scan(&a.ID, &a.Name, &a.CompanyID, &a.NodeID, &a.AgentType, &a.Role, &a.Status, &a.ContainerID, &a.DeployMode, &config, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	if strings.TrimSpace(config) != "" {
		if err := json.Unmarshal([]byte(config), &a.Config); err != nil {
			return nil, fmt.Errorf("decode agent config: %w", err)
		}
	}
	return &a, nil
}

func queryAgents(db *sql.DB, query string, args ...interface{}) ([]*model.AgentInstance, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.AgentInstance, error) { return scanAgent(r) })
}
