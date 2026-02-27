package container

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/adapter"
)

// MockManager simulates Docker for testing without a real Docker daemon.
type MockManager struct {
	mu         sync.Mutex
	containers map[string]*mockContainer
}

type mockContainer struct {
	ID     string
	Name   string
	Image  string
	State  string
	Labels map[string]string
}

func NewMockManager() *MockManager {
	log.Println("[container] Mock Docker manager initialized (no real containers)")
	return &MockManager{
		containers: make(map[string]*mockContainer),
	}
}

func (m *MockManager) Create(ctx context.Context, name string, cfg *adapter.ContainerConfig) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	containerName := "clawfleet-" + SanitizeName(name)
	id := fmt.Sprintf("mock_%s_%d", SanitizeName(name), time.Now().UnixNano())

	m.containers[id] = &mockContainer{
		ID:     id,
		Name:   containerName,
		Image:  cfg.Image,
		State:  "running",
		Labels: cfg.Labels,
	}

	log.Printf("[container-mock] created %s (id=%s, image=%s)", containerName, id[:12], cfg.Image)
	return id, nil
}

func (m *MockManager) Restart(ctx context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.containers[containerID]; ok {
		c.State = "running"
		log.Printf("[mock-container] restarted %s", containerID[:12])
		return nil
	}
	return fmt.Errorf("container %s not found", containerID)
}

func (m *MockManager) Stop(ctx context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.containers[containerID]; ok {
		c.State = "exited"
		return nil
	}
	return fmt.Errorf("container not found: %s", containerID)
}

func (m *MockManager) Remove(ctx context.Context, containerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.containers, containerID)
	return nil
}

func (m *MockManager) Status(ctx context.Context, containerID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if c, ok := m.containers[containerID]; ok {
		return c.State, nil
	}
	return "", fmt.Errorf("container not found: %s", containerID)
}

func (m *MockManager) Logs(ctx context.Context, containerID string, tail int) (string, error) {
	return "[mock] no real logs available\n", nil
}

func (m *MockManager) ListManaged(ctx context.Context) ([]ManagedContainer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []ManagedContainer
	for _, c := range m.containers {
		agentID := ""
		if c.Labels != nil {
			agentID = c.Labels["clawfleet.agent.id"]
		}
		result = append(result, ManagedContainer{
			ID:      c.ID,
			Name:    c.Name,
			Image:   c.Image,
			Status:  c.State,
			State:   c.State,
			AgentID: agentID,
		})
	}
	return result, nil
}

// ContainerManagerInterface is the interface both Manager and MockManager implement.
type ContainerManagerInterface interface {
	Create(ctx context.Context, name string, cfg *adapter.ContainerConfig) (string, error)
	Restart(ctx context.Context, containerID string) error
	Stop(ctx context.Context, containerID string) error
	Remove(ctx context.Context, containerID string) error
	Status(ctx context.Context, containerID string) (string, error)
	Logs(ctx context.Context, containerID string, tail int) (string, error)
	ListManaged(ctx context.Context) ([]ManagedContainer, error)
}

// Ensure both types implement the interface
var _ ContainerManagerInterface = (*Manager)(nil)
var _ ContainerManagerInterface = (*MockManager)(nil)
