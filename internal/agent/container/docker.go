package container

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/NeoTheCapt/clawfleet/internal/adapter"
)

// Manager handles Docker container lifecycle via CLI.
type Manager struct{}

// NewManager creates a new container manager and verifies Docker is available.
func NewManager() (*Manager, error) {
	out, err := exec.Command("docker", "info", "--format", "{{.ServerVersion}}").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker not available: %w (%s)", err, string(out))
	}
	log.Printf("[container] Docker connected (server %s)", strings.TrimSpace(string(out)))
	return &Manager{}, nil
}

// Create pulls the image and creates + starts a container.
func (m *Manager) Create(ctx context.Context, name string, cfg *adapter.ContainerConfig) (string, error) {
	containerName := "clawfleet-" + SanitizeName(name)

	// Remove existing container with same name (stop + rm) to avoid conflicts on restart/redeploy
	if out, err := exec.CommandContext(ctx, "docker", "rm", "-f", containerName).CombinedOutput(); err == nil {
		log.Printf("[container] removed existing container %s", containerName)
	} else {
		_ = out // container didn't exist, that's fine
	}

	// Pull image (skip if local-only image exists)
	log.Printf("[container] pulling image %s", cfg.Image)
	// Check if image exists locally first
	if out, err := exec.CommandContext(ctx, "docker", "image", "inspect", cfg.Image).CombinedOutput(); err != nil {
		// Image not local, try pulling
		if out, err := exec.CommandContext(ctx, "docker", "pull", cfg.Image).CombinedOutput(); err != nil {
			return "", fmt.Errorf("pull image: %w (%s)", err, string(out))
		}
	} else {
		log.Printf("[container] image %s found locally, skipping pull", cfg.Image)
		_ = out
	}

	// Build docker run args
	args := []string{"run", "-d", "--name", containerName}

	// Env vars
	for k, v := range cfg.Env {
		args = append(args, "-e", k+"="+v)
	}

	// Labels
	for k, v := range cfg.Labels {
		args = append(args, "-l", k+"="+v)
	}

	// Ports
	for hostPort, containerPort := range cfg.Ports {
		args = append(args, "-p", hostPort+":"+containerPort)
	}

	// Volumes
	for hostPath, containerPath := range cfg.Volumes {
		args = append(args, "-v", hostPath+":"+containerPath)
	}

	// Resource limits
	if cfg.CPULimit > 0 {
		cpus := float64(cfg.CPULimit) / 1000.0
		args = append(args, "--cpus", fmt.Sprintf("%.1f", cpus))
	}
	if cfg.MemoryLimit > 0 {
		args = append(args, "--memory", fmt.Sprintf("%d", cfg.MemoryLimit))
	}

	// User
	if cfg.User != "" {
		args = append(args, "--user", cfg.User)
	}

	// Restart policy
	args = append(args, "--restart", "unless-stopped")

	// Image
	args = append(args, cfg.Image)

	// Cmd
	args = append(args, cfg.Cmd...)

	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("create container: %w (%s)", err, string(out))
	}

	containerID := strings.TrimSpace(string(out))
	log.Printf("[container] started %s (id=%s)", containerName, containerID[:12])
	return containerID, nil
}

// Restart restarts a running container.
func (m *Manager) Restart(ctx context.Context, containerID string) error {
	out, err := exec.CommandContext(ctx, "docker", "restart", containerID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("restart: %w (%s)", err, string(out))
	}
	return nil
}

// Stop stops a running container.
func (m *Manager) Stop(ctx context.Context, containerID string) error {
	out, err := exec.CommandContext(ctx, "docker", "stop", containerID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop: %w (%s)", err, string(out))
	}
	return nil
}

// Remove force removes a container.
func (m *Manager) Remove(ctx context.Context, containerID string) error {
	out, err := exec.CommandContext(ctx, "docker", "rm", "-f", containerID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove: %w (%s)", err, string(out))
	}
	return nil
}

// Status returns the container status.
func (m *Manager) Status(ctx context.Context, containerID string) (string, error) {
	out, err := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Status}}", containerID).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect: %w (%s)", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

// Logs returns the last N lines of container logs.
func (m *Manager) Logs(ctx context.Context, containerID string, tail int) (string, error) {
	out, err := exec.CommandContext(ctx, "docker", "logs", "--tail", fmt.Sprintf("%d", tail), containerID).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("logs: %w (%s)", err, string(out))
	}
	return string(out), nil
}

func SanitizeName(name string) string {
	// Docker container name must match: [a-zA-Z0-9][a-zA-Z0-9_.-]+
	// We normalize to lowercase and replace any unsupported char with '-'.
	name = strings.ToLower(name)
	var b strings.Builder
	for _, ch := range name {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' || ch == '.'
		if ok {
			b.WriteRune(ch)
		} else {
			b.WriteByte('-')
		}
	}
	out := b.String()
	// Ensure first char is alnum
	if out != "" {
		c := out[0]
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			out = "a" + out
		}
	}
	// Collapse repeated dashes
	out = strings.Trim(out, "-_.")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	if out == "" {
		out = "agent"
	}
	return out
}
