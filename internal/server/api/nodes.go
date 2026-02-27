package api

import (
	"log"
	"net/http"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// --- Nodes ---

func (s *Server) handleNodes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		nodes, err := s.store.ListNodes()
		if err != nil {
			writeServerError(w, err)
			return
		}
		if nodes == nil {
			nodes = []*model.Node{}
		}
		writeJSON(w, http.StatusOK, nodes)
	case http.MethodPost:
		var req struct {
			Name    string            `json:"name"`
			Address string            `json:"address"`
			Labels  map[string]string `json:"labels,omitempty"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.Name == "" || req.Address == "" {
			writeBadRequest(w, "name and address required")
			return
		}
		node := &model.Node{
			ID:        generateID("node"),
			Name:      req.Name,
			Address:   req.Address,
			Status:    model.NodeStatusOffline,
			Labels:    req.Labels,
			Resources: model.Resources{},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			LastSeen:  time.Now(),
		}
		if err := s.store.CreateNode(node); err != nil {
			writeServerError(w, err)
			return
		}
		log.Printf("[api] node manually added: %s (%s)", node.Name, node.ID)
		writeJSON(w, http.StatusCreated, node)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleNodeByID(w http.ResponseWriter, r *http.Request) {
	// Check for sub-routes like {id}/deploy, {id}/agents
	id, subRoute := splitSubpath(r.URL.Path, "/api/nodes/")

	if id == "" {
		writeBadRequest(w, "missing node id")
		return
	}

	// Sub-routes
	if subRoute == "deploy" {
		s.handleNodeDeploy(w, r, id)
		return
	}
	if subRoute == "agents" {
		s.handleNodeAgents(w, r, id)
		return
	}

	switch r.Method {
	case http.MethodGet:
		node, err := s.store.GetNode(id)
		if err != nil {
			writeNotFound(w, "node not found")
			return
		}
		writeJSON(w, http.StatusOK, node)
	case http.MethodPut:
		node, err := s.store.GetNode(id)
		if err != nil {
			writeNotFound(w, "node not found")
			return
		}
		var req struct {
			Name    *string           `json:"name,omitempty"`
			Address *string           `json:"address,omitempty"`
			Labels  map[string]string `json:"labels,omitempty"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.Name != nil {
			node.Name = *req.Name
		}
		if req.Address != nil {
			node.Address = *req.Address
		}
		if req.Labels != nil {
			node.Labels = req.Labels
		}
		if err := s.store.UpdateNode(node); err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, node)
	case http.MethodDelete:
		if err := s.store.DeleteNode(id); err != nil {
			writeServerError(w, err)
			return
		}
		writeStatus(w, http.StatusOK, "deleted")
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// handleNodeAgents returns agents on a specific node.
func (s *Server) handleNodeAgents(w http.ResponseWriter, r *http.Request, nodeID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	agents, err := s.store.ListAgentsByNode(nodeID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	if agents == nil {
		agents = []*model.AgentInstance{}
	}
	writeJSON(w, http.StatusOK, sanitizeAgentsForResponse(agents))
}

// handleNodeDeploy deploys an agent to a specific node.
func (s *Server) handleNodeDeploy(w http.ResponseWriter, r *http.Request, nodeID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	node, err := s.store.GetNode(nodeID)
	if err != nil {
		writeNotFound(w, "node not found")
		return
	}

	var req CreateAgentRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Name == "" || req.AgentType == "" {
		writeBadRequest(w, "name and agent_type required")
		return
	}

	agent := newAgentInstanceFromRequest(req, node.ID)
	if err := s.store.CreateAgentInstance(agent); err != nil {
		writeServerError(w, err)
		return
	}

	log.Printf("[api] deploying agent %s (%s) to node %s", agent.Name, agent.AgentType, node.Name)

	// Deploy to node async
	go s.deployToNode(agent)

	writeJSON(w, http.StatusCreated, sanitizeAgentForResponse(agent))
}
