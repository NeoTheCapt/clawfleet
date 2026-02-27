package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ═══════════════════════════════════════════════════════════
// Node CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateNode(n *model.Node) error {
	labels, _ := json.Marshal(n.Labels)
	resources, _ := json.Marshal(n.Resources)
	_, err := s.db.Exec(
		`INSERT INTO nodes (id,name,address,status,labels,resources,agent_key,agent_version,agent_commit,agent_build_time,created_at,updated_at,last_seen) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		n.ID, n.Name, n.Address, n.Status, string(labels), string(resources), n.AgentKey,
		n.Resources.AgentVersion, n.Resources.AgentCommit, n.Resources.AgentBuildTime,
		n.CreatedAt, n.UpdatedAt, n.LastSeen)
	return err
}

func (s *Store) GetNode(id string) (*model.Node, error) {
	return scanNode(s.db.QueryRow(`SELECT id,name,address,status,labels,resources,agent_key,agent_version,agent_commit,agent_build_time,created_at,updated_at,last_seen FROM nodes WHERE id=?`, id))
}

func (s *Store) ListNodes() ([]*model.Node, error) {
	return queryNodes(s.db, `SELECT id,name,address,status,labels,resources,agent_key,agent_version,agent_commit,agent_build_time,created_at,updated_at,last_seen FROM nodes ORDER BY created_at DESC`)
}

func (s *Store) UpdateNodeStatus(id string, status model.NodeStatus) error {
	_, err := s.db.Exec(`UPDATE nodes SET status=?,updated_at=?,last_seen=? WHERE id=?`, status, time.Now(), time.Now(), id)
	return err
}

func (s *Store) UpdateNodeHeartbeat(id string, resources model.Resources) error {
	res, _ := json.Marshal(resources)
	_, err := s.db.Exec(
		`UPDATE nodes SET resources=?,agent_version=?,agent_commit=?,agent_build_time=?,last_seen=?,updated_at=? WHERE id=?`,
		string(res), resources.AgentVersion, resources.AgentCommit, resources.AgentBuildTime, time.Now(), time.Now(), id,
	)
	return err
}

func (s *Store) UpdateNode(n *model.Node) error {
	labels, _ := json.Marshal(n.Labels)
	_, err := s.db.Exec(`UPDATE nodes SET name=?,address=?,labels=?,updated_at=? WHERE id=?`, n.Name, n.Address, string(labels), time.Now(), n.ID)
	return err
}

func (s *Store) UpdateNodeFull(n *model.Node) error {
	labels, _ := json.Marshal(n.Labels)
	resources, _ := json.Marshal(n.Resources)
	_, err := s.db.Exec(`UPDATE nodes SET name=?,address=?,status=?,labels=?,resources=?,agent_key=?,updated_at=?,last_seen=? WHERE id=?`,
		n.Name, n.Address, n.Status, string(labels), string(resources), n.AgentKey, n.UpdatedAt, n.LastSeen, n.ID)
	return err
}

func (s *Store) DeleteNode(id string) error {
	if err := s.ExpireTokensByNode(id); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM nodes WHERE id=?`, id)
	return err
}

func (s *Store) FindNodeByAgentKey(agentKey string) (*model.Node, error) {
	return scanNode(s.db.QueryRow(`SELECT id,name,address,status,labels,resources,agent_key,agent_version,agent_commit,agent_build_time,created_at,updated_at,last_seen FROM nodes WHERE agent_key=?`, agentKey))
}

func scanNode(row scannable) (*model.Node, error) {
	var n model.Node
	var labels, resources string
	var agentVersion, agentCommit, agentBuildTime string
	if err := row.Scan(
		&n.ID, &n.Name, &n.Address, &n.Status,
		&labels, &resources, &n.AgentKey,
		&agentVersion, &agentCommit, &agentBuildTime,
		&n.CreatedAt, &n.UpdatedAt, &n.LastSeen,
	); err != nil {
		return nil, err
	}
	if strings.TrimSpace(labels) != "" {
		if err := json.Unmarshal([]byte(labels), &n.Labels); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(resources) != "" {
		if err := json.Unmarshal([]byte(resources), &n.Resources); err != nil {
			return nil, err
		}
	}
	// Copy version fields into Resources for API convenience
	n.Resources.AgentVersion = agentVersion
	n.Resources.AgentCommit = agentCommit
	n.Resources.AgentBuildTime = agentBuildTime
	return &n, nil
}

func queryNodes(db *sql.DB, query string, args ...interface{}) ([]*model.Node, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.Node, error) { return scanNode(r) })
}
