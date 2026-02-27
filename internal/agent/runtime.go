package agent

import (
	"context"

	"github.com/NeoTheCapt/clawfleet/internal/agent/container"
	"github.com/NeoTheCapt/clawfleet/internal/agent/direct"
)

type runtimeKind int

const (
	runtimeDocker runtimeKind = iota
	runtimeDirect
)

func detectRuntime(containerID string) runtimeKind {
	if isDockerContainer(containerID) {
		return runtimeDocker
	}
	return runtimeDirect
}

// stopRuntime stops either docker container or direct service.
func stopRuntime(ctx context.Context, cm container.ContainerManagerInterface, inst *direct.Installer, containerID string) error {
	if detectRuntime(containerID) == runtimeDocker {
		return cm.Stop(ctx, containerID)
	}
	return inst.Stop(containerID)
}

// restartRuntime restarts either docker container or direct service.
func restartRuntime(ctx context.Context, cm container.ContainerManagerInterface, inst *direct.Installer, containerID string) error {
	if detectRuntime(containerID) == runtimeDocker {
		return cm.Restart(ctx, containerID)
	}
	// Direct: best-effort stop then install/start
	_ = inst.Stop(containerID)
	if _, err := inst.Install(nil); err != nil {
		return err
	}
	return nil
}

// removeRuntime removes either docker container/name or direct service.
func removeRuntime(ctx context.Context, cm container.ContainerManagerInterface, inst *direct.Installer, containerID string) error {
	if detectRuntime(containerID) == runtimeDocker {
		return cm.Remove(ctx, containerID)
	}
	// Try docker by name first (supports container names), then fallback to direct remove
	if err := cm.Remove(ctx, containerID); err == nil {
		return nil
	}
	return inst.Remove(containerID)
}
