package store

import (
	"database/sql"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ═══════════════════════════════════════════════════════════
// Token CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateToken(t *model.Token) error {
	_, err := s.db.Exec(`INSERT INTO tokens (id,token,name,node_id,used,used_by_node,created_at,used_at) VALUES (?,?,?,?,?,?,?,?)`,
		t.ID, t.Token, t.Name, t.NodeID, t.Used, t.UsedByNode, t.CreatedAt, t.UsedAt)
	return err
}

func (s *Store) GetToken(token string) (*model.Token, error) {
	return scanToken(s.db.QueryRow(`SELECT id,token,name,node_id,used,used_by_node,created_at,used_at FROM tokens WHERE token=?`, token))
}

func (s *Store) GetTokenByID(id string) (*model.Token, error) {
	return scanToken(s.db.QueryRow(`SELECT id,token,name,node_id,used,used_by_node,created_at,used_at FROM tokens WHERE id=?`, id))
}

func (s *Store) ListTokens() ([]*model.Token, error) {
	return queryRows(s.db,
		`SELECT id,token,name,node_id,used,used_by_node,created_at,used_at FROM tokens ORDER BY created_at DESC`,
		nil,
		func(r *sql.Rows) (*model.Token, error) { return scanToken(r) },
	)
}

func (s *Store) MarkTokenUsed(token string, nodeID string) error {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE tokens SET used=TRUE,used_by_node=?,used_at=? WHERE token=?`, nodeID, now, token)
	return err
}

func (s *Store) ExpireTokensByNode(nodeID string) error {
	now := time.Now()
	_, err := s.db.Exec(`UPDATE tokens SET used=TRUE,used_at=? WHERE node_id=? AND used=FALSE`, now, nodeID)
	return err
}

func (s *Store) DeleteToken(id string) error {
	_, err := s.db.Exec(`DELETE FROM tokens WHERE id=?`, id)
	return err
}

func scanToken(row scannable) (*model.Token, error) {
	var t model.Token
	var usedAt sql.NullTime
	if err := row.Scan(&t.ID, &t.Token, &t.Name, &t.NodeID, &t.Used, &t.UsedByNode, &t.CreatedAt, &usedAt); err != nil {
		return nil, err
	}
	if usedAt.Valid {
		t.UsedAt = &usedAt.Time
	}
	return &t, nil
}
