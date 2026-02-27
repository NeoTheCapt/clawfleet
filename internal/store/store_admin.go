package store

import (
	"database/sql"
	"fmt"
	"time"
)

type AdminCredentials struct {
	Username  string
	PassHash  []byte
	JWTSecret []byte
	UpdatedAt time.Time
}

func (s *Store) GetAdminCredentials() (*AdminCredentials, error) {
	row := s.db.QueryRow(`SELECT username, pass_hash, jwt_secret, updated_at FROM admin_credentials WHERE id=1`)
	var c AdminCredentials
	if err := row.Scan(&c.Username, &c.PassHash, &c.JWTSecret, &c.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (s *Store) UpsertAdminCredentials(username string, passHash, jwtSecret []byte) error {
	if username == "" {
		return fmt.Errorf("username required")
	}
	if len(passHash) == 0 || len(jwtSecret) == 0 {
		return fmt.Errorf("pass_hash and jwt_secret required")
	}
	_, err := s.db.Exec(`INSERT INTO admin_credentials (id, username, pass_hash, jwt_secret, updated_at)
		VALUES (1, ?, ?, ?, ?) 
		ON CONFLICT(id) DO UPDATE SET username=excluded.username, pass_hash=excluded.pass_hash, jwt_secret=excluded.jwt_secret, updated_at=excluded.updated_at`,
		username, passHash, jwtSecret, time.Now())
	return err
}
