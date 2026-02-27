package store

import (
	"database/sql"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/util"
)

type IMOutboxItem struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	ToAgentID      string    `json:"to_agent_id"`
	Body           string    `json:"body"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (s *Store) ListIMOutbox(agentID, status string, limit int) ([]*IMOutboxItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if status == "" {
		status = "pending"
	}
	rows, err := s.db.Query(`SELECT id,conversation_id,to_agent_id,body,status,created_at,updated_at
        FROM im_outbox
        WHERE to_agent_id=? AND status=?
        ORDER BY created_at ASC
        LIMIT ?`, agentID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]*IMOutboxItem, 0)
	for rows.Next() {
		var it IMOutboxItem
		if err := rows.Scan(&it.ID, &it.ConversationID, &it.ToAgentID, &it.Body, &it.Status, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &it)
	}
	return out, nil
}

func (s *Store) GetIMOutboxItem(id string) (*IMOutboxItem, error) {
	var it IMOutboxItem
	err := s.db.QueryRow(`SELECT id,conversation_id,to_agent_id,body,status,created_at,updated_at FROM im_outbox WHERE id=?`, id).
		Scan(&it.ID, &it.ConversationID, &it.ToAgentID, &it.Body, &it.Status, &it.CreatedAt, &it.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Store) CreateIMOutbox(conversationID, toAgentID, body string) error {
	now := time.Now()
	_, err := s.db.Exec(`INSERT INTO im_outbox (id,conversation_id,to_agent_id,body,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		util.GenerateID("imo"), conversationID, toAgentID, body, "pending", now, now)
	return err
}

func (s *Store) UpdateIMOutboxStatus(id, status string) error {
	_, err := s.db.Exec(`UPDATE im_outbox SET status=?, updated_at=? WHERE id=?`, status, time.Now(), id)
	return err
}

func (s *Store) AckIMOutbox(id string) error {
	return s.UpdateIMOutboxStatus(id, "done")
}

func (s *Store) FailIMOutbox(id string) error {
	return s.UpdateIMOutboxStatus(id, "failed")
}

// ListPendingIMOutbox returns pending outbox items across all agents.
func (s *Store) ListPendingIMOutbox(limit int) ([]*IMOutboxItem, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.db.Query(`SELECT id,conversation_id,to_agent_id,body,status,created_at,updated_at
		FROM im_outbox
		WHERE status='pending'
		ORDER BY created_at ASC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*IMOutboxItem, 0)
	for rows.Next() {
		var it IMOutboxItem
		if err := rows.Scan(&it.ID, &it.ConversationID, &it.ToAgentID, &it.Body, &it.Status, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &it)
	}
	return out, nil
}

// ClaimIMOutbox attempts to move a pending item to processing.
// Returns true if claimed.
func (s *Store) ClaimIMOutbox(id string) (bool, error) {
	res, err := s.db.Exec(`UPDATE im_outbox SET status='processing', updated_at=? WHERE id=? AND status='pending'`, time.Now(), id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *Store) EnsureIMOutboxTable() error {
	// legacy placeholder: table is created in migrations
	return nil
}

var _ = sql.ErrNoRows
