package adapter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ParseCPU converts CPU string (e.g. "2") to millicores
func ParseCPU(s string) int64 {
	c, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	if c > 0 {
		return int64(c) * 1000
	}
	return 0
}

// ParseMemory converts memory string (e.g. "2Gi", "512Mi") to bytes
func ParseMemory(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var val int64
	var unit string
	n, err := fmt.Sscanf(s, "%d%s", &val, &unit)
	if err != nil || n == 0 {
		return 0
	}
	switch unit {
	case "Gi":
		return val * 1024 * 1024 * 1024
	case "Mi":
		return val * 1024 * 1024
	default:
		return val
	}
}

// WithUserEnv merges user env vars into base env (user values win)
func WithUserEnv(env map[string]string, user map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range env {
		out[k] = v
	}
	for k, v := range user {
		out[k] = v
	}
	return out
}

// BaseEnv includes common clawfleet metadata
func BaseEnv(instance *model.AgentInstance) map[string]string {
	return map[string]string{
		"CLAWFLEET_AGENT_ID":   instance.ID,
		"CLAWFLEET_AGENT_NAME": instance.Name,
		"CLAWFLEET_AGENT_ROLE": instance.Role,
	}
}

// Labels returns standard container labels
func Labels(instance *model.AgentInstance, agentType string) map[string]string {
	return map[string]string{
		"clawfleet.agent.id":   instance.ID,
		"clawfleet.agent.name": instance.Name,
		"clawfleet.agent.type": agentType,
		"clawfleet.cluster.id": instance.CompanyID,
		"managed-by":           "clawfleet",
	}
}

func WorkDir(instance *model.AgentInstance) string {
	return fmt.Sprintf("/data/clawfleet/agents/%s", instance.Name)
}

func ServiceName(instance *model.AgentInstance) string {
	return fmt.Sprintf("clawfleet-agent-%s", instance.Name)
}
