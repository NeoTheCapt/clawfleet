package adapter

import "github.com/NeoTheCapt/clawfleet/internal/model"

// DeployMode determines how an agent is deployed
type DeployMode string

const (
	DeployModeDocker  DeployMode = "docker"
	DeployModeDirect  DeployMode = "direct"
)

// ContainerConfig describes how to run an agent container.
type ContainerConfig struct {
	Image      string            `json:"image"`
	Cmd        []string          `json:"cmd,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Volumes    map[string]string `json:"volumes,omitempty"` // host:container
	Ports      map[string]string `json:"ports,omitempty"`   // hostPort:containerPort
	CPULimit   int64             `json:"cpu_limit,omitempty"`   // in millicores
	MemoryLimit int64            `json:"memory_limit,omitempty"` // in bytes
	Labels     map[string]string `json:"labels,omitempty"`
	User       string            `json:"user,omitempty"`       // e.g. "0" for root, "65534" for nobody
	InitCmd    []string          `json:"init_cmd,omitempty"`   // run before main cmd (e.g. mkdir/chown)
}

// DirectInstallConfig describes how to install an agent directly
type DirectInstallConfig struct {
	// Commands to run to install the agent (in order)
	InstallCommands []string
	// Command to start the agent
	StartCommand    string
	// Command to stop the agent  
	StopCommand     string
	// Environment variables
	Env             map[string]string
	// Working directory
	WorkDir         string
	// Config files to write (path -> content)
	ConfigFiles     map[string]string
	// Whether to create a systemd service
	Systemd         bool
	ServiceName     string
}

// Adapter defines how to manage a specific agent framework.
type Adapter interface {
	// Name returns the adapter name (e.g. "openclaw").
	Name() string
	
	// SupportedModes returns the deployment modes this adapter supports
	SupportedModes() []DeployMode
	
	// Docker mode
	ContainerConfig(instance *model.AgentInstance) (*ContainerConfig, error)
	
	// Direct install mode
	InstallConfig(instance *model.AgentInstance) (*DirectInstallConfig, error)

	// HealthCheck checks if the agent is healthy.
	HealthCheck(containerID string) (bool, error)
}

// Registry holds all registered adapters.
type Registry struct {
	adapters map[model.AgentType]Adapter
}

// NewRegistry creates a registry with all built-in adapters.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[model.AgentType]Adapter),
	}
}

// Register adds an adapter.
func (r *Registry) Register(agentType model.AgentType, a Adapter) {
	r.adapters[agentType] = a
}

// Get returns the adapter for a given agent type.
func (r *Registry) Get(agentType model.AgentType) (Adapter, bool) {
	a, ok := r.adapters[agentType]
	return a, ok
}
