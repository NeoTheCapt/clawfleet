package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

func (s *Store) CreateWorkItem(w *model.WorkItem) error {
	_, err := s.db.Exec(`INSERT INTO work_items (id,company_id,title,description,status,created_by_type,created_by_id,owner_agent_id,reviewer_agent_id,parent_id,priority,due_at,meta,created_at,updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		w.ID, w.CompanyID, w.Title, w.Description, w.Status, w.CreatedByType, w.CreatedByID, w.OwnerAgentID, w.ReviewerAgentID, w.ParentID, w.Priority, w.DueAt, w.Meta, w.CreatedAt, w.UpdatedAt)
	return err
}

func (s *Store) GetWorkItem(id string) (*model.WorkItem, error) {
	return scanWorkItem(s.db.QueryRow(`SELECT id,company_id,title,description,status,created_by_type,created_by_id,owner_agent_id,reviewer_agent_id,parent_id,priority,due_at,meta,created_at,updated_at FROM work_items WHERE id=?`, id))
}

func (s *Store) ListWorkItems(companyID, status string) ([]*model.WorkItem, error) {
	query := `SELECT id,company_id,title,description,status,created_by_type,created_by_id,owner_agent_id,reviewer_agent_id,parent_id,priority,due_at,meta,created_at,updated_at
		FROM work_items WHERE company_id=?`
	args := []interface{}{companyID}
	if strings.TrimSpace(status) != "" {
		query += ` AND status=?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`
	return queryRows(s.db, query, args, func(r *sql.Rows) (*model.WorkItem, error) { return scanWorkItem(r) })
}

func (s *Store) UpdateWorkItemStatus(id, status string) error {
	_, err := s.db.Exec(`UPDATE work_items SET status=?,updated_at=? WHERE id=?`, status, time.Now(), id)
	return err
}

func (s *Store) AppendWorkItemLog(id string, entry map[string]interface{}) error {
	item, err := s.GetWorkItem(id)
	if err != nil {
		return err
	}
	meta := strings.TrimSpace(item.Meta)
	if meta == "" {
		meta = "{}"
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(meta), &payload); err != nil {
		payload = map[string]interface{}{}
	}
	logEntries := make([]map[string]interface{}, 0)
	if raw, ok := payload["log"]; ok {
		if arr, ok := raw.([]interface{}); ok {
			for _, v := range arr {
				if m, ok := v.(map[string]interface{}); ok {
					logEntries = append(logEntries, m)
				}
			}
		}
	}
	logEntries = append(logEntries, entry)
	payload["log"] = logEntries
	buf, _ := json.Marshal(payload)
	_, err = s.db.Exec(`UPDATE work_items SET meta=?,updated_at=? WHERE id=?`, string(buf), time.Now(), id)
	return err
}

func scanWorkItem(row scannable) (*model.WorkItem, error) {
	var w model.WorkItem
	var dueAt sql.NullTime
	if err := row.Scan(&w.ID, &w.CompanyID, &w.Title, &w.Description, &w.Status, &w.CreatedByType, &w.CreatedByID, &w.OwnerAgentID, &w.ReviewerAgentID, &w.ParentID, &w.Priority, &dueAt, &w.Meta, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return nil, err
	}
	if dueAt.Valid {
		t := dueAt.Time
		w.DueAt = &t
	}
	return &w, nil
}
