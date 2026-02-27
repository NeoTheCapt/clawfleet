package sync

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
	"github.com/NeoTheCapt/clawfleet/internal/store"
)

// PersonaSyncEngine 人设同步引擎
type PersonaSyncEngine struct {
	store *store.Store
}

// SyncOptions 同步选项
type SyncOptions struct {
	Force        bool   // 强制同步
	Trigger      string // 触发原因: assignment, position_updated, manual
	SkipConflict bool   // 跳过冲突检测
}

// NewPersonaSyncEngine 创建新的同步引擎
func NewPersonaSyncEngine(s *store.Store) *PersonaSyncEngine {
	return &PersonaSyncEngine{store: s}
}

// SyncPositionPersonaToAllAgents 同步职位人设到所有分配的 agents
func (pse *PersonaSyncEngine) SyncPositionPersonaToAllAgents(positionID string, options *SyncOptions) error {
	if options == nil {
		options = &SyncOptions{Trigger: "manual"}
	}

	// 获取该职位的所有分配
	assignments, err := pse.store.ListAssignmentsByPosition(positionID)
	if err != nil {
		return fmt.Errorf("failed to get assignments: %w", err)
	}

	if len(assignments) == 0 {
		log.Printf("[sync] no agents assigned to position %s", positionID)
		return nil
	}

	log.Printf("[sync] syncing position %s to %d agents (trigger: %s)",
		positionID, len(assignments), options.Trigger)

	successCount := 0
	failureCount := 0

	for _, assignment := range assignments {
		err := pse.SyncToAgent(assignment.AgentID, assignment.PositionID, options.Trigger)
		if err != nil {
			log.Printf("[sync] failed to sync agent %s: %v", assignment.AgentID, err)
			failureCount++
		} else {
			successCount++
		}
	}

	log.Printf("[sync] position %s sync completed: %d success, %d failure",
		positionID, successCount, failureCount)

	if failureCount > 0 {
		return fmt.Errorf("%d agents failed to sync", failureCount)
	}

	return nil
}

// SyncToAgent 同步人设到单个 agent
func (pse *PersonaSyncEngine) SyncToAgent(agentID, positionID, trigger string) error {
	// 获取 agent
	agent, err := pse.store.GetAgentInstance(agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// 获取职位人设
	systemPrompt, background, err := pse.store.GetPersonaForPosition(positionID)
	if err != nil {
		return fmt.Errorf("failed to get persona: %w", err)
	}

	// Sync Telegram bot token from position.telegram_bot_id → agent.Config.Channel
	// (Phase A: Telegram-only; multi-platform bot types can be introduced later.)
	pos, err := pse.store.GetPosition(positionID)
	if err == nil {
		botID := strings.TrimSpace(pos.TelegramBotID)
		if botID != "" {
			if bot, berr := pse.store.GetBot(botID); berr == nil {
				if agent.Config.Channel == nil {
					agent.Config.Channel = &model.ChannelConfig{}
				}
				agent.Config.Channel.Type = "telegram"
				// Keep dm policy / allowFrom if already set elsewhere; only ensure token is present.
				agent.Config.Channel.Token = bot.Token
			} else {
				log.Printf("[sync] warning: telegram bot not found for position %s: bot_id=%s err=%v", positionID, botID, berr)
			}
		}
	}

	// 记录旧配置
	oldSystemPrompt := agent.Config.SystemPrompt
	oldBackground := agent.Config.Description

	// 更新配置
	if systemPrompt != "" {
		agent.Config.SystemPrompt = systemPrompt
	}
	if background != "" {
		agent.Config.Description = background
	}

	// 设置角色名称（如果未设置）
	if agent.Config.PersonaName == "" {
		position, err := pse.store.GetPosition(positionID)
		if err == nil && position.Title != "" {
			agent.Config.PersonaName = position.Title
		}
		if agent.Config.PersonaName == "" {
			agent.Config.PersonaName = agent.Name
		}
	}

	agent.UpdatedAt = time.Now()

	// 更新 agent
	if err := pse.store.UpdateAgentInstance(agent); err != nil {
		// 记录失败
		pse.recordSyncHistory(positionID, agentID, trigger,
			oldSystemPrompt, systemPrompt, oldBackground, background, "failed", err.Error())
		return fmt.Errorf("failed to update agent: %w", err)
	}

	// 记录成功
	pse.recordSyncHistory(positionID, agentID, trigger,
		oldSystemPrompt, systemPrompt, oldBackground, background, "success", "")

	// 创建快照保存历史
	_, err = pse.store.CreatePersonaSnapshotForAgent(agent, positionID, trigger)
	if err != nil {
		log.Printf("[sync] warning: failed to create snapshot for agent %s: %v", agent.ID, err)
	}

	// 更新同步状态
	if err := pse.store.UpdateAgentPersonaSyncStatus(agentID, "synced"); err != nil {
		log.Printf("[sync] warning: failed to update sync status for agent %s: %v", agentID, err)
	}

	log.Printf("[sync] persona synced to agent %s: system_prompt updated=%v, background updated=%v",
		agent.ID, oldSystemPrompt != systemPrompt, oldBackground != background)

	// 如果 agent 正在运行且有 Telegram bot 配置，需要重新部署容器以注入环境变量
	hasTelegramBot := pos != nil && strings.TrimSpace(pos.TelegramBotID) != ""
	if agent.Status == model.AgentStatusRunning && hasTelegramBot {
		err := pse.triggerAgentRedeploy(agent)
		if err != nil {
			log.Printf("[sync] warning: failed to trigger redeploy for agent %s: %v", agent.ID, err)
		}
	} else if agent.Status == model.AgentStatusRunning {
		// 没有 bot 配置时只重载 persona 文件
		err := pse.triggerAgentReload(agent, systemPrompt, background)
		if err != nil {
			log.Printf("[sync] warning: failed to trigger reload for agent %s: %v", agent.ID, err)
		}
	}

	return nil
}

// recordSyncHistory 记录同步历史
func (pse *PersonaSyncEngine) recordSyncHistory(positionID, agentID, action,
	oldSystemPrompt, newSystemPrompt, oldBackground, newBackground, status, errorMsg string) {

	record := &model.PersonaSyncRecord{
		ID:              fmt.Sprintf("psync_%d", time.Now().UnixNano()),
		PositionID:      positionID,
		AgentID:         agentID,
		Action:          action,
		OldSystemPrompt: oldSystemPrompt,
		NewSystemPrompt: newSystemPrompt,
		OldBackground:   oldBackground,
		NewBackground:   newBackground,
		Status:          status,
		Error:           errorMsg,
		SyncedAt:        time.Now(),
	}

	if err := pse.store.CreatePersonaSyncRecord(record); err != nil {
		log.Printf("[sync] failed to record sync history: %v", err)
	}
}

// triggerAgentReload 触发 Agent 重新加载人设配置
func (pse *PersonaSyncEngine) triggerAgentReload(agent *model.AgentInstance, systemPrompt, background string) error {
	// 通过节点 agent 的任务系统触发配置重载
	// 这里创建一个任务让节点 agent 执行人设重载

	reloadTask := &model.Task{
		ID:     fmt.Sprintf("task_reload_%s_%d", agent.ID, time.Now().UnixNano()),
		NodeID: agent.NodeID,
		Action: "reload_persona",
		Payload: map[string]interface{}{
			"agent_id":      agent.ID,
			"agent_name":    agent.Name,
			"system_prompt": systemPrompt,
			"background":    background,
			"type":          "persona_reload",
		},
		Status:    model.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := pse.store.CreateTask(reloadTask); err != nil {
		return fmt.Errorf("failed to create reload task: %w", err)
	}

	log.Printf("[sync] created reload task %s for agent %s on node %s",
		reloadTask.ID, agent.ID, agent.NodeID)

	return nil
}

// GetAgentPersonaStatus 获取 agent 的人设状态
func (pse *PersonaSyncEngine) GetAgentPersonaStatus(agentID string) (map[string]interface{}, error) {
	agent, err := pse.store.GetAgentInstance(agentID)
	if err != nil {
		return nil, err
	}

	// 获取分配信息
	assignments, _ := pse.store.ListAssignmentsByAgent(agentID)
	var positionID string
	if len(assignments) > 0 {
		positionID = assignments[0].PositionID
	}

	result := map[string]interface{}{
		"agent_id":      agentID,
		"agent_name":    agent.Name,
		"position_id":   positionID,
		"system_prompt": agent.Config.SystemPrompt,
		"background":    agent.Config.Description,
		"persona_name":  agent.Config.PersonaName,
		"last_updated":  agent.UpdatedAt,
	}

	// 如果有职位，获取职位人设对比
	if positionID != "" {
		position, err := pse.store.GetPosition(positionID)
		if err == nil {
			result["position_title"] = position.Title
			result["position_system_prompt"] = position.SystemPrompt
			result["position_background"] = position.Background

			// 检查是否同步
			inSync := agent.Config.SystemPrompt == position.SystemPrompt &&
				agent.Config.Description == position.Background
			result["in_sync"] = inSync
		}
	}

	return result, nil
}

// triggerAgentRedeploy triggers stop + deploy tasks for an agent.
// This is required when env-based config changes (e.g. Telegram bot token) so the container is recreated.
func (pse *PersonaSyncEngine) triggerAgentRedeploy(agent *model.AgentInstance) error {
	// Mark creating (UI/state)
	agent.Status = model.AgentStatusCreating
	agent.UpdatedAt = time.Now()
	_ = pse.store.UpdateAgentInstance(agent)

	// Stop old container if known
	if agent.ContainerID != "" {
		stopTask := &model.Task{
			ID:     fmt.Sprintf("task_stop_%s_%d", agent.ID, time.Now().UnixNano()),
			NodeID: agent.NodeID,
			Action: "stop_agent",
			Payload: map[string]interface{}{
				"agent_id":     agent.ID,
				"agent_name":   agent.Name,
				"container_id": agent.ContainerID,
				"type":         "redeploy",
			},
			Status:    model.TaskStatusPending,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := pse.store.CreateTask(stopTask); err != nil {
			return fmt.Errorf("failed to create stop task: %w", err)
		}
		log.Printf("[sync] created stop task %s for agent %s on node %s", stopTask.ID, agent.ID, agent.NodeID)
	}

	// Deploy with fresh config (includes updated channel token)
	deployTask := &model.Task{
		ID:     fmt.Sprintf("task_deploy_%s_%d", agent.ID, time.Now().UnixNano()),
		NodeID: agent.NodeID,
		Action: "deploy_agent",
		Payload: map[string]interface{}{
			"agent_id":    agent.ID,
			"agent_name":  agent.Name,
			"agent_type":  string(agent.AgentType),
			"deploy_mode": agent.DeployMode,
			"config":      agent.Config,
			"type":        "redeploy",
		},
		Status:    model.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := pse.store.CreateTask(deployTask); err != nil {
		return fmt.Errorf("failed to create deploy task: %w", err)
	}
	log.Printf("[sync] created deploy task %s for agent %s on node %s (env config updated)", deployTask.ID, agent.ID, agent.NodeID)
	return nil
}
