package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
	"github.com/NeoTheCapt/clawfleet/internal/util"
)

const IMKeyEnvVar = "CLAWFLEET_IM_KEY"

type IMKeyRecord struct {
	AgentID   string
	KeyHash   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func hashIMKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

func (s *Store) GetIMKeyByHash(keyHash string) (*IMKeyRecord, error) {
	var rec IMKeyRecord
	err := s.db.QueryRow(`SELECT agent_id,key_hash,created_at,updated_at FROM im_keys WHERE key_hash=?`, keyHash).
		Scan(&rec.AgentID, &rec.KeyHash, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

func (s *Store) HasIMKey(agentID string) (bool, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM im_keys WHERE agent_id=?`, agentID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) UpsertIMKey(agentID, keyHash string) error {
	now := time.Now()
	_, err := s.db.Exec(`INSERT INTO im_keys (agent_id,key_hash,created_at,updated_at) VALUES (?,?,?,?)
		ON CONFLICT(agent_id) DO UPDATE SET key_hash=excluded.key_hash, updated_at=excluded.updated_at`,
		agentID, keyHash, now, now)
	return err
}

func (s *Store) RotateIMKey(agentID string) (string, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var configRaw string
	if err := tx.QueryRow(`SELECT config FROM agent_instances WHERE id=?`, agentID).Scan(&configRaw); err != nil {
		return "", err
	}

	var cfg model.AgentConfig
	if strings.TrimSpace(configRaw) != "" {
		if err := json.Unmarshal([]byte(configRaw), &cfg); err != nil {
			return "", err
		}
	}
	if cfg.EnvVars == nil {
		cfg.EnvVars = map[string]string{}
	}

	plaintext := util.GenerateToken()
	cfg.EnvVars[IMKeyEnvVar] = plaintext
	configJSON, _ := json.Marshal(cfg)

	now := time.Now()
	if _, err := tx.Exec(`UPDATE agent_instances SET config=?,updated_at=? WHERE id=?`, string(configJSON), now, agentID); err != nil {
		return "", err
	}

	keyHash := hashIMKey(plaintext)
	if _, err := tx.Exec(`INSERT INTO im_keys (agent_id,key_hash,created_at,updated_at) VALUES (?,?,?,?)
		ON CONFLICT(agent_id) DO UPDATE SET key_hash=excluded.key_hash, updated_at=excluded.updated_at`,
		agentID, keyHash, now, now); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return plaintext, nil
}

func (s *Store) FindAgentIDByIMKey(plaintext string) (string, error) {
	rec, err := s.GetIMKeyByHash(hashIMKey(plaintext))
	if err != nil {
		return "", err
	}
	return rec.AgentID, nil
}
