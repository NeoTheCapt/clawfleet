package api

import (
	"log"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/agent/container"
	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// removeAgentOnNode enqueues a remove task for an agent's runtime container/service.
func (s *Server) removeAgentOnNode(agent *model.AgentInstance) {
	node, err := s.store.GetNode(agent.NodeID)
	if err != nil {
		log.Printf("[remove] node %s not found: %v", agent.NodeID, err)
		return
	}

	cid := agent.ContainerID
	if cid == "" {
		// Fallback to deterministic container name
		cid = container.ContainerName(agent.Name)
	}

	task := &model.Task{
		ID:     generateID("task"),
		NodeID: node.ID,
		Action: "remove_agent",
		Payload: map[string]interface{}{
			"agent_id":     agent.ID,
			"agent_name":   agent.Name,
			"container_id": cid,
		},
		Status:    model.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.CreateTask(task); err != nil {
		log.Printf("[remove] failed to create task: %v", err)
		return
	}
	log.Printf("[remove] task %s created to remove agent %s on node %s (container=%s)", task.ID, agent.Name, node.Name, cid)

	if node.Address != "" {
		s.pushTaskToNode(node, task)
	}
}
