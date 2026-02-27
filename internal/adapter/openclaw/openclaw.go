package openclaw

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/NeoTheCapt/clawfleet/internal/util"

	"github.com/NeoTheCapt/clawfleet/internal/adapter"
	"github.com/NeoTheCapt/clawfleet/internal/model"
)

type Adapter struct {
	Image string
}

func New(image string) *Adapter {
	if image == "" {
		image = "node:22-bookworm"
	}
	return &Adapter{Image: image}
}

func (a *Adapter) Name() string { return "openclaw" }

func (a *Adapter) SupportedModes() []adapter.DeployMode {
	return []adapter.DeployMode{adapter.DeployModeDocker, adapter.DeployModeDirect}
}

// generateToken creates a random hex token for gateway auth
func generateToken() string {
	// 16 bytes is plenty for gateway auth; reuse shared util.
	return util.GenerateToken()[:32]
}

func (a *Adapter) ContainerConfig(instance *model.AgentInstance) (*adapter.ContainerConfig, error) {
	gatewayToken := generateToken()

	env := adapter.BaseEnv(instance)
	env["NODE_ENV"] = "production"
	// Bind loopback inside the container. We don't need the Control UI exposed on LAN,
	// and non-loopback binds require extra controlUi origin config.
	env["OPENCLAW_GATEWAY_BIND"] = "loopback"
	env["OPENCLAW_GATEWAY_TOKEN"] = gatewayToken

	// Model: ensure provider prefix (e.g. "openrouter/deepseek/deepseek-chat")
	modelStr := instance.Config.Model
	if modelStr != "" {
		if instance.Config.Provider != "" && !strings.HasPrefix(modelStr, instance.Config.Provider+"/") {
			modelStr = instance.Config.Provider + "/" + modelStr
		}
		env["OPENCLAW_DEFAULT_MODEL"] = modelStr
	}

	// Channel configuration → env vars
	// Note: OpenClaw has had multiple env var names across versions.
	// We set the common aliases to maximize compatibility.
	if ch := instance.Config.Channel; ch != nil {
		switch ch.Type {
		case "telegram":
			if ch.Token != "" {
				// Newer/OpenClaw-native
				env["OPENCLAW_TELEGRAM_TOKEN"] = ch.Token
				// Common alias used by some builds
				env["TELEGRAM_BOT_TOKEN"] = ch.Token
			}
		case "discord":
			if ch.Token != "" {
				env["DISCORD_BOT_TOKEN"] = ch.Token
			}
		case "slack":
			if ch.BotToken != "" {
				env["SLACK_BOT_TOKEN"] = ch.BotToken
			}
			if ch.AppToken != "" {
				env["SLACK_APP_TOKEN"] = ch.AppToken
			}
		}
	}

	// User-provided env vars (API keys, etc.) — these override auto-generated ones
	env = adapter.WithUserEnv(env, instance.Config.EnvVars)

	ocHome := "/root/.openclaw"
	dataDir := fmt.Sprintf("/data/clawfleet/agents/%s/openclaw", instance.Name)
	workspaceDir := fmt.Sprintf("/data/clawfleet/agents/%s/workspace", instance.Name)

	// Build openclaw.json config
	configJSON := buildOpenClawConfig(instance, modelStr)
	env["CLAWFLEET_OPENCLAW_CONFIG"] = configJSON

	// Persist persona into workspace on container start (so agent "knows who it is" on first boot/redeploy).
	// Use base64 to avoid shell quoting issues.
	if sp := strings.TrimSpace(instance.Config.SystemPrompt); sp != "" {
		env["CLAWFLEET_SOUL_B64"] = base64.StdEncoding.EncodeToString([]byte(sp))
	}
	if bg := strings.TrimSpace(instance.Config.Description); bg != "" {
		env["CLAWFLEET_BACKGROUND_B64"] = base64.StdEncoding.EncodeToString([]byte(bg))
	}

	// Startup: write config + persona, install openclaw (once), ensure gateway.mode is set, start gateway.
	// IMPORTANT: do NOT run `npm install -g openclaw@latest` on every boot; it can race and fail (ENOTEMPTY).
	startCmd := fmt.Sprintf(`set -e; mkdir -p %s/workspace && \
  if [ -n "$CLAWFLEET_SOUL_B64" ]; then printf "%%s" "$CLAWFLEET_SOUL_B64" | base64 -d 2>/dev/null > %s/workspace/SOUL.md || true; fi && \
  if [ -n "$CLAWFLEET_BACKGROUND_B64" ]; then printf "%%s" "$CLAWFLEET_BACKGROUND_B64" | base64 -d 2>/dev/null > %s/workspace/BACKGROUND.md || true; fi && \
  echo "$CLAWFLEET_OPENCLAW_CONFIG" > %s/openclaw.json && \
  chmod 700 %s && chmod 600 %s/openclaw.json && \
  (command -v openclaw >/dev/null 2>&1 || npm install -g openclaw@latest 2>&1) && \
  (openclaw config set gateway.mode local || true) && \
  (openclaw plugins enable telegram || true) && \
  (openclaw plugins enable clawfleet-im || true) && \
  openclaw gateway --allow-unconfigured --bind loopback`, ocHome, ocHome, ocHome, ocHome, ocHome, ocHome)

	cfg := &adapter.ContainerConfig{
		Image: a.Image,
		User:  "0",
		Cmd:   []string{"sh", "-c", startCmd},
		Env:   env,
		Volumes: map[string]string{
			dataDir:      ocHome,
			workspaceDir: ocHome + "/workspace",
		},
		Ports: map[string]string{
			"0": "18789",
		},
		Labels: adapter.Labels(instance, "openclaw"),
	}

	if instance.Config.Resources.CPULimit != "" {
		cfg.CPULimit = adapter.ParseCPU(instance.Config.Resources.CPULimit)
	}
	if instance.Config.Resources.MemoryLimit != "" {
		cfg.MemoryLimit = adapter.ParseMemory(instance.Config.Resources.MemoryLimit)
	}

	return cfg, nil
}

func buildOpenClawConfig(instance *model.AgentInstance, modelStr string) string {
	// Build channels config
	channelsCfg := ""
	if ch := instance.Config.Channel; ch != nil {
		switch ch.Type {
		case "telegram":
			dmPolicy := ch.DMPolicy
			if dmPolicy == "" {
				dmPolicy = "open"
			}
			// Build allowFrom list
			allowItems := []string{}
			if dmPolicy == "open" {
				// "open" requires "*" in allowFrom
				allowItems = append(allowItems, `"*"`)
			}
			for _, a := range ch.AllowFrom {
				if a != "*" {
					allowItems = append(allowItems, fmt.Sprintf("%q", a))
				}
			}
			allowFrom := fmt.Sprintf(`, "allowFrom": [%s]`, strings.Join(allowItems, ","))
			channelsCfg = fmt.Sprintf(`"channels": { "telegram": { "enabled": true, "dmPolicy": %q%s } }`, dmPolicy, allowFrom)
		case "discord":
			channelsCfg = `"channels": { "discord": { "enabled": true } }`
		case "slack":
			channelsCfg = `"channels": { "slack": { "enabled": true } }`
		}
	}

	// Build model config
	modelCfg := ""
	if modelStr != "" {
		fallbacks := ""
		if len(instance.Config.FallbackModels) > 0 {
			items := make([]string, len(instance.Config.FallbackModels))
			for i, m := range instance.Config.FallbackModels {
				items[i] = fmt.Sprintf("%q", m)
			}
			fallbacks = fmt.Sprintf(`, "fallbacks": [%s]`, strings.Join(items, ","))
		}
		modelCfg = fmt.Sprintf(`"agents": { "defaults": { "model": { "primary": %q%s } } }`, modelStr, fallbacks)
	}

	// Gateway config (required to avoid interactive configure; matches `openclaw config set gateway.mode local`).
	// Also set the Control UI origin fallback so future OpenClaw versions don't refuse non-loopback binds.
	gatewayCfg := `"gateway": { "mode": "local", "controlUi": { "dangerouslyAllowHostHeaderOriginFallback": true } }`

	// Plugins: bundled plugins are disabled by default in OpenClaw.
	// We enable telegram (existing) + clawfleet-im (bridge).
	pluginsCfg := `"plugins": { "enabled": true, "allow": ["telegram", "clawfleet-im"], "entries": { "telegram": { "enabled": true }, "clawfleet-im": { "enabled": true } } }`

	// Optional Clawfleet IM channel (used by clawfleet-im plugin)
	clawfleetChanCfg := ""
	if k := strings.TrimSpace(instance.Config.EnvVars["CLAWFLEET_IM_KEY"]); k != "" {
		baseURL := strings.TrimSpace(instance.Config.EnvVars["CLAWFLEET_CP_BASE_URL"])
			// Avoid hardcoding a real domain in OSS. For production you should set
			// CLAWFLEET_CP_BASE_URL explicitly (e.g. https://control-plane.example.com).
			if baseURL == "" {
				baseURL = "http://localhost:8090"
			}
		clawfleetChanCfg = fmt.Sprintf(`"channels": { "clawfleet": { "accounts": { "default": { "enabled": true, "name": %q, "baseUrl": %q, "imKey": %q, "pollIntervalMs": 800 } } } }`, instance.Config.PersonaName, baseURL, k)
	}

	// Combine
	parts := []string{gatewayCfg, pluginsCfg}
	if clawfleetChanCfg != "" {
		parts = append(parts, clawfleetChanCfg)
	}
	if modelCfg != "" {
		parts = append(parts, modelCfg)
	}
	if channelsCfg != "" {
		parts = append(parts, channelsCfg)
	}

	return "{" + strings.Join(parts, ", ") + "}"
}

func (a *Adapter) InstallConfig(instance *model.AgentInstance) (*adapter.DirectInstallConfig, error) {
	env := adapter.BaseEnv(instance)
	env["NODE_ENV"] = "production"
	if instance.Config.Model != "" {
		env["OPENCLAW_DEFAULT_MODEL"] = instance.Config.Model
	}
	env = adapter.WithUserEnv(env, instance.Config.EnvVars)

	workDir := adapter.WorkDir(instance)
	serviceName := adapter.ServiceName(instance)

	return &adapter.DirectInstallConfig{
		InstallCommands: []string{
			"curl -fsSL https://deb.nodesource.com/setup_22.x | bash -",
			"apt-get install -y nodejs",
			"npm install -g openclaw@latest",
		},
		StartCommand: "openclaw gateway --allow-unconfigured --bind lan",
		StopCommand:  "openclaw gateway stop",
		Env:          env,
		WorkDir:      workDir,
		ConfigFiles:  map[string]string{},
		Systemd:      true,
		ServiceName:  serviceName,
	}, nil
}

func (a *Adapter) HealthCheck(containerID string) (bool, error) { return true, nil }
