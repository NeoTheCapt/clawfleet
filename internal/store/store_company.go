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
// Company CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateCompany(c *model.Company) error {
	_, err := s.db.Exec(`INSERT INTO companies (id,name,vision,mission,owner_id,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		c.ID, c.Name, c.Vision, c.Mission, c.OwnerID, c.Status, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *Store) GetCompany(id string) (*model.Company, error) {
	var c model.Company
	err := s.db.QueryRow(`SELECT id,name,vision,mission,owner_id,status,created_at,updated_at FROM companies WHERE id=?`, id).
		Scan(&c.ID, &c.Name, &c.Vision, &c.Mission, &c.OwnerID, &c.Status, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) ListCompanies() ([]*model.Company, error) {
	return queryRows(s.db,
		`SELECT id,name,vision,mission,owner_id,status,created_at,updated_at FROM companies ORDER BY created_at DESC`,
		nil,
		func(r *sql.Rows) (*model.Company, error) {
			var c model.Company
			if err := r.Scan(&c.ID, &c.Name, &c.Vision, &c.Mission, &c.OwnerID, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
				return nil, err
			}
			return &c, nil
		},
	)
}

func (s *Store) UpdateCompany(c *model.Company) error {
	_, err := s.db.Exec(`UPDATE companies SET name=?,vision=?,mission=?,owner_id=?,status=?,updated_at=? WHERE id=?`,
		c.Name, c.Vision, c.Mission, c.OwnerID, c.Status, time.Now(), c.ID)
	return err
}

func (s *Store) DeleteCompany(id string) error {
	_, err := s.db.Exec(`DELETE FROM companies WHERE id=?`, id)
	return err
}

// ═══════════════════════════════════════════════════════════
// Company Goal CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateGoal(g *model.CompanyGoal) error {
	_, err := s.db.Exec(`INSERT INTO company_goals (id,company_id,title,description,priority,status,owner_position_id,deadline,parent_goal_id,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		g.ID, g.CompanyID, g.Title, g.Description, g.Priority, g.Status, g.OwnerPositionID, g.Deadline, g.ParentGoalID, g.CreatedAt, g.UpdatedAt)
	return err
}

func (s *Store) ListGoals(companyID string) ([]*model.CompanyGoal, error) {
	return queryGoals(s.db, `SELECT id,company_id,title,description,priority,status,owner_position_id,deadline,parent_goal_id,created_at,updated_at FROM company_goals WHERE company_id=? ORDER BY priority,created_at`, companyID)
}

func (s *Store) UpdateGoal(g *model.CompanyGoal) error {
	_, err := s.db.Exec(`UPDATE company_goals SET title=?,description=?,priority=?,status=?,owner_position_id=?,deadline=?,parent_goal_id=?,updated_at=? WHERE id=?`,
		g.Title, g.Description, g.Priority, g.Status, g.OwnerPositionID, g.Deadline, g.ParentGoalID, time.Now(), g.ID)
	return err
}

func (s *Store) DeleteGoal(id string) error {
	_, err := s.db.Exec(`DELETE FROM company_goals WHERE id=?`, id)
	return err
}

func scanGoal(row scannable) (*model.CompanyGoal, error) {
	var g model.CompanyGoal
	if err := row.Scan(&g.ID, &g.CompanyID, &g.Title, &g.Description, &g.Priority, &g.Status, &g.OwnerPositionID, &g.Deadline, &g.ParentGoalID, &g.CreatedAt, &g.UpdatedAt); err != nil {
		return nil, err
	}
	return &g, nil
}

func queryGoals(db *sql.DB, query string, args ...interface{}) ([]*model.CompanyGoal, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.CompanyGoal, error) { return scanGoal(r) })
}

// ═══════════════════════════════════════════════════════════
// Department CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateDepartment(d *model.Department) error {
	_, err := s.db.Exec(`INSERT INTO departments (id,company_id,name,parent_id,type,description,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		d.ID, d.CompanyID, d.Name, d.ParentID, d.Type, d.Description, d.CreatedAt, d.UpdatedAt)
	return err
}

func (s *Store) ListDepartments(companyID string) ([]*model.Department, error) {
	return queryDepartments(s.db, `SELECT id,company_id,name,parent_id,type,description,created_at,updated_at FROM departments WHERE company_id=? ORDER BY created_at`, companyID)
}

func (s *Store) UpdateDepartment(d *model.Department) error {
	_, err := s.db.Exec(`UPDATE departments SET name=?,parent_id=?,type=?,description=?,updated_at=? WHERE id=?`,
		d.Name, d.ParentID, d.Type, d.Description, time.Now(), d.ID)
	return err
}

func (s *Store) DeleteDepartment(id string) error {
	_, err := s.db.Exec(`DELETE FROM departments WHERE id=?`, id)
	return err
}

func scanDepartment(row scannable) (*model.Department, error) {
	var d model.Department
	if err := row.Scan(&d.ID, &d.CompanyID, &d.Name, &d.ParentID, &d.Type, &d.Description, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, err
	}
	return &d, nil
}

func queryDepartments(db *sql.DB, query string, args ...interface{}) ([]*model.Department, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.Department, error) { return scanDepartment(r) })
}

// ═══════════════════════════════════════════════════════════
// Position CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreatePosition(p *model.Position) error {
	_, err := s.db.Exec(`INSERT INTO positions (id,department_id,title,level,reports_to,responsibilities,system_prompt,background,persona_template_id,workflow_id,telegram_bot_id,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.DepartmentID, p.Title, p.Level, p.ReportsTo, p.Responsibilities, p.SystemPrompt, p.Background, p.PersonaTemplateID, p.WorkflowID, p.TelegramBotID, p.CreatedAt, p.UpdatedAt)
	return err
}

func (s *Store) GetPosition(id string) (*model.Position, error) {
	return scanPosition(s.db.QueryRow(`SELECT id,department_id,title,level,reports_to,responsibilities,system_prompt,background,persona_template_id,workflow_id,telegram_bot_id,created_at,updated_at FROM positions WHERE id=?`, id))
}

func (s *Store) ListPositions(departmentID string) ([]*model.Position, error) {
	return queryPositions(s.db, `SELECT id,department_id,title,level,reports_to,responsibilities,system_prompt,background,persona_template_id,workflow_id,telegram_bot_id,created_at,updated_at FROM positions WHERE department_id=? ORDER BY level,created_at`, departmentID)
}

// ListPositionsByTemplate returns all positions using a persona template
func (s *Store) ListPositionsByTemplate(templateID string) ([]*model.Position, error) {
	return queryPositions(s.db, `SELECT id,department_id,title,level,reports_to,responsibilities,system_prompt,background,persona_template_id,workflow_id,telegram_bot_id,created_at,updated_at FROM positions WHERE persona_template_id=? ORDER BY level,created_at`, templateID)
}

func (s *Store) ListAllPositions(companyID string) ([]*model.Position, error) {
	return queryPositions(s.db, `SELECT p.id,p.department_id,p.title,p.level,p.reports_to,p.responsibilities,p.system_prompt,p.background,p.persona_template_id,p.workflow_id,p.telegram_bot_id,p.created_at,p.updated_at FROM positions p JOIN departments d ON p.department_id=d.id WHERE d.company_id=? ORDER BY p.level,p.created_at`, companyID)
}

func scanPosition(row scannable) (*model.Position, error) {
	var p model.Position
	if err := row.Scan(&p.ID, &p.DepartmentID, &p.Title, &p.Level, &p.ReportsTo, &p.Responsibilities, &p.SystemPrompt, &p.Background, &p.PersonaTemplateID, &p.WorkflowID, &p.TelegramBotID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func queryPositions(db *sql.DB, query string, args ...interface{}) ([]*model.Position, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.Position, error) { return scanPosition(r) })
}

func (s *Store) UpdatePosition(p *model.Position) error {
	_, err := s.db.Exec(`UPDATE positions SET title=?,level=?,reports_to=?,responsibilities=?,system_prompt=?,background=?,persona_template_id=?,workflow_id=?,telegram_bot_id=?,updated_at=? WHERE id=?`,
		p.Title, p.Level, p.ReportsTo, p.Responsibilities, p.SystemPrompt, p.Background, p.PersonaTemplateID, p.WorkflowID, p.TelegramBotID, time.Now(), p.ID)
	return err
}

func (s *Store) DeletePosition(id string) error {
	_, err := s.db.Exec(`DELETE FROM positions WHERE id=?`, id)
	return err
}

// GetCompanyByPositionID returns the company ID for a given position
func (s *Store) GetCompanyByPositionID(positionID string) (string, error) {
	var companyID string
	err := s.db.QueryRow(`SELECT d.company_id FROM positions p JOIN departments d ON p.department_id=d.id WHERE p.id=?`, positionID).
		Scan(&companyID)
	if err != nil {
		return "", err
	}
	return companyID, nil
}

// ═══════════════════════════════════════════════════════════
// Persona Template CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreatePersonaTemplate(pt *model.PersonaTemplate) error {
	tags, _ := json.Marshal(pt.Tags)
	_, err := s.db.Exec(`INSERT INTO persona_templates (id,name,description,system_prompt,background,tags,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		pt.ID, pt.Name, pt.Description, pt.SystemPrompt, pt.Background, string(tags), pt.CreatedAt, pt.UpdatedAt)
	return err
}

func (s *Store) GetPersonaTemplate(id string) (*model.PersonaTemplate, error) {
	var pt model.PersonaTemplate
	var tags string
	err := s.db.QueryRow(`SELECT id,name,description,system_prompt,background,tags,created_at,updated_at FROM persona_templates WHERE id=?`, id).
		Scan(&pt.ID, &pt.Name, &pt.Description, &pt.SystemPrompt, &pt.Background, &tags, &pt.CreatedAt, &pt.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(tags) != "" {
		if err := json.Unmarshal([]byte(tags), &pt.Tags); err != nil {
			return nil, err
		}
	}
	return &pt, nil
}

func (s *Store) ListPersonaTemplates() ([]*model.PersonaTemplate, error) {
	return queryRows(s.db,
		`SELECT id,name,description,system_prompt,background,tags,created_at,updated_at FROM persona_templates ORDER BY created_at DESC`,
		nil,
		func(r *sql.Rows) (*model.PersonaTemplate, error) {
			var pt model.PersonaTemplate
			var tags string
			if err := r.Scan(&pt.ID, &pt.Name, &pt.Description, &pt.SystemPrompt, &pt.Background, &tags, &pt.CreatedAt, &pt.UpdatedAt); err != nil {
				return nil, err
			}
			if strings.TrimSpace(tags) != "" {
				if err := json.Unmarshal([]byte(tags), &pt.Tags); err != nil {
					return nil, err
				}
			}
			return &pt, nil
		},
	)
}

func (s *Store) UpdatePersonaTemplate(pt *model.PersonaTemplate) error {
	tags, _ := json.Marshal(pt.Tags)
	_, err := s.db.Exec(`UPDATE persona_templates SET name=?,description=?,system_prompt=?,background=?,tags=?,updated_at=? WHERE id=?`,
		pt.Name, pt.Description, pt.SystemPrompt, pt.Background, string(tags), time.Now(), pt.ID)
	return err
}

func (s *Store) DeletePersonaTemplate(id string) error {
	_, err := s.db.Exec(`DELETE FROM persona_templates WHERE id=?`, id)
	return err
}

// GetPersonaForPosition returns the effective persona (system_prompt, background) for a position
// If position has persona_template_id, fetch from template; otherwise use position's own fields
func (s *Store) GetPersonaForPosition(positionID string) (systemPrompt, background string, err error) {
	var p model.Position
	var templateID string
	err = s.db.QueryRow(`SELECT system_prompt,background,persona_template_id FROM positions WHERE id=?`, positionID).
		Scan(&p.SystemPrompt, &p.Background, &templateID)
	if err != nil {
		return "", "", err
	}

	// If position has custom fields, use them
	if p.SystemPrompt != "" || p.Background != "" {
		return p.SystemPrompt, p.Background, nil
	}

	// If no custom fields but has template, fetch from template
	if templateID != "" {
		pt, err := s.GetPersonaTemplate(templateID)
		if err != nil {
			return "", "", err
		}
		return pt.SystemPrompt, pt.Background, nil
	}

	// No persona configured
	return "", "", nil
}

// ListAssignmentsByPosition returns all assignments for a specific position
func (s *Store) ListAssignmentsByPosition(positionID string) ([]*model.AgentAssignment, error) {
	return queryAssignments(s.db, `SELECT id,agent_id,position_id,status,created_at,updated_at FROM agent_assignments WHERE position_id=? ORDER BY created_at`, positionID)
}

// ListAssignmentsByAgent returns all assignments for a specific agent
func (s *Store) ListAssignmentsByAgent(agentID string) ([]*model.AgentAssignment, error) {
	return queryAssignments(s.db, `SELECT id,agent_id,position_id,status,created_at,updated_at FROM agent_assignments WHERE agent_id=? ORDER BY created_at`, agentID)
}

// CreatePersonaSyncRecord creates a new persona sync history record
func (s *Store) CreatePersonaSyncRecord(record *model.PersonaSyncRecord) error {
	_, err := s.db.Exec(`INSERT INTO persona_sync_history (id,position_id,agent_id,action,old_system_prompt,new_system_prompt,old_background,new_background,status,error,synced_at) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		record.ID, record.PositionID, record.AgentID, record.Action, record.OldSystemPrompt, record.NewSystemPrompt,
		record.OldBackground, record.NewBackground, record.Status, record.Error, record.SyncedAt)
	return err
}

// ListPersonaSyncHistory returns persona sync history for a position
func (s *Store) ListPersonaSyncHistory(positionID string, limit int) ([]*model.PersonaSyncRecord, error) {
	query := `SELECT id,position_id,agent_id,action,old_system_prompt,new_system_prompt,old_background,new_background,status,error,synced_at FROM persona_sync_history WHERE position_id=? ORDER BY synced_at DESC`
	if limit > 0 {
		query += ` LIMIT ` + fmt.Sprintf("%d", limit)
	}
	rows, err := s.db.Query(query, positionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.PersonaSyncRecord
	for rows.Next() {
		var r model.PersonaSyncRecord
		if err := rows.Scan(&r.ID, &r.PositionID, &r.AgentID, &r.Action, &r.OldSystemPrompt, &r.NewSystemPrompt,
			&r.OldBackground, &r.NewBackground, &r.Status, &r.Error, &r.SyncedAt); err != nil {
			return nil, err
		}
		out = append(out, &r)
	}
	return out, nil
}

// ═══════════════════════════════════════════════════════════
// Agent Assignment CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateAssignment(a *model.AgentAssignment) error {
	_, err := s.db.Exec(`INSERT INTO agent_assignments (id,agent_id,position_id,status,created_at,updated_at) VALUES (?,?,?,?,?,?)`,
		a.ID, a.AgentID, a.PositionID, a.Status, a.CreatedAt, a.UpdatedAt)
	return err
}

func (s *Store) ListAssignments(companyID string) ([]*model.AgentAssignment, error) {
	return queryAssignments(s.db, `SELECT a.id,a.agent_id,a.position_id,a.status,a.created_at,a.updated_at FROM agent_assignments a JOIN positions p ON a.position_id=p.id JOIN departments d ON p.department_id=d.id WHERE d.company_id=? ORDER BY a.created_at`, companyID)
}

func (s *Store) UpdateAssignment(a *model.AgentAssignment) error {
	_, err := s.db.Exec(`UPDATE agent_assignments SET agent_id=?,position_id=?,status=?,updated_at=? WHERE id=?`,
		a.AgentID, a.PositionID, a.Status, time.Now(), a.ID)
	return err
}

func (s *Store) DeleteAssignment(id string) error {
	_, err := s.db.Exec(`DELETE FROM agent_assignments WHERE id=?`, id)
	return err
}

func scanAssignment(row scannable) (*model.AgentAssignment, error) {
	var a model.AgentAssignment
	if err := row.Scan(&a.ID, &a.AgentID, &a.PositionID, &a.Status, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

func queryAssignments(db *sql.DB, query string, args ...interface{}) ([]*model.AgentAssignment, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.AgentAssignment
	for rows.Next() {
		a, err := scanAssignment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}
