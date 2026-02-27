package zeroclaw

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
		image = "ghcr.io/theonlyhennygod/zeroclaw:latest"
	}
	return &Adapter{Image: image}
}

func (a *Adapter) Name() string { return "zeroclaw" }

func (a *Adapter) SupportedModes() []adapter.DeployMode {
	return []adapter.DeployMode{adapter.DeployModeDocker, adapter.DeployModeDirect}
}

func (a *Adapter) ContainerConfig(instance *model.AgentInstance) (*adapter.ContainerConfig, error) {
	env := adapter.BaseEnv(instance)
	env["ZEROCLAW_ALLOW_PUBLIC_BIND"] = "true"

	// Map clawfleet config to ZeroClaw env vars
	if instance.Config.Model != "" {
		env["ZEROCLAW_MODEL"] = instance.Config.Model
	}

	// User-provided env vars (API_KEY, PROVIDER, etc.)
	env = adapter.WithUserEnv(env, instance.Config.EnvVars)

	dataDir := fmt.Sprintf("/data/clawfleet/agents/%s/zeroclaw-data", instance.Name)

	cfg := &adapter.ContainerConfig{
		Image: a.Image,
		// Run as root to avoid volume permission issues; container is already sandboxed
		User: "0",
		Cmd:  []string{"daemon"},
		Env:   env,
		Volumes: map[string]string{
			dataDir: "/zeroclaw-data",
		},
		Ports: map[string]string{
			"0": "3000", // random host port -> zeroclaw gateway
		},
		Labels: adapter.Labels(instance, "zeroclaw"),
	}

	if instance.Config.Resources.CPULimit != "" {
		cfg.CPULimit = adapter.ParseCPU(instance.Config.Resources.CPULimit)
	} else {
		cfg.CPULimit = 2000 // default 2 CPUs
	}
	if instance.Config.Resources.MemoryLimit != "" {
		cfg.MemoryLimit = adapter.ParseMemory(instance.Config.Resources.MemoryLimit)
	} else {
		cfg.MemoryLimit = 2 * 1024 * 1024 * 1024 // default 2Gi
	}

	return cfg, nil
}

func (a *Adapter) InstallConfig(instance *model.AgentInstance) (*adapter.DirectInstallConfig, error) {
	env := adapter.BaseEnv(instance)
	if instance.Config.Model != "" {
		env["ZEROCLAW_MODEL"] = instance.Config.Model
	}
	env = adapter.WithUserEnv(env, instance.Config.EnvVars)

	workDir := adapter.WorkDir(instance)
	serviceName := adapter.ServiceName(instance)

	return &adapter.DirectInstallConfig{
		InstallCommands: []string{
			"curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y",
			"export PATH=\"$HOME/.cargo/bin:$PATH\"",
			"git clone https://github.com/zeroclaw-labs/zeroclaw.git /tmp/zeroclaw-build",
			"cd /tmp/zeroclaw-build && cargo install --path . --force --locked",
			"rm -rf /tmp/zeroclaw-build",
		},
		StartCommand: "zeroclaw daemon",
		StopCommand:  "zeroclaw gateway stop",
		Env:          env,
		WorkDir:      workDir,
		ConfigFiles:  map[string]string{},
		Systemd:      true,
		ServiceName:  serviceName,
	}, nil
}

func (a *Adapter) HealthCheck(containerID string) (bool, error) { return true, nil }
