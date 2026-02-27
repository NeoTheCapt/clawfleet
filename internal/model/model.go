package model

import "time"

// ─── Node ───────────────────────────────────────────────────

type NodeStatus string

const (
	NodeStatusPending NodeStatus = "pending"
	NodeStatusOnline  NodeStatus = "online"
	NodeStatusOffline NodeStatus = "offline"
	NodeStatusBusy    NodeStatus = "busy"
)

type Node struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Address   string            `json:"address"`
	Status    NodeStatus        `json:"status"`
	Labels    map[string]string `json:"labels,omitempty"`
	Resources Resources         `json:"resources"`
	AgentKey  string            `json:"agent_key,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	LastSeen  time.Time         `json:"last_seen"`
}

type Resources struct {
	CPUCores    int     `json:"cpu_cores"`
	MemoryMB    int     `json:"memory_mb"`
	DiskMB      int     `json:"disk_mb"`
	GPUs        int     `json:"gpus"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`

	// Node agent build info (for version tracking)
	AgentVersion   string `json:"agent_version,omitempty"`
	AgentCommit    string `json:"agent_commit,omitempty"`
	AgentBuildTime string `json:"agent_build_time,omitempty"`
}

// ─── Agent ──────────────────────────────────────────────────

type AgentType string

const (
	AgentTypeOpenClaw AgentType = "openclaw"
	AgentTypeZeroClaw AgentType = "zeroclaw"
	AgentTypeNanobot  AgentType = "nanobot"
)

type AgentStatus string

const (
	AgentStatusCreating AgentStatus = "creating"
	AgentStatusRunning  AgentStatus = "running"
	AgentStatusStopped  AgentStatus = "stopped"
	AgentStatusDeleting AgentStatus = "deleting"
	AgentStatusError    AgentStatus = "error"
)

type AgentInstance struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	CompanyID   string      `json:"company_id,omitempty"`
	NodeID      string      `json:"node_id"`
	AgentType   AgentType   `json:"agent_type"`
	Role        string      `json:"role"`
	Status      AgentStatus `json:"status"`
	ContainerID string      `json:"container_id,omitempty"`
	DeployMode  string      `json:"deploy_mode"`
	Config      AgentConfig `json:"config"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type AgentConfig struct {
	// Model & Provider
	Provider       string   `json:"provider,omitempty"`
	Model          string   `json:"model,omitempty"`
	FallbackModels []string `json:"fallback_models,omitempty"`
	CustomAPIBase  string   `json:"custom_api_base,omitempty"`

	// Persona
	SystemPrompt string `json:"system_prompt,omitempty"`
	PersonaName  string `json:"persona_name,omitempty"`
	Language     string `json:"language,omitempty"`

	// Channel/Messaging
	Channel *ChannelConfig `json:"channel,omitempty"`

	// Environment & Resources
	EnvVars   map[string]string `json:"env_vars,omitempty"`
	Resources ResourceLimit     `json:"resources"`

	// Additional metadata
	Image       string `json:"image,omitempty"`
	Description string `json:"description,omitempty"`
}

type ResourceLimit struct {
	CPULimit    string `json:"cpu_limit,omitempty"`
	MemoryLimit string `json:"memory_limit,omitempty"`
}

// ─── Task ───────────────────────────────────────────────────

type TaskStatus string

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusDone    TaskStatus = "done"
	TaskStatusFailed  TaskStatus = "failed"
)

type Task struct {
	ID        string                 `json:"id"`
	NodeID    string                 `json:"node_id"`
	Action    string                 `json:"action"`
	Payload   map[string]interface{} `json:"payload"`
	Status    TaskStatus             `json:"status"`
	Result    map[string]interface{} `json:"result,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// ─── Token ──────────────────────────────────────────────────

type Token struct {
	ID         string     `json:"id"`
	Token      string     `json:"token"`
	Name       string     `json:"name"`
	NodeID     string     `json:"node_id"`
	Used       bool       `json:"used"`
	UsedByNode string     `json:"used_by_node,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UsedAt     *time.Time `json:"used_at,omitempty"`
}

// ═══════════════════════════════════════════════════════════
// Company Management
// ═══════════════════════════════════════════════════════════

// ─── Company ────────────────────────────────────────────────

type Company struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Vision    string    `json:"vision,omitempty"`
	Mission   string    `json:"mission,omitempty"`
	OwnerID   string    `json:"owner_id,omitempty"`
	Status    string    `json:"status"` // active, archived
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ─── Goal ───────────────────────────────────────────────────

type GoalStatus string

const (
	GoalStatusPlanning  GoalStatus = "planning"
	GoalStatusActive    GoalStatus = "active"
	GoalStatusCompleted GoalStatus = "completed"
	GoalStatusCancelled GoalStatus = "cancelled"
)

type CompanyGoal struct {
	ID              string     `json:"id"`
	CompanyID       string     `json:"company_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description,omitempty"`
	Priority        string     `json:"priority"` // high, medium, low
	Status          GoalStatus `json:"status"`
	OwnerPositionID string     `json:"owner_position_id,omitempty"`
	Deadline        *time.Time `json:"deadline,omitempty"`
	ParentGoalID    string     `json:"parent_goal_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ─── Department ─────────────────────────────────────────────

type Department struct {
	ID          string    `json:"id"`
	CompanyID   string    `json:"company_id"`
	Name        string    `json:"name"`
	ParentID    string    `json:"parent_id,omitempty"` // tree structure
	Type        string    `json:"type"`                // management, engineering, operations, sales, support
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ─── Position ───────────────────────────────────────────────

type PositionLevel string

const (
	PositionLevelCSuite   PositionLevel = "c-suite"
	PositionLevelDirector PositionLevel = "director"
	PositionLevelManager  PositionLevel = "manager"
	PositionLevelStaff    PositionLevel = "staff"
)

// PersonaTemplate 人设模板
type PersonaTemplate struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
	Background   string    `json:"background,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PersonaSyncRecord 人设同步记录
type PersonaSyncRecord struct {
	ID              string    `json:"id"`
	PositionID      string    `json:"position_id"`
	AgentID         string    `json:"agent_id"`
	Action          string    `json:"action"` // "create", "update", "sync", "error"
	OldSystemPrompt string    `json:"old_system_prompt,omitempty"`
	NewSystemPrompt string    `json:"new_system_prompt,omitempty"`
	OldBackground   string    `json:"old_background,omitempty"`
	NewBackground   string    `json:"new_background,omitempty"`
	Status          string    `json:"status"` // "pending", "success", "failed"
	Error           string    `json:"error,omitempty"`
	SyncedAt        time.Time `json:"synced_at"`
}

// PersonaSnapshot 人设快照 - 用于版本管理和回滚
type PersonaSnapshot struct {
	ID           string    `json:"id"`
	AgentID      string    `json:"agent_id"`
	PositionID   string    `json:"position_id"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
	Background   string    `json:"background,omitempty"`
	PersonaName  string    `json:"persona_name,omitempty"`
	FilesJSON    string    `json:"files_json,omitempty"`
	Version      int       `json:"version"`
	Reason       string    `json:"reason,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// AgentPersonaStatus Agent 人设状态
type AgentPersonaStatus struct {
	AgentID            string    `json:"agent_id"`
	PositionID         string    `json:"position_id"`
	CurrentSnapshotID  string    `json:"current_snapshot_id,omitempty"`
	LastSyncAt         time.Time `json:"last_sync_at,omitempty"`
	SyncStatus         string    `json:"sync_status"` // "synced", "pending", "conflict", "error"
	CustomModified     bool      `json:"custom_modified"`
	ConflictResolution string    `json:"conflict_resolution,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type Position struct {
	ID               string        `json:"id"`
	DepartmentID     string        `json:"department_id"`
	Title            string        `json:"title"`
	Level            PositionLevel `json:"level"`
	ReportsTo        string        `json:"reports_to,omitempty"` // position ID
	Responsibilities string        `json:"responsibilities,omitempty"`
	// 人设配置：可直接设置，或引用模板
	SystemPrompt      string    `json:"system_prompt,omitempty"`       // SOUL / persona prompt
	Background        string    `json:"background,omitempty"`          // role background context
	PersonaTemplateID string    `json:"persona_template_id,omitempty"` // 引用的人设模板ID
	WorkflowID        string    `json:"workflow_id,omitempty"`
	TelegramBotID     string    `json:"telegram_bot_id,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ─── Agent Assignment ───────────────────────────────────────

type AgentAssignment struct {
	ID         string    `json:"id"`
	AgentID    string    `json:"agent_id"`
	PositionID string    `json:"position_id"`
	Status     string    `json:"status"` // active, inactive
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ─── Work Item ─────────────────────────────────────────────

type WorkItemStatus string

const (
	WorkItemStatusDraft      WorkItemStatus = "draft"
	WorkItemStatusProposed   WorkItemStatus = "proposed"
	WorkItemStatusApproved   WorkItemStatus = "approved"
	WorkItemStatusInProgress WorkItemStatus = "in_progress"
	WorkItemStatusDone       WorkItemStatus = "done"
	WorkItemStatusCancelled  WorkItemStatus = "cancelled"
)

type WorkItem struct {
	ID              string     `json:"id"`
	CompanyID       string     `json:"company_id"`
	Title           string     `json:"title"`
	Description     string     `json:"description,omitempty"`
	Status          string     `json:"status"`
	CreatedByType   string     `json:"created_by_type"` // user, agent, system
	CreatedByID     string     `json:"created_by_id"`
	OwnerAgentID    string     `json:"owner_agent_id,omitempty"`
	ReviewerAgentID string     `json:"reviewer_agent_id,omitempty"`
	ParentID        string     `json:"parent_id,omitempty"`
	Priority        string     `json:"priority"` // low, medium, high
	DueAt           *time.Time `json:"due_at,omitempty"`
	Meta            string     `json:"meta,omitempty"` // json
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ─── KPI / Review ───────────────────────────────────────────

type ReviewCycle struct {
	ID             string    `json:"id"`
	CompanyID      string    `json:"company_id"`
	Name           string    `json:"name"`
	Frequency      string    `json:"frequency"` // weekly, biweekly, monthly, quarterly
	Metrics        []string  `json:"metrics"`   // metric names
	ReportTemplate string    `json:"report_template,omitempty"`
	AutoReview     bool      `json:"auto_review"`
	LastTriggered  time.Time `json:"last_triggered,omitempty"`
	Status         string    `json:"status,omitempty"` // active, completed, pending
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type KPIReview struct {
	ID         string             `json:"id"`
	AgentID    string             `json:"agent_id"`
	CycleID    string             `json:"cycle_id"`
	Period     string             `json:"period"` // 2026-W07, 2026-02
	Metrics    map[string]float64 `json:"metrics,omitempty"`
	Score      float64            `json:"score"`
	SelfReport string             `json:"self_report,omitempty"`
	ReviewerID string             `json:"reviewer_id,omitempty"` // manager agent_id
	Feedback   string             `json:"feedback,omitempty"`
	Status     string             `json:"status"` // pending, submitted, reviewed
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// ─── Web IM ────────────────────────────────────────────────

const (
	IMChairmanUserID = "chairman"
	IMCEOAgentID     = "agent_ceo"
)

type IMMemberType string

const (
	IMMemberTypeUser  IMMemberType = "user"
	IMMemberTypeAgent IMMemberType = "agent"
)

func IsValidIMMemberType(v string) bool {
	return v == string(IMMemberTypeUser) || v == string(IMMemberTypeAgent)
}

type IMConversation struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"` // department, line, custom
	CompanyID     string     `json:"company_id,omitempty"`
	RefID         string     `json:"ref_id,omitempty"`
	Title         string     `json:"title"`
	MemberCount   int        `json:"member_count,omitempty"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type IMMember struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	MemberType     string    `json:"member_type"` // user, agent
	MemberID       string    `json:"member_id"`
	Role           string    `json:"role,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type IMMessage struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	SenderType     string    `json:"sender_type"` // user, agent
	SenderID       string    `json:"sender_id"`
	Body           string    `json:"body"`
	Meta           string    `json:"meta,omitempty"`
	Status         string    `json:"status,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type IMOutbox struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	ToAgentID      string    `json:"to_agent_id"`
	Body           string    `json:"body"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ─── IM Group ───────────────────────────────────────────────

type IMGroup struct {
	ID              string            `json:"id"`
	CompanyID       string            `json:"company_id"`
	Name            string            `json:"name"`
	DepartmentID    string            `json:"department_id,omitempty"`
	Type            string            `json:"type"`    // all-hands, department, team, 1v1
	Members         []string          `json:"members"` // agent IDs
	ExternalChannel map[string]string `json:"external_channel,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// ─── Workflow ───────────────────────────────────────────────

type WorkflowStep struct {
	Order       int    `json:"order"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Action      string `json:"action,omitempty"`  // what to do
	Output      string `json:"output,omitempty"`  // expected output
	Timeout     string `json:"timeout,omitempty"` // e.g. "1h"
}

type Workflow struct {
	ID                 string         `json:"id"`
	CompanyID          string         `json:"company_id"`
	Name               string         `json:"name"`
	PositionTitle      string         `json:"position_title,omitempty"` // applicable role
	Steps              []WorkflowStep `json:"steps"`
	Triggers           []string       `json:"triggers,omitempty"`
	OutputRequirements string         `json:"output_requirements,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

// ─── Governance Log ─────────────────────────────────────────

type GovernanceLog struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Action    string    `json:"action"` // org_change, policy_update, escalation, announcement
	ActorID   string    `json:"actor_id"`
	Detail    string    `json:"detail"`
	Timestamp time.Time `json:"timestamp"`
}

// BotConfig represents a managed telegram bot
type BotConfig struct {
	ID        string    `json:"id"`
	CompanyID string    `json:"company_id"`
	Name      string    `json:"name"`
	BotID     string    `json:"bot_id"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ─── Channel Configuration ──────────────────────────────────

type ChannelConfig struct {
	Type          string   `json:"type"` // telegram, discord, slack, whatsapp, webhook
	Token         string   `json:"token,omitempty"`
	BotToken      string   `json:"bot_token,omitempty"`
	AppToken      string   `json:"app_token,omitempty"`
	URL           string   `json:"url,omitempty"`
	Secret        string   `json:"secret,omitempty"`
	GuildID       string   `json:"guild_id,omitempty"`
	AllowFrom     []string `json:"allow_from,omitempty"`
	DMPolicy      string   `json:"dm_policy,omitempty"`
	WebhookSecret string   `json:"webhook_secret,omitempty"`
}
