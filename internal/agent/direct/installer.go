package direct

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/adapter"
)

// Installer handles direct installation of agents (without Docker).
type Installer struct{}

// NewInstaller creates a new direct installer.
func NewInstaller() *Installer {
	return &Installer{}
}

// Install runs the installation commands and starts the agent service.
func (i *Installer) Install(cfg *adapter.DirectInstallConfig) (string, error) {
	log.Printf("[direct] installing agent with %d commands", len(cfg.InstallCommands))

	// Create working directory
	if cfg.WorkDir != "" {
		if err := os.MkdirAll(cfg.WorkDir, 0755); err != nil {
			return "", fmt.Errorf("create workdir %s: %w", cfg.WorkDir, err)
		}
	}

	// Write config files
	for path, content := range cfg.ConfigFiles {
		if !filepath.IsAbs(path) && cfg.WorkDir != "" {
			path = filepath.Join(cfg.WorkDir, path)
		}

		// Create directory if needed
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return "", fmt.Errorf("create config dir %s: %w", filepath.Dir(path), err)
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("write config file %s: %w", path, err)
		}
		log.Printf("[direct] wrote config file: %s", path)
	}

	// Run install commands
	for j, cmd := range cfg.InstallCommands {
		log.Printf("[direct] running install command %d/%d: %s", j+1, len(cfg.InstallCommands), cmd)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		var execCmd *exec.Cmd
		if strings.Contains(cmd, "&&") || strings.Contains(cmd, "||") || strings.Contains(cmd, "|") {
			execCmd = exec.CommandContext(ctx, "bash", "-c", cmd)
		} else {
			parts := strings.Fields(cmd)
			if len(parts) == 0 {
				continue
			}
			execCmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
		}

		if cfg.WorkDir != "" {
			execCmd.Dir = cfg.WorkDir
		}

		// Set environment variables
		execCmd.Env = os.Environ()
		for k, v := range cfg.Env {
			execCmd.Env = append(execCmd.Env, k+"="+v)
		}

		out, err := execCmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("install command failed: %w (%s)", err, string(out))
		}
		log.Printf("[direct] install command output: %s", string(out))
	}

	// If systemd service requested, create it
	if cfg.Systemd && cfg.ServiceName != "" {
		if err := i.createSystemdService(cfg); err != nil {
			return "", fmt.Errorf("create systemd service: %w", err)
		}
	}

	// Start the service
	serviceID, err := i.startService(cfg)
	if err != nil {
		return "", fmt.Errorf("start service: %w", err)
	}

	log.Printf("[direct] agent installed and started with ID: %s", serviceID)
	return serviceID, nil
}

// createSystemdService creates a systemd unit file for the agent.
func (i *Installer) createSystemdService(cfg *adapter.DirectInstallConfig) error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=Clawfleet Agent: %s
After=network.target
Wants=network.target

[Service]
Type=simple
ExecStart=%s
Restart=always
RestartSec=5
User=root
WorkingDirectory=%s
`,
		cfg.ServiceName,
		cfg.StartCommand,
		cfg.WorkDir,
	)

	// Add environment variables
	for k, v := range cfg.Env {
		serviceContent += fmt.Sprintf("Environment=\"%s=%s\"\n", k, v)
	}

	serviceContent += `
[Install]
WantedBy=multi-user.target
`

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", cfg.ServiceName)
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("write systemd service: %w", err)
	}

	// Reload systemd
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w (%s)", err, string(out))
	}

	// Enable service
	if out, err := exec.Command("systemctl", "enable", cfg.ServiceName).CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl enable: %w (%s)", err, string(out))
	}

	log.Printf("[direct] created systemd service: %s", cfg.ServiceName)
	return nil
}

// startService starts the agent service and returns a service ID.
func (i *Installer) startService(cfg *adapter.DirectInstallConfig) (string, error) {
	if cfg.Systemd && cfg.ServiceName != "" {
		// Start systemd service
		if out, err := exec.Command("systemctl", "start", cfg.ServiceName).CombinedOutput(); err != nil {
			return "", fmt.Errorf("systemctl start: %w (%s)", err, string(out))
		}
		return cfg.ServiceName, nil
	}

	// Direct process start (background)
	ctx := context.Background()
	var cmd *exec.Cmd

	if strings.Contains(cfg.StartCommand, "&&") || strings.Contains(cfg.StartCommand, "||") || strings.Contains(cfg.StartCommand, "|") {
		cmd = exec.CommandContext(ctx, "bash", "-c", cfg.StartCommand)
	} else {
		parts := strings.Fields(cfg.StartCommand)
		if len(parts) == 0 {
			return "", fmt.Errorf("empty start command")
		}
		cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
	}

	if cfg.WorkDir != "" {
		cmd.Dir = cfg.WorkDir
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start process: %w", err)
	}

	// Return process ID as string
	pid := fmt.Sprintf("pid-%d", cmd.Process.Pid)
	log.Printf("[direct] started process with PID: %d", cmd.Process.Pid)
	return pid, nil
}

// Stop stops the agent service.
func (i *Installer) Stop(serviceID string) error {
	if strings.HasPrefix(serviceID, "pid-") {
		// Kill process by PID
		pidStr := strings.TrimPrefix(serviceID, "pid-")
		out, err := exec.Command("kill", pidStr).CombinedOutput()
		if err != nil {
			return fmt.Errorf("kill process: %w (%s)", err, string(out))
		}
		return nil
	}

	// Assume it's a systemd service
	out, err := exec.Command("systemctl", "stop", serviceID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl stop: %w (%s)", err, string(out))
	}
	return nil
}

// Remove removes the agent completely.
func (i *Installer) Remove(serviceID string) error {
	// Stop first
	if err := i.Stop(serviceID); err != nil {
		log.Printf("[direct] failed to stop service %s: %v", serviceID, err)
	}

	if !strings.HasPrefix(serviceID, "pid-") {
		// Remove systemd service
		if out, err := exec.Command("systemctl", "disable", serviceID).CombinedOutput(); err != nil {
			log.Printf("[direct] failed to disable service %s: %v (%s)", serviceID, err, string(out))
		}
		servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceID)
		if err := os.Remove(servicePath); err != nil && !os.IsNotExist(err) {
			log.Printf("[direct] failed to remove service file %s: %v", servicePath, err)
		}
		if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
			log.Printf("[direct] failed to reload systemd daemon: %v (%s)", err, string(out))
		}
		log.Printf("[direct] removed systemd service: %s", serviceID)
	}

	return nil
}

// Status returns the service status.
func (i *Installer) Status(serviceID string) (string, error) {
	if strings.HasPrefix(serviceID, "pid-") {
		// Check if process is running
		pidStr := strings.TrimPrefix(serviceID, "pid-")
		out, err := exec.Command("ps", "-p", pidStr, "-o", "stat=").CombinedOutput()
		if err != nil {
			return "stopped", nil
		}
		status := strings.TrimSpace(string(out))
		if status != "" {
			return "running", nil
		}
		return "stopped", nil
	}

	// Check systemd service
	out, err := exec.Command("systemctl", "is-active", serviceID).CombinedOutput()
	if err != nil {
		return "stopped", nil
	}

	status := strings.TrimSpace(string(out))
	switch status {
	case "active":
		return "running", nil
	case "inactive":
		return "stopped", nil
	case "failed":
		return "error", nil
	default:
		return status, nil
	}
}

// Logs returns service logs.
func (i *Installer) Logs(serviceID string, tail int) (string, error) {
	if strings.HasPrefix(serviceID, "pid-") {
		// For direct processes, we can't easily get logs
		return "Logs not available for direct processes", nil
	}

	// Get systemd service logs
	out, err := exec.Command("journalctl", "-u", serviceID, "-n", fmt.Sprintf("%d", tail), "--no-pager").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("journalctl: %w (%s)", err, string(out))
	}
	return string(out), nil
}
