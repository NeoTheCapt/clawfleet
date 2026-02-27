package nanobot

import (
	"fmt"

	"github.com/NeoTheCapt/clawfleet/internal/adapter"
	"github.com/NeoTheCapt/clawfleet/internal/model"
)

type Adapter struct {
	Image string
}

func New(image string) *Adapter {
	if image == "" {
		image = "ghcr.io/astral-sh/uv:python3.12-bookworm-slim"
	}
	return &Adapter{Image: image}
}

func (a *Adapter) Name() string { return "nanobot" }

func (a *Adapter) SupportedModes() []adapter.DeployMode {
	return []adapter.DeployMode{adapter.DeployModeDocker, adapter.DeployModeDirect}
}

func (a *Adapter) ContainerConfig(instance *model.AgentInstance) (*adapter.ContainerConfig, error) {
	env := adapter.BaseEnv(instance)

	// Map clawfleet config to Nanobot env vars
	if instance.Config.Model != "" {
		env["NANOBOT_MODEL"] = instance.Config.Model
	}

	// User-provided env vars (API keys, provider config, etc.)
	env = adapter.WithUserEnv(env, instance.Config.EnvVars)

	dataDir := fmt.Sprintf("/data/clawfleet/agents/%s/nanobot", instance.Name)

	cfg := &adapter.ContainerConfig{
		Image: a.Image,
		Cmd:   []string{"sh", "-c", "pip install nanobot-ai && nanobot gateway"},
		Env:   env,
		Volumes: map[string]string{
			dataDir: "/root/.nanobot",
		},
		Ports: map[string]string{
			"0": "18790", // random host port -> nanobot gateway
		},
		Labels: adapter.Labels(instance, "nanobot"),
	}

	if instance.Config.Resources.CPULimit != "" {
		cfg.CPULimit = adapter.ParseCPU(instance.Config.Resources.CPULimit)
	}
	if instance.Config.Resources.MemoryLimit != "" {
		cfg.MemoryLimit = adapter.ParseMemory(instance.Config.Resources.MemoryLimit)
	}

	return cfg, nil
}

func (a *Adapter) InstallConfig(instance *model.AgentInstance) (*adapter.DirectInstallConfig, error) {
	env := adapter.BaseEnv(instance)
	if instance.Config.Model != "" {
		env["NANOBOT_MODEL"] = instance.Config.Model
	}
	env = adapter.WithUserEnv(env, instance.Config.EnvVars)

	workDir := adapter.WorkDir(instance)
	serviceName := adapter.ServiceName(instance)

	return &adapter.DirectInstallConfig{
		InstallCommands: []string{
			"pip install nanobot-ai",
		},
		StartCommand: "nanobot gateway",
		StopCommand:  "pkill -f 'nanobot gateway'",
		Env:          env,
		WorkDir:      workDir,
		ConfigFiles:  map[string]string{},
		Systemd:      true,
		ServiceName:  serviceName,
	}, nil
}

func (a *Adapter) HealthCheck(containerID string) (bool, error) { return true, nil }
