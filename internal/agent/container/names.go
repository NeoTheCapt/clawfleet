package container

// Version-safe helpers for naming.

// ContainerName returns the deterministic container name used for a given agent name.
func ContainerName(agentName string) string {
	return "clawfleet-" + SanitizeName(agentName)
}
