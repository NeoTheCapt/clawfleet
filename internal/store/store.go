package store

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

type scannable interface {
	Scan(dest ...interface{}) error
}

func (s *Store) migrate() error {
	stmts := []string{
		// Core infrastructure
		`CREATE TABLE IF NOT EXISTS nodes (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, address TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'offline', labels TEXT DEFAULT '{}',
			resources TEXT DEFAULT '{}', agent_key TEXT DEFAULT '',
			agent_version TEXT DEFAULT '', agent_commit TEXT DEFAULT '', agent_build_time TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_seen DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE nodes ADD COLUMN agent_version TEXT DEFAULT ''`,
		`ALTER TABLE nodes ADD COLUMN agent_commit TEXT DEFAULT ''`,
		`ALTER TABLE nodes ADD COLUMN agent_build_time TEXT DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS agent_instances (
			id TEXT PRIMARY KEY, name TEXT NOT NULL, company_id TEXT DEFAULT '',
			node_id TEXT NOT NULL, agent_type TEXT NOT NULL, role TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'creating', container_id TEXT DEFAULT '',
			deploy_mode TEXT DEFAULT 'docker', config TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (node_id) REFERENCES nodes(id)
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY, node_id TEXT NOT NULL, action TEXT NOT NULL,
			payload TEXT DEFAULT '{}', status TEXT NOT NULL DEFAULT 'pending',
			result TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (node_id) REFERENCES nodes(id)
		)`,
		`CREATE TABLE IF NOT EXISTS tokens (
			id TEXT PRIMARY KEY, token TEXT UNIQUE NOT NULL, name TEXT DEFAULT '',
			node_id TEXT DEFAULT '', used BOOLEAN DEFAULT FALSE,
			used_by_node TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP, used_at DATETIME
		)`,

		// Admin credentials (persisted; single-row table)
		`CREATE TABLE IF NOT EXISTS admin_credentials (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			username TEXT NOT NULL,
			pass_hash BLOB NOT NULL,
			jwt_secret BLOB NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// Company Management
		`CREATE TABLE IF NOT EXISTS companies (
			id TEXT PRIMARY KEY, name TEXT NOT NULL,
			vision TEXT DEFAULT '', mission TEXT DEFAULT '',
			owner_id TEXT DEFAULT '',
			status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`ALTER TABLE companies ADD COLUMN owner_id TEXT DEFAULT ''`,
		`CREATE TABLE IF NOT EXISTS bots (
			id TEXT PRIMARY KEY, company_id TEXT NOT NULL,
			name TEXT NOT NULL, bot_id TEXT NOT NULL,
			token TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (company_id) REFERENCES companies(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bots_company ON bots(company_id)`,

		`CREATE TABLE IF NOT EXISTS company_goals (
			id TEXT PRIMARY KEY, company_id TEXT NOT NULL,
			title TEXT NOT NULL, description TEXT DEFAULT '',
			priority TEXT DEFAULT 'medium', status TEXT DEFAULT 'planning',
			owner_position_id TEXT DEFAULT '', deadline DATETIME,
			parent_goal_id TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (company_id) REFERENCES companies(id)
		)`,
		`CREATE TABLE IF NOT EXISTS departments (
			id TEXT PRIMARY KEY, company_id TEXT NOT NULL,
			name TEXT NOT NULL, parent_id TEXT DEFAULT '',
			type TEXT DEFAULT '', description TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (company_id) REFERENCES companies(id)
		)`,
		`CREATE TABLE IF NOT EXISTS positions (
			id TEXT PRIMARY KEY, department_id TEXT NOT NULL,
			title TEXT NOT NULL, level TEXT DEFAULT 'staff',
			reports_to TEXT DEFAULT '', responsibilities TEXT DEFAULT '',
			system_prompt TEXT DEFAULT '', background TEXT DEFAULT '',
			persona_template_id TEXT DEFAULT '', workflow_id TEXT DEFAULT '',
			telegram_bot_id TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (department_id) REFERENCES departments(id)
		)`,
		`ALTER TABLE positions ADD COLUMN telegram_bot_id TEXT DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_positions_bot ON positions(telegram_bot_id)`,
		`CREATE TABLE IF NOT EXISTS persona_templates (
			id TEXT PRIMARY KEY, name TEXT NOT NULL,
			description TEXT DEFAULT '', system_prompt TEXT DEFAULT '',
			background TEXT DEFAULT '', tags TEXT DEFAULT '[]',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS persona_sync_history (
			id TEXT PRIMARY KEY, position_id TEXT NOT NULL,
			agent_id TEXT NOT NULL, action TEXT NOT NULL,
			old_system_prompt TEXT DEFAULT '', new_system_prompt TEXT DEFAULT '',
			old_background TEXT DEFAULT '', new_background TEXT DEFAULT '',
			status TEXT NOT NULL, error TEXT DEFAULT '',
			synced_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (position_id) REFERENCES positions(id),
			FOREIGN KEY (agent_id) REFERENCES agent_instances(id)
		)`,
		// Persona Snapshots for versioning and rollback
		`CREATE TABLE IF NOT EXISTS persona_snapshots (
			id TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL,
			position_id TEXT NOT NULL,
			system_prompt TEXT DEFAULT '',
			background TEXT DEFAULT '',
			persona_name TEXT DEFAULT '',
			files_json TEXT DEFAULT '{}',
			version INTEGER DEFAULT 1,
			reason TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (agent_id) REFERENCES agent_instances(id),
			FOREIGN KEY (position_id) REFERENCES positions(id)
		)`,
		// Agent Persona Status for tracking sync state
		`CREATE TABLE IF NOT EXISTS agent_persona_status (
			agent_id TEXT PRIMARY KEY,
			position_id TEXT NOT NULL,
			current_snapshot_id TEXT,
			last_sync_at DATETIME,
			sync_status TEXT DEFAULT 'synced',
			custom_modified BOOLEAN DEFAULT FALSE,
			conflict_resolution TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (agent_id) REFERENCES agent_instances(id),
			FOREIGN KEY (position_id) REFERENCES positions(id)
		)`,
		`CREATE TABLE IF NOT EXISTS agent_assignments (
			id TEXT PRIMARY KEY, agent_id TEXT NOT NULL,
			position_id TEXT NOT NULL, persona TEXT DEFAULT '',
			background TEXT DEFAULT '', im_type TEXT DEFAULT '',
			im_config TEXT DEFAULT '{}', status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS review_cycles (
			id TEXT PRIMARY KEY, company_id TEXT NOT NULL,
			name TEXT NOT NULL, frequency TEXT DEFAULT 'weekly',
			metrics TEXT DEFAULT '[]', report_template TEXT DEFAULT '',
			auto_review INTEGER DEFAULT 0, last_triggered DATETIME,
			status TEXT DEFAULT 'active',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (company_id) REFERENCES companies(id)
		)`,
		// Migration: add columns if missing
		"ALTER TABLE positions ADD COLUMN system_prompt TEXT DEFAULT ''",
		"ALTER TABLE positions ADD COLUMN background TEXT DEFAULT ''",
		"ALTER TABLE im_conversations ADD COLUMN company_id TEXT DEFAULT ''",
		"ALTER TABLE review_cycles ADD COLUMN auto_review INTEGER DEFAULT 0",
		"ALTER TABLE review_cycles ADD COLUMN last_triggered DATETIME",
		"ALTER TABLE review_cycles ADD COLUMN status TEXT DEFAULT 'active'",
		`UPDATE im_conversations
		SET company_id = COALESCE(
			(SELECT d.company_id FROM departments d WHERE d.id=im_conversations.ref_id LIMIT 1),
			''
		)
		WHERE type='department' AND (company_id IS NULL OR company_id='')`,
		`UPDATE im_conversations
		SET company_id = COALESCE(
			(SELECT d.company_id
			 FROM positions p
			 JOIN departments d ON d.id=p.department_id
			 WHERE p.id=(CASE
				WHEN instr(im_conversations.ref_id, ':') > 0 THEN substr(im_conversations.ref_id, instr(im_conversations.ref_id, ':')+1)
				ELSE im_conversations.ref_id
			 END)
			 LIMIT 1),
			(SELECT d.company_id
			 FROM positions p
			 JOIN departments d ON d.id=p.department_id
			 WHERE p.id=(CASE
				WHEN instr(im_conversations.ref_id, ':') > 0 THEN substr(im_conversations.ref_id, 1, instr(im_conversations.ref_id, ':')-1)
				ELSE ''
			 END)
			 LIMIT 1),
			''
		)
		WHERE type='line' AND (company_id IS NULL OR company_id='')`,
		`CREATE TABLE IF NOT EXISTS kpi_reviews (
			id TEXT PRIMARY KEY, agent_id TEXT NOT NULL,
			cycle_id TEXT NOT NULL, period TEXT NOT NULL,
			metrics TEXT DEFAULT '{}', score REAL DEFAULT 0,
			self_report TEXT DEFAULT '', reviewer_id TEXT DEFAULT '',
			feedback TEXT DEFAULT '', status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS im_groups (
			id TEXT PRIMARY KEY, company_id TEXT NOT NULL,
			name TEXT NOT NULL, department_id TEXT DEFAULT '',
			type TEXT DEFAULT 'team', members TEXT DEFAULT '[]',
			external_channel TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (company_id) REFERENCES companies(id)
		)`,
		`CREATE TABLE IF NOT EXISTS im_conversations (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			company_id TEXT DEFAULT '',
			ref_id TEXT DEFAULT '',
			title TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS im_members (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			member_type TEXT NOT NULL,
			member_id TEXT NOT NULL,
			role TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conversation_id) REFERENCES im_conversations(id)
		)`,
		`CREATE TABLE IF NOT EXISTS im_messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			sender_type TEXT NOT NULL,
			sender_id TEXT NOT NULL,
			body TEXT NOT NULL,
			meta TEXT DEFAULT '{}',
			status TEXT DEFAULT 'sent',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conversation_id) REFERENCES im_conversations(id)
		)`,
		`CREATE TABLE IF NOT EXISTS im_outbox (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			to_agent_id TEXT NOT NULL,
			body TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (conversation_id) REFERENCES im_conversations(id)
		)`,
		`CREATE TABLE IF NOT EXISTS im_keys (
			agent_id TEXT PRIMARY KEY,
			key_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (agent_id) REFERENCES agent_instances(id)
		)`,
		`CREATE TABLE IF NOT EXISTS workflows (
			id TEXT PRIMARY KEY, company_id TEXT NOT NULL,
			name TEXT NOT NULL, position_title TEXT DEFAULT '',
			steps TEXT DEFAULT '[]', triggers TEXT DEFAULT '[]',
			output_requirements TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (company_id) REFERENCES companies(id)
		)`,
		`CREATE TABLE IF NOT EXISTS governance_logs (
			id TEXT PRIMARY KEY, company_id TEXT NOT NULL,
			action TEXT NOT NULL, actor_id TEXT DEFAULT '',
			detail TEXT DEFAULT '',
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS work_items (
			id TEXT PRIMARY KEY,
			company_id TEXT NOT NULL,
			title TEXT NOT NULL,
			description TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'draft',
			created_by_type TEXT NOT NULL DEFAULT 'system',
			created_by_id TEXT NOT NULL DEFAULT '',
			owner_agent_id TEXT DEFAULT '',
			reviewer_agent_id TEXT DEFAULT '',
			parent_id TEXT DEFAULT '',
			priority TEXT NOT NULL DEFAULT 'medium',
			due_at DATETIME,
			meta TEXT DEFAULT '{}',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (company_id) REFERENCES companies(id)
		)`,
		// Indexes for common lookups
		`CREATE INDEX IF NOT EXISTS idx_positions_department ON positions(department_id)`,
		`CREATE INDEX IF NOT EXISTS idx_departments_company ON departments(company_id)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_instances_node ON agent_instances(node_id)`,
		`CREATE INDEX IF NOT EXISTS idx_agent_instances_company ON agent_instances(company_id)`,
		`CREATE INDEX IF NOT EXISTS idx_assignments_position ON agent_assignments(position_id)`,
		`CREATE INDEX IF NOT EXISTS idx_assignments_agent ON agent_assignments(agent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_im_conversations_company ON im_conversations(company_id, updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_im_conversations_type_ref ON im_conversations(type, ref_id)`,
		`CREATE INDEX IF NOT EXISTS idx_im_members_conversation ON im_members(conversation_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_im_member_unique ON im_members(conversation_id, member_type, member_id)`,
		`CREATE INDEX IF NOT EXISTS idx_im_messages_conversation ON im_messages(conversation_id, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_im_outbox_status ON im_outbox(status, updated_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_im_keys_hash ON im_keys(key_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_work_items_company_status ON work_items(company_id, status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_work_items_owner ON work_items(owner_agent_id, status, created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_work_items_parent ON work_items(parent_id)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			// Ignore "duplicate column" errors from ALTER TABLE migrations
			if strings.Contains(err.Error(), "duplicate column") {
				continue
			}
			return fmt.Errorf("exec %q: %w", stmt[:min(40, len(stmt))], err)
		}
	}
	return nil
}
