package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

const maxTaskResultStringBytes = 48 * 1024

var (
	apiJSONSecretPattern   = regexp.MustCompile(`(?i)"(token|api[_-]?key|im[_-]?key|secret|password)"\s*:\s*"([^"]*)"`)
	apiInlineSecretPattern = regexp.MustCompile(`(?i)\b(token|api[_-]?key|im[_-]?key|secret|password)\b(\s*[:=]\s*)(\S+)`)
	apiTelegramURLPattern  = regexp.MustCompile(`https?://api\.telegram\.org/bot[^/\s]+`)
)

// --- Node Agent Endpoints (registration, heartbeat, task completion) ---

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Token     string            `json:"token"`
		Name      string            `json:"name"`
		Address   string            `json:"address"`
		Labels    map[string]string `json:"labels,omitempty"`
		Resources model.Resources   `json:"resources,omitempty"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Token == "" {
		writeBadRequest(w, "token required")
		return
	}

	// Validate token
	tok, err := s.store.GetToken(req.Token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}
	if tok.Used {
		writeError(w, http.StatusConflict, "token already used")
		return
	}

	// Generate agent key for this node
	agentKey := generateToken()

	node := &model.Node{
		ID:        generateID("node"),
		Name:      req.Name,
		Address:   req.Address,
		Status:    model.NodeStatusOnline,
		Labels:    req.Labels,
		Resources: req.Resources,
		AgentKey:  agentKey,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	if err := s.store.CreateNode(node); err != nil {
		writeServerError(w, err)
		return
	}

	// Mark token as used
	if err := s.store.MarkTokenUsed(req.Token, node.ID); err != nil {
		if delErr := s.store.DeleteNode(node.ID); delErr != nil {
			log.Printf("[api] failed to roll back node %s after token mark failure: %v", node.ID, delErr)
		}
		writeServerError(w, err)
		return
	}

	log.Printf("[api] node registered: %s (%s) from token %s", node.Name, node.ID, tok.Name)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"node_id":   node.ID,
		"node":      node,
		"agent_key": agentKey,
	})
}

func (s *Server) handleHeartbeat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract node from agent key
	agentKey := extractAgentKey(r)
	node, err := s.store.FindNodeByAgentKey(agentKey)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid agent key")
		return
	}

	var req struct {
		Resources model.Resources `json:"resources,omitempty"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}

	// Update heartbeat (includes node agent version info embedded in Resources)
	if err := s.store.UpdateNodeHeartbeat(node.ID, req.Resources); err != nil {
		writeServerError(w, err)
		return
	}

	// Check for pending tasks
	tasks, _ := s.store.ListPendingTasks(node.ID)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"tasks":  tasks,
	})
}

func (s *Server) handleTaskComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		TaskID string                 `json:"task_id"`
		Status model.TaskStatus       `json:"status"`
		Result map[string]interface{} `json:"result,omitempty"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Result = sanitizeTaskResult(req.Result)

	if err := s.store.UpdateTaskStatus(req.TaskID, req.Status, req.Result); err != nil {
		writeServerError(w, err)
		return
	}

	// Use task metadata to decide what to do.
	task, _ := s.store.GetTask(req.TaskID)

	// If task was agent deployment, update agent status
	if req.Result != nil {
		if agentID, ok := req.Result["agent_id"].(string); ok {
			if req.Status == model.TaskStatusDone {
				containerID, _ := req.Result["container_id"].(string)
				_ = s.store.UpdateAgentStatus(agentID, model.AgentStatusRunning, containerID)
			} else if req.Status == model.TaskStatusFailed {
				// Don't downgrade a running agent to error (e.g. duplicate deploy task)
				if existing, err := s.store.GetAgentInstance(agentID); err == nil && existing.Status == model.AgentStatusRunning {
					log.Printf("[task-complete] ignoring failed task for already-running agent %s", agentID)
				} else {
					_ = s.store.UpdateAgentStatus(agentID, model.AgentStatusError, "")
				}
			}
		}
	}

	// If task was stop/remove, and agent is in deleting status, purge it.
	if task != nil && req.Status == model.TaskStatusDone {
		switch task.Action {
		case "remove_agent", "stop_agent":
			agentID, _ := task.Payload["agent_id"].(string)
			if agentID == "" {
				if req.Result != nil {
					agentID, _ = req.Result["agent_id"].(string)
				}
			}
			if agentID != "" {
				if existing, err := s.store.GetAgentInstance(agentID); err == nil && existing.Status == model.AgentStatusDeleting {
					if err := s.store.DeleteAgentInstance(agentID); err != nil {
						log.Printf("[task-complete] warning: failed to delete agent instance %s: %v", agentID, err)
					} else {
						log.Printf("[task-complete] deleted agent instance %s after %s", agentID, task.Action)
					}
				}
			}
		}
	}

	writeStatus(w, http.StatusOK, "ok")
}

// --- Install Script & Binary Download ---

func (s *Server) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")

	// Allow passing token/name via query for one-click UX.
	qToken := r.URL.Query().Get("token")
	qName := r.URL.Query().Get("name")

	host := r.Host
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	serverURL := scheme + "://" + host

	script := `#!/bin/bash
set -e

echo "🚢 Clawfleet Node Agent Installer"
echo "================================="

SERVER="` + serverURL + `"
TOKEN="` + qToken + `"
NAME="` + qName + `"

if [ -z "$TOKEN" ]; then
  echo "ERROR: missing token"
  echo "Usage: curl -fsSL $SERVER/api/install.sh?token=XXX\&name=aussie | sudo bash"
  exit 1
fi

if [ -z "$NAME" ]; then
  NAME=$(hostname)
fi

# Detect arch
ARCH=$(uname -m)
case $ARCH in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

OS=$(uname -s | tr '[:upper:]' '[:lower:]')

DATA_DIR="/data/clawfleet-node"
BIN="$DATA_DIR/clawfleet-node"
CFG="$DATA_DIR/config.yaml"

echo "Installing to $DATA_DIR ..."
mkdir -p "$DATA_DIR"

echo "Downloading clawfleet-node ($OS/$ARCH)..."
curl -fsSL "$SERVER/dl/fleet-node?os=$OS&arch=$ARCH" -o "$BIN"
chmod +x "$BIN"

cat > "$CFG" <<EOF
control_plane: $SERVER
name: $NAME
register_token: $TOKEN
heartbeat_interval: 30
data_dir: $DATA_DIR
EOF

cat > /etc/systemd/system/clawfleet-node.service <<'EOF'
[Unit]
Description=Clawfleet Node Agent
After=network.target docker.service
Wants=docker.service

[Service]
Type=simple
WorkingDirectory=/data/clawfleet-node
ExecStart=/data/clawfleet-node/clawfleet-node --config /data/clawfleet-node/config.yaml
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable clawfleet-node
systemctl restart clawfleet-node

sleep 2
$BIN -version || true
systemctl status clawfleet-node --no-pager -l | head -n 20

echo "✅ Done"
`

	if _, err := w.Write([]byte(script)); err != nil {
		log.Printf("[api] failed to write install script response: %v", err)
	}
}

func (s *Server) handleBinaryDownload(w http.ResponseWriter, r *http.Request) {
	reqOS := r.URL.Query().Get("os")
	reqArch := r.URL.Query().Get("arch")
	if reqOS == "" {
		reqOS = "linux"
	}
	if reqArch == "" {
		reqArch = runtime.GOARCH
	}

	filename := fmt.Sprintf("clawfleet-node-%s-%s", reqOS, reqArch)

	candidates := []string{
		filepath.Join("dist", "bin", filename),
		filepath.Join("dist", "bin", "clawfleet-node"),
		filepath.Join("/data", "clawfleet", "bin", "clawfleet-node"),
		filepath.Join("dist", "bin", "clawfleet-agent"), // legacy fallback
		filepath.Join("bin", "clawfleet-agent"),         // legacy fallback
	}

	path := ""
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			path = candidate
			break
		}
	}

	if path == "" {
		writeNotFound(w, "binary not found")
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=fleet-node")
	http.ServeFile(w, r, path)
}

// --- Stop Agent on Node ---

func (s *Server) restartAgentOnNode(agent *model.AgentInstance) {
	node, err := s.store.GetNode(agent.NodeID)
	if err != nil {
		log.Printf("[restart] node %s not found: %v", agent.NodeID, err)
		return
	}

	task := &model.Task{
		ID:     generateID("task"),
		NodeID: node.ID,
		Action: "restart_agent",
		Payload: map[string]interface{}{
			"agent_id":     agent.ID,
			"agent_name":   agent.Name,
			"container_id": agent.ContainerID,
		},
		Status:    model.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.CreateTask(task); err != nil {
		log.Printf("[restart] failed to create task: %v", err)
		return
	}

	log.Printf("[restart] task %s created for agent %s on node %s", task.ID, agent.Name, node.Name)

	if node.Address != "" {
		s.pushTaskToNode(node, task)
	}
}

func (s *Server) stopAgentOnNode(agent *model.AgentInstance) {
	node, err := s.store.GetNode(agent.NodeID)
	if err != nil {
		log.Printf("[stop] node %s not found: %v", agent.NodeID, err)
		return
	}

	task := &model.Task{
		ID:     generateID("task"),
		NodeID: node.ID,
		Action: "stop_agent",
		Payload: map[string]interface{}{
			"agent_id":     agent.ID,
			"agent_name":   agent.Name,
			"container_id": agent.ContainerID,
		},
		Status:    model.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.CreateTask(task); err != nil {
		log.Printf("[stop] failed to create task: %v", err)
		return
	}

	log.Printf("[stop] task %s created to stop agent %s on node %s", task.ID, agent.Name, node.Name)

	if node.Address != "" {
		s.pushTaskToNode(node, task)
	}

	// Wait a bit for stop to complete before redeploying
	time.Sleep(5 * time.Second)
}

// --- Deploy to Node ---

func (s *Server) deployToNode(agent *model.AgentInstance) {
	node, err := s.store.GetNode(agent.NodeID)
	if err != nil {
		log.Printf("[deploy] node %s not found: %v", agent.NodeID, err)
		if uErr := s.store.UpdateAgentStatus(agent.ID, model.AgentStatusError, ""); uErr != nil {
			log.Printf("[deploy] failed to set agent %s status to error: %v", agent.ID, uErr)
		}
		return
	}

	task := &model.Task{
		ID:     generateID("task"),
		NodeID: node.ID,
		Action: "deploy_agent",
		Payload: map[string]interface{}{
			"agent_id":    agent.ID,
			"agent_name":  agent.Name,
			"agent_type":  string(agent.AgentType),
			"deploy_mode": agent.DeployMode,
			"config":      agent.Config,
		},
		Status:    model.TaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.CreateTask(task); err != nil {
		log.Printf("[deploy] failed to create task: %v", err)
		if uErr := s.store.UpdateAgentStatus(agent.ID, model.AgentStatusError, ""); uErr != nil {
			log.Printf("[deploy] failed to set agent %s status to error: %v", agent.ID, uErr)
		}
		return
	}

	log.Printf("[deploy] task %s created for agent %s on node %s", task.ID, agent.Name, node.Name)

	// Also try direct HTTP push to node if address available
	if node.Address != "" {
		s.pushTaskToNode(node, task)
	}
}

func (s *Server) pushTaskToNode(node *model.Node, task *model.Task) {
	url := strings.TrimRight(node.Address, "/") + "/task"
	body, _ := json.Marshal(task)
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		log.Printf("[deploy] failed to create request to node %s: %v", node.Name, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Key", node.AgentKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[deploy] failed to push task to node %s: %v", node.Name, err)
		return
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		log.Printf("[deploy] failed to drain node response body for %s: %v", node.Name, err)
	}
	log.Printf("[deploy] pushed task to node %s (status: %d)", node.Name, resp.StatusCode)
}

// extractAgentKey gets the agent key from request header.
func extractAgentKey(r *http.Request) string {
	key := r.Header.Get("X-Agent-Key")
	if key == "" {
		// Try Authorization: Bearer
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			key = strings.TrimPrefix(auth, "Bearer ")
		}
	}
	return key
}

func sanitizeTaskResult(v map[string]interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	out := make(map[string]interface{}, len(v))
	for k, val := range v {
		out[k] = sanitizeTaskValue(val)
	}
	return out
}

func sanitizeTaskValue(v interface{}) interface{} {
	switch t := v.(type) {
	case map[string]interface{}:
		n := make(map[string]interface{}, len(t))
		for k, vv := range t {
			n[k] = sanitizeTaskValue(vv)
		}
		return n
	case []interface{}:
		n := make([]interface{}, 0, len(t))
		for _, vv := range t {
			n = append(n, sanitizeTaskValue(vv))
		}
		return n
	case string:
		s := strings.ReplaceAll(t, "\r", "\n")
		s = apiTelegramURLPattern.ReplaceAllString(s, "https://api.telegram.org/bot[REDACTED]")
		s = apiJSONSecretPattern.ReplaceAllString(s, `"$1":"[REDACTED]"`)
		s = apiInlineSecretPattern.ReplaceAllString(s, `$1$2[REDACTED]`)
		if len(s) > maxTaskResultStringBytes {
			s = s[:maxTaskResultStringBytes] + "\n...[truncated]..."
		}
		return s
	default:
		return v
	}
}
