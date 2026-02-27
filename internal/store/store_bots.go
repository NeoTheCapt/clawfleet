package store

import (
	"database/sql"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// Bot CRUD
func (s *Store) CreateBot(b *model.BotConfig) error {
	_, err := s.db.Exec(`INSERT INTO bots (id,company_id,name,bot_id,token,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		b.ID, b.CompanyID, b.Name, b.BotID, b.Token, b.CreatedAt, b.UpdatedAt)
	return err
}

func (s *Store) GetBot(id string) (*model.BotConfig, error) {
	return scanBot(s.db.QueryRow(`SELECT id,company_id,name,bot_id,token,created_at,updated_at FROM bots WHERE id=?`, id))
}

func (s *Store) ListBots(companyID string) ([]*model.BotConfig, error) {
	return queryBots(s.db, `SELECT id,company_id,name,bot_id,token,created_at,updated_at FROM bots WHERE company_id=? ORDER BY created_at`, companyID)
}

func (s *Store) UpdateBot(b *model.BotConfig) error {
	_, err := s.db.Exec(`UPDATE bots SET name=?,bot_id=?,token=?,updated_at=? WHERE id=?`,
		b.Name, b.BotID, b.Token, time.Now(), b.ID)
	return err
}

func (s *Store) DeleteBot(id string) error {
	_, err := s.db.Exec(`DELETE FROM bots WHERE id=?`, id)
	return err
}

func scanBot(row scannable) (*model.BotConfig, error) {
	var b model.BotConfig
	if err := row.Scan(&b.ID, &b.CompanyID, &b.Name, &b.BotID, &b.Token, &b.CreatedAt, &b.UpdatedAt); err != nil {
		return nil, err
	}
	return &b, nil
}

func queryBots(db *sql.DB, query string, args ...interface{}) ([]*model.BotConfig, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.BotConfig, error) { return scanBot(r) })
}
