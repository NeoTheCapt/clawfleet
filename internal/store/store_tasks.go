package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ═══════════════════════════════════════════════════════════
// Task CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateTask(t *model.Task) error {
	payload, _ := json.Marshal(t.Payload)
	result, _ := json.Marshal(t.Result)
	_, err := s.db.Exec(`INSERT INTO tasks (id,node_id,action,payload,status,result,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)`,
		t.ID, t.NodeID, t.Action, string(payload), t.Status, string(result), t.CreatedAt, t.UpdatedAt)
	return err
}

func (s *Store) GetTask(id string) (*model.Task, error) {
	return scanTask(s.db.QueryRow(`SELECT id,node_id,action,payload,status,result,created_at,updated_at FROM tasks WHERE id=?`, id))
}

func (s *Store) ListPendingTasks(nodeID string) ([]*model.Task, error) {
	return queryTasks(s.db, `SELECT id,node_id,action,payload,status,result,created_at,updated_at FROM tasks WHERE node_id=? AND status='pending' ORDER BY created_at ASC`, nodeID)
}

func (s *Store) UpdateTaskStatus(id string, status model.TaskStatus, result map[string]interface{}) error {
	res, _ := json.Marshal(result)
	_, err := s.db.Exec(`UPDATE tasks SET status=?,result=?,updated_at=? WHERE id=?`, status, string(res), time.Now(), id)
	return err
}

func scanTask(row scannable) (*model.Task, error) {
	var t model.Task
	var payload, result string
	if err := row.Scan(&t.ID, &t.NodeID, &t.Action, &payload, &t.Status, &result, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload) != "" {
		if err := json.Unmarshal([]byte(payload), &t.Payload); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(result) != "" {
		if err := json.Unmarshal([]byte(result), &t.Result); err != nil {
			return nil, err
		}
	}
	return &t, nil
}

func queryTasks(db *sql.DB, query string, args ...interface{}) ([]*model.Task, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.Task, error) { return scanTask(r) })
}
