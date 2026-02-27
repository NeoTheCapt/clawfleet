package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/adapter"
	adapterNanobot "github.com/NeoTheCapt/clawfleet/internal/adapter/nanobot"
	adapterOpenclaw "github.com/NeoTheCapt/clawfleet/internal/adapter/openclaw"
	adapterZeroclaw "github.com/NeoTheCapt/clawfleet/internal/adapter/zeroclaw"
	"github.com/NeoTheCapt/clawfleet/internal/agent/container"
	"github.com/NeoTheCapt/clawfleet/internal/agent/direct"
	"github.com/NeoTheCapt/clawfleet/internal/model"
)

const (
	debugCheckTimeout   = 30 * time.Second
	maxCheckOutputBytes = 12 * 1024
	maxTaskOutputBytes  = 48 * 1024
)

var (
	ansiPattern           = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	jsonSecretPattern     = regexp.MustCompile(`(?i)"(token|api[_-]?key|im[_-]?key|secret|password)"\s*:\s*"([^"]*)"`)
	inlineSecretPattern   = regexp.MustCompile(`(?i)\b(token|api[_-]?key|im[_-]?key|secret|password)\b(\s*[:=]\s*)(\S+)`)
	telegramBotURLPattern = regexp.MustCompile(`https?://api\.telegram\.org/bot[^/\s]+`)
	debugAllowedChecks    = map[string]struct{}{
		"openclaw_status_deep":  {},
		"openclaw_plugins_list": {},
		"openclaw_config":       {},
		"openclaw_logs_tail":    {},
		"telegram_getme":        {},
		"env_keys":              {},
	}
)

// TaskExecutor executes tasks received from the control plane.
type TaskExecutor struct {
	nodeID    string
	container container.ContainerManagerInterface
	installer *direct.Installer
	adapters  *adapter.Registry
}

// NewTaskExecutor creates a new task executor.
func NewTaskExecutor(nodeID string, cm container.ContainerManagerInterface, agentImage string) *TaskExecutor {
	reg := adapter.NewRegistry()
	// Each adapter uses its own default image; agentImage only overrides if non-empty and non-default
	reg.Register(model.AgentTypeOpenClaw, adapterOpenclaw.New(""))
	reg.Register(model.AgentTypeZeroClaw, adapterZeroclaw.New(""))
	reg.Register(model.AgentTypeNanobot, adapterNanobot.New(""))

	return &TaskExecutor{
		nodeID:    nodeID,
		container: cm,
		installer: direct.NewInstaller(),
		adapters:  reg,
	}
}

// --- Task Execution Methods ---

func (e *TaskExecutor) ExecuteDeploy(payload map[string]interface{}) (map[string]interface{}, string) {
	var inst model.AgentInstance

	// Support both payload formats:
	// 1. {"instance": {...}} (full AgentInstance)
	// 2. {"agent_id": "...", "agent_name": "...", "config": {...}} (flat format from CP)
	if instanceData, ok := payload["instance"]; ok {
		data, err := json.Marshal(instanceData)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("marshal error: %v", err)}, "failed"
		}
		if err := json.Unmarshal(data, &inst); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("unmarshal error: %v", err)}, "failed"
		}
	} else {
		// Flat format from CP deployToNode
		inst.ID, _ = payload["agent_id"].(string)
		inst.Name, _ = payload["agent_name"].(string)
		if at, ok := payload["agent_type"].(string); ok {
			inst.AgentType = model.AgentType(at)
		}
		inst.DeployMode, _ = payload["deploy_mode"].(string)
		if cfgData, ok := payload["config"]; ok {
			data, err := json.Marshal(cfgData)
			if err != nil {
				return map[string]interface{}{"error": fmt.Sprintf("marshal config error: %v", err)}, "failed"
			}
			if err := json.Unmarshal(data, &inst.Config); err != nil {
				return map[string]interface{}{"error": fmt.Sprintf("unmarshal config error: %v", err)}, "failed"
			}
		}
	}

	a, ok := e.adapters.Get(inst.AgentType)
	if !ok {
		return map[string]interface{}{"error": fmt.Sprintf("unsupported agent type: %s", inst.AgentType)}, "failed"
	}

	// Default to docker mode if not specified
	deployMode := inst.DeployMode
	if deployMode == "" {
		deployMode = "docker"
	}

	var containerID string
	var deployErr error

	switch deployMode {
	case "docker":
		// Docker deployment
		cfg, err := a.ContainerConfig(&inst)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("container config error: %v", err)}, "failed"
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		containerID, deployErr = e.container.Create(ctx, inst.Name, cfg)
		if deployErr != nil {
			return map[string]interface{}{"error": fmt.Sprintf("docker deploy error: %v", deployErr)}, "failed"
		}

		log.Printf("[task-executor] deployed %s (%s) via Docker -> container %s", inst.Name, inst.AgentType, containerID[:12])

	case "direct":
		// Direct installation
		cfg, err := a.InstallConfig(&inst)
		if err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("install config error: %v", err)}, "failed"
		}

		containerID, deployErr = e.installer.Install(cfg)
		if deployErr != nil {
			return map[string]interface{}{"error": fmt.Sprintf("direct install error: %v", deployErr)}, "failed"
		}

		log.Printf("[task-executor] deployed %s (%s) via direct install -> service %s", inst.Name, inst.AgentType, containerID)

	default:
		return map[string]interface{}{"error": fmt.Sprintf("unsupported deploy mode: %s", deployMode)}, "failed"
	}

	return map[string]interface{}{
		"agent_id":     inst.ID,
		"container_id": containerID,
		"status":       "running",
	}, "done"
}

func (e *TaskExecutor) ExecuteRestart(payload map[string]interface{}) (map[string]interface{}, string) {
	containerID, ok := payload["container_id"].(string)
	if !ok || containerID == "" {
		return map[string]interface{}{"error": "missing or invalid container_id in payload"}, "failed"
	}

	if err := restartRuntime(context.Background(), e.container, e.installer, containerID); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("restart error: %v", err)}, "failed"
	}
	if isDockerContainer(containerID) {
		log.Printf("[task-executor] restarted Docker container %s", containerID[:12])
	} else {
		log.Printf("[task-executor] restarted direct service %s", containerID)
	}

	agentID, _ := payload["agent_id"].(string)
	return map[string]interface{}{
		"agent_id":     agentID,
		"container_id": containerID,
		"status":       "running",
	}, "done"
}

func (e *TaskExecutor) ExecuteReloadPersona(payload map[string]interface{}) (map[string]interface{}, string) {
	agentName, _ := payload["agent_name"].(string)
	systemPrompt, _ := payload["system_prompt"].(string)
	background, _ := payload["background"].(string)
	if agentName == "" {
		return map[string]interface{}{"error": "missing agent_name in payload"}, "failed"
	}
	if systemPrompt == "" && background == "" {
		return map[string]interface{}{"error": "missing persona content in payload"}, "failed"
	}

	baseDir := fmt.Sprintf("/data/clawfleet/agents/%s/workspace", agentName)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("mkdir error: %v", err)}, "failed"
	}
	if systemPrompt != "" {
		if err := os.WriteFile(filepath.Join(baseDir, "SOUL.md"), []byte(systemPrompt), 0644); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("write SOUL.md error: %v", err)}, "failed"
		}
	}
	if background != "" {
		if err := os.WriteFile(filepath.Join(baseDir, "BACKGROUND.md"), []byte(background), 0644); err != nil {
			return map[string]interface{}{"error": fmt.Sprintf("write BACKGROUND.md error: %v", err)}, "failed"
		}
	}

	return map[string]interface{}{"status": "persona_reloaded"}, "done"
}

func (e *TaskExecutor) ExecuteStop(payload map[string]interface{}) (map[string]interface{}, string) {
	containerID, ok := payload["container_id"].(string)
	if !ok || containerID == "" {
		return map[string]interface{}{"error": "missing or invalid container_id in payload"}, "failed"
	}

	err := stopRuntime(context.Background(), e.container, e.installer, containerID)
	if err == nil {
		if isDockerContainer(containerID) {
			log.Printf("[task-executor] stopped Docker container %s", containerID[:12])
		} else {
			log.Printf("[task-executor] stopped direct service %s", containerID)
		}
	}

	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("stop error: %v", err)}, "failed"
	}

	return map[string]interface{}{"status": "stopped"}, "done"
}

func (e *TaskExecutor) ExecuteRemove(payload map[string]interface{}) (map[string]interface{}, string) {
	containerID, ok := payload["container_id"].(string)
	if !ok || containerID == "" {
		return map[string]interface{}{"error": "missing or invalid container_id in payload"}, "failed"
	}

	err := removeRuntime(context.Background(), e.container, e.installer, containerID)
	if err == nil {
		if isDockerContainer(containerID) {
			log.Printf("[task-executor] removed Docker container %s", containerID[:12])
		} else {
			log.Printf("[task-executor] removed container/service %s", containerID)
		}
	}

	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("remove error: %v", err)}, "failed"
	}

	return map[string]interface{}{"status": "removed"}, "done"
}

func (e *TaskExecutor) ExecuteDebugAgent(payload map[string]interface{}) (map[string]interface{}, string) {
	agentID, _ := payload["agent_id"].(string)
	agentName, _ := payload["agent_name"].(string)
	containerID, _ := payload["container_id"].(string)

	checks, err := parseDebugChecks(payload["checks"])
	if err != nil {
		return map[string]interface{}{
			"agent_id":     agentID,
			"agent_name":   agentName,
			"container_id": containerID,
			"error":        err.Error(),
		}, "failed"
	}

	results := make([]map[string]interface{}, 0, len(checks))
	allDone := true
	var totalOutput int

	for _, check := range checks {
		if _, ok := debugAllowedChecks[check]; !ok {
			allDone = false
			results = append(results, map[string]interface{}{
				"check":  check,
				"status": "failed",
				"error":  "unsupported debug check",
			})
			continue
		}

		output, runErr := e.runDebugCheck(check, containerID)
		output = sanitizeDebugOutput(output)
		output = capOutput(output, maxCheckOutputBytes)
		if totalOutput+len(output) > maxTaskOutputBytes {
			remaining := maxTaskOutputBytes - totalOutput
			if remaining < 0 {
				remaining = 0
			}
			output = capOutput(output, remaining)
		}
		totalOutput += len(output)

		item := map[string]interface{}{
			"check":  check,
			"output": output,
		}
		if runErr != nil {
			allDone = false
			item["status"] = "failed"
			item["error"] = runErr.Error()
		} else {
			item["status"] = "done"
		}
		results = append(results, item)
	}

	result := map[string]interface{}{
		"agent_id":     agentID,
		"agent_name":   agentName,
		"container_id": containerID,
		"checks":       checks,
		"results":      results,
	}
	if totalOutput > maxTaskOutputBytes {
		result["note"] = fmt.Sprintf("output truncated to %d bytes per check and %d bytes total", maxCheckOutputBytes, maxTaskOutputBytes)
	}

	if allDone {
		return result, "done"
	}
	return result, "failed"
}

func parseDebugChecks(v interface{}) ([]string, error) {
	switch vv := v.(type) {
	case []string:
		if len(vv) == 0 {
			return nil, fmt.Errorf("checks are required")
		}
		return vv, nil
	case []interface{}:
		if len(vv) == 0 {
			return nil, fmt.Errorf("checks are required")
		}
		out := make([]string, 0, len(vv))
		for _, raw := range vv {
			s, ok := raw.(string)
			if !ok || strings.TrimSpace(s) == "" {
				return nil, fmt.Errorf("checks must be non-empty strings")
			}
			out = append(out, strings.TrimSpace(s))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("checks are required")
	}
}

func (e *TaskExecutor) runDebugCheck(check, containerID string) (string, error) {
	switch check {
	case "openclaw_status_deep":
		return e.runCommandInRuntime(containerID, "openclaw status --deep")
	case "openclaw_plugins_list":
		return e.runCommandInRuntime(containerID, "openclaw plugins list --json")
	case "openclaw_config":
		return e.runCommandInRuntime(containerID, "cat /root/.openclaw/openclaw.json")
	case "openclaw_logs_tail":
		return e.runCommandInRuntime(containerID, "tail -n 200 /tmp/openclaw/openclaw-*.log")
	case "telegram_getme":
		script := `token="${OPENCLAW_TELEGRAM_TOKEN:-${TELEGRAM_BOT_TOKEN:-}}"
if [ -z "$token" ]; then
  echo "Telegram token not configured (OPENCLAW_TELEGRAM_TOKEN or TELEGRAM_BOT_TOKEN missing)"
  exit 1
fi
curl -fsS --max-time 20 "https://api.telegram.org/bot${token}/getMe"`
		return e.runCommandInRuntime(containerID, script)
	case "env_keys":
		return e.runEnvKeysCheck(containerID)
	default:
		return "", fmt.Errorf("unsupported debug check")
	}
}

func (e *TaskExecutor) runEnvKeysCheck(containerID string) (string, error) {
	out, err := e.runCommandInRuntime(containerID, "printenv")
	keys := map[string]bool{
		"OPENCLAW_TELEGRAM_TOKEN":   false,
		"TELEGRAM_BOT_TOKEN":        false,
		"CLAWFLEET_OPENCLAW_CONFIG": false,
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if _, ok := keys[parts[0]]; ok && strings.TrimSpace(parts[1]) != "" {
			keys[parts[0]] = true
		}
	}

	lines := make([]string, 0, len(keys))
	for _, key := range []string{"OPENCLAW_TELEGRAM_TOKEN", "TELEGRAM_BOT_TOKEN", "CLAWFLEET_OPENCLAW_CONFIG"} {
		state := "missing"
		if keys[key] {
			state = "present"
		}
		lines = append(lines, fmt.Sprintf("%s=%s", key, state))
	}
	return strings.Join(lines, "\n"), err
}

func (e *TaskExecutor) runCommandInRuntime(containerID, script string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), debugCheckTimeout)
	defer cancel()

	var cmd *exec.Cmd
	if isDockerContainer(containerID) {
		cmd = exec.CommandContext(ctx, "docker", "exec", containerID, "sh", "-lc", script)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-lc", script)
	}
	out, err := cmd.CombinedOutput()
	output := string(out)
	if err != nil {
		if output == "" {
			output = err.Error()
		}
		return output, fmt.Errorf("command failed")
	}
	return output, nil
}

func sanitizeDebugOutput(s string) string {
	clean := strings.ReplaceAll(s, "\r", "\n")
	clean = ansiPattern.ReplaceAllString(clean, "")
	clean = telegramBotURLPattern.ReplaceAllString(clean, "https://api.telegram.org/bot[REDACTED]")
	clean = jsonSecretPattern.ReplaceAllString(clean, `"$1":"[REDACTED]"`)
	clean = inlineSecretPattern.ReplaceAllString(clean, `$1$2[REDACTED]`)
	return clean
}

func capOutput(s string, maxBytes int) string {
	if maxBytes <= 0 || len(s) <= maxBytes {
		return s
	}
	return s[:maxBytes] + "\n...[truncated]..."
}

// isDockerContainer determines if an ID represents a Docker container or direct service
func isDockerContainer(id string) bool {
	// Docker container IDs are typically 64-character hex strings
	// Direct services use prefixes like "pid-", service names, etc.
	if len(id) == 64 {
		// Check if it's all hex
		for _, c := range id {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return false
			}
		}
		return true
	}
	// Could also be the short form (12 characters)
	if len(id) == 12 {
		for _, c := range id {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return false
			}
		}
		return true
	}
	return false
}

// Additional utility methods for the task executor can be added here as needed
