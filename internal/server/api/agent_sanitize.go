package api

import "github.com/NeoTheCapt/clawfleet/internal/model"

const imKeyEnvVar = "CLAWFLEET_IM_KEY"

func sanitizeAgentConfigForResponse(cfg model.AgentConfig) model.AgentConfig {
	safe := cfg
	if len(cfg.EnvVars) == 0 {
		return safe
	}
	env := make(map[string]string, len(cfg.EnvVars))
	for k, v := range cfg.EnvVars {
		env[k] = v
	}
	delete(env, imKeyEnvVar)
	safe.EnvVars = env
	return safe
}

func sanitizeAgentForResponse(agent *model.AgentInstance) *model.AgentInstance {
	if agent == nil {
		return nil
	}
	safe := *agent
	safe.Config = sanitizeAgentConfigForResponse(agent.Config)
	return &safe
}

func sanitizeAgentsForResponse(agents []*model.AgentInstance) []*model.AgentInstance {
	if len(agents) == 0 {
		return agents
	}
	out := make([]*model.AgentInstance, 0, len(agents))
	for _, a := range agents {
		out = append(out, sanitizeAgentForResponse(a))
	}
	return out
}
