package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ═══════════════════════════════════════════════════════════
// IM Group CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateIMGroup(g *model.IMGroup) error {
	members, _ := json.Marshal(g.Members)
	extCh, _ := json.Marshal(g.ExternalChannel)
	_, err := s.db.Exec(`INSERT INTO im_groups (id,company_id,name,department_id,type,members,external_channel,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		g.ID, g.CompanyID, g.Name, g.DepartmentID, g.Type, string(members), string(extCh), g.CreatedAt, g.UpdatedAt)
	return err
}

func (s *Store) ListIMGroups(companyID string) ([]*model.IMGroup, error) {
	return queryIMGroups(s.db, `SELECT id,company_id,name,department_id,type,members,external_channel,created_at,updated_at FROM im_groups WHERE company_id=? ORDER BY type,created_at`, companyID)
}

func (s *Store) UpdateIMGroup(g *model.IMGroup) error {
	members, _ := json.Marshal(g.Members)
	extCh, _ := json.Marshal(g.ExternalChannel)
	_, err := s.db.Exec(`UPDATE im_groups SET name=?,department_id=?,type=?,members=?,external_channel=?,updated_at=? WHERE id=?`,
		g.Name, g.DepartmentID, g.Type, string(members), string(extCh), time.Now(), g.ID)
	return err
}

func (s *Store) DeleteIMGroup(id string) error {
	_, err := s.db.Exec(`DELETE FROM im_groups WHERE id=?`, id)
	return err
}

func scanIMGroup(row scannable) (*model.IMGroup, error) {
	var g model.IMGroup
	var members, extCh string
	if err := row.Scan(&g.ID, &g.CompanyID, &g.Name, &g.DepartmentID, &g.Type, &members, &extCh, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return nil, err
	}
	if strings.TrimSpace(members) != "" {
		if err := json.Unmarshal([]byte(members), &g.Members); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(extCh) != "" {
		if err := json.Unmarshal([]byte(extCh), &g.ExternalChannel); err != nil {
			return nil, err
		}
	}
	return &g, nil
}

func queryIMGroups(db *sql.DB, query string, args ...interface{}) ([]*model.IMGroup, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.IMGroup, error) { return scanIMGroup(r) })
}

// ═══════════════════════════════════════════════════════════
// Workflow CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateWorkflow(w *model.Workflow) error {
	steps, _ := json.Marshal(w.Steps)
	triggers, _ := json.Marshal(w.Triggers)
	_, err := s.db.Exec(`INSERT INTO workflows (id,company_id,name,position_title,steps,triggers,output_requirements,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		w.ID, w.CompanyID, w.Name, w.PositionTitle, string(steps), string(triggers), w.OutputRequirements, w.CreatedAt, w.UpdatedAt)
	return err
}

func (s *Store) ListWorkflows(companyID string) ([]*model.Workflow, error) {
	return queryWorkflows(s.db, `SELECT id,company_id,name,position_title,steps,triggers,output_requirements,created_at,updated_at FROM workflows WHERE company_id=? ORDER BY created_at`, companyID)
}

func (s *Store) UpdateWorkflow(w *model.Workflow) error {
	steps, _ := json.Marshal(w.Steps)
	triggers, _ := json.Marshal(w.Triggers)
	_, err := s.db.Exec(`UPDATE workflows SET name=?,position_title=?,steps=?,triggers=?,output_requirements=?,updated_at=? WHERE id=?`,
		w.Name, w.PositionTitle, string(steps), string(triggers), w.OutputRequirements, time.Now(), w.ID)
	return err
}

func (s *Store) DeleteWorkflow(id string) error {
	_, err := s.db.Exec(`DELETE FROM workflows WHERE id=?`, id)
	return err
}

func scanWorkflow(row scannable) (*model.Workflow, error) {
	var w model.Workflow
	var steps, triggers string
	if err := row.Scan(&w.ID, &w.CompanyID, &w.Name, &w.PositionTitle, &steps, &triggers, &w.OutputRequirements, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return nil, err
	}
	if strings.TrimSpace(steps) != "" {
		if err := json.Unmarshal([]byte(steps), &w.Steps); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(triggers) != "" {
		if err := json.Unmarshal([]byte(triggers), &w.Triggers); err != nil {
			return nil, err
		}
	}
	return &w, nil
}

func queryWorkflows(db *sql.DB, query string, args ...interface{}) ([]*model.Workflow, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.Workflow, error) { return scanWorkflow(r) })
}

// ═══════════════════════════════════════════════════════════
// Governance Log
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateGovernanceLog(g *model.GovernanceLog) error {
	_, err := s.db.Exec(`INSERT INTO governance_logs (id,company_id,action,actor_id,detail,timestamp) VALUES (?,?,?,?,?,?)`,
		g.ID, g.CompanyID, g.Action, g.ActorID, g.Detail, g.Timestamp)
	return err
}

func scanGovernanceLog(row scannable) (*model.GovernanceLog, error) {
	var g model.GovernanceLog
	if err := row.Scan(&g.ID, &g.CompanyID, &g.Action, &g.ActorID, &g.Detail, &g.Timestamp); err != nil {
		return nil, err
	}
	return &g, nil
}

func queryGovernanceLogs(db *sql.DB, query string, args ...interface{}) ([]*model.GovernanceLog, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.GovernanceLog, error) { return scanGovernanceLog(r) })
}

func (s *Store) ListGovernanceLogs(companyID string, limit int) ([]*model.GovernanceLog, error) {
	return queryGovernanceLogs(s.db, `SELECT id,company_id,action,actor_id,detail,timestamp FROM governance_logs WHERE company_id=? ORDER BY timestamp DESC LIMIT ?`, companyID, limit)
}
