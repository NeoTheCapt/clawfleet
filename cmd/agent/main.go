package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	agentsvc "github.com/NeoTheCapt/clawfleet/internal/agent"
	"github.com/NeoTheCapt/clawfleet/internal/agent/container"
	"github.com/NeoTheCapt/clawfleet/internal/model"
	"github.com/NeoTheCapt/clawfleet/internal/version"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

type Config struct {
	Server            string        `yaml:"server"`
	ControlPlane      string        `yaml:"control_plane"`
	Name              string        `yaml:"name"`
	Token             string        `yaml:"token"`
	RegisterToken     string        `yaml:"register_token"`
	Interval          time.Duration `yaml:"interval"`
	HeartbeatInterval int           `yaml:"heartbeat_interval"`
	DataDir           string        `yaml:"data_dir"`
}

// NodeState persists registration info across restarts.
type NodeState struct {
	NodeID   string `json:"node_id"`
	AgentKey string `json:"agent_key"`
}

func loadState(dataDir string) (*NodeState, error) {
	path := dataDir + "/state.json"
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s NodeState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func saveState(dataDir string, state *NodeState) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(dataDir+"/state.json", data, 0600)
}

func (c *Config) normalize() {
	// control_plane is an alias for server
	if c.Server == "" && c.ControlPlane != "" {
		c.Server = c.ControlPlane
	}
	// register_token is an alias for token
	if c.Token == "" && c.RegisterToken != "" {
		c.Token = c.RegisterToken
	}
	// heartbeat_interval (seconds) overrides interval
	if c.Interval == 0 && c.HeartbeatInterval > 0 {
		c.Interval = time.Duration(c.HeartbeatInterval) * time.Second
	}
}

func (c *Config) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}

func main() {
	server := flag.String("server", "http://localhost:8090", "Control Plane URL")
	name := flag.String("name", "", "Node name (defaults to hostname)")
	token := flag.String("token", "", "Registration token (required)")
	interval := flag.Duration("interval", 15*time.Second, "Heartbeat interval")
	noDocker := flag.Bool("no-docker", false, "Disable Docker integration (for testing)")
	agentImage := flag.String("agent-image", "openclaw:latest", "Docker image for agent containers")
	configFile := flag.String("config", "", "Config file path")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		return
	}

	// Load config from file if provided
	var config Config
	if *configFile != "" {
		if err := config.LoadFromFile(*configFile); err != nil {
			log.Fatalf("failed to load config: %v", err)
		}
		config.normalize()
	}

	// Command line args override config file
	if *server != "http://localhost:8090" {
		config.Server = *server
	}
	if *name != "" {
		config.Name = *name
	}
	if *token != "" {
		config.Token = *token
	}
	if *interval != 15*time.Second {
		config.Interval = *interval
	}

	// Use environment variables as fallback
	if config.Token == "" {
		config.Token = os.Getenv("CLAWFLEET_TOKEN")
	}
	// Token is only required if no saved state exists (checked later)
	if config.Server == "" {
		config.Server = "http://localhost:8090"
	}
	if config.Name == "" {
		hostname, _ := os.Hostname()
		config.Name = hostname
	}
	if config.Interval == 0 {
		config.Interval = 15 * time.Second
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("🛰️  Clawfleet Node Agent starting (name=%s) (%s)", config.Name, version.String())
	log.Printf("[agent] connecting to control plane: %s", config.Server)

	// Determine data directory for state persistence
	dataDir := config.DataDir
	if dataDir == "" {
		if *configFile != "" {
			dataDir = filepath.Dir(*configFile)
		} else {
			dataDir = "."
		}
	}

	// Try loading saved state (from previous registration)
	var nodeID, agentKey string
	if state, err := loadState(dataDir); err == nil && state.NodeID != "" && state.AgentKey != "" {
		nodeID = state.NodeID
		agentKey = state.AgentKey
		log.Printf("[agent] restored state: node=%s", nodeID)
	} else {
		// Register with control plane using token
		if config.Token == "" {
			log.Fatal("no saved state and no registration token provided")
		}
		res := collectResources()
		res.AgentVersion = version.Version
		res.AgentCommit = version.Commit
		res.AgentBuildTime = version.BuildTime
		var err error
		nodeID, agentKey, err = register(config.Server, config.Name, res, config.Token)
		if err != nil {
			log.Fatalf("failed to register: %v", err)
		}
		log.Printf("[agent] registered as node %s", nodeID)

		// Save state for next restart
		if err := saveState(dataDir, &NodeState{NodeID: nodeID, AgentKey: agentKey}); err != nil {
			log.Printf("[agent] warning: failed to save state: %v", err)
		} else {
			log.Printf("[agent] state saved to %s/state.json", dataDir)
		}
	}

	// Initialize container manager
	var cm container.ContainerManagerInterface
	if !*noDocker {
		realCM, err := container.NewManager()
		if err != nil {
			log.Printf("[agent] Docker not available: %v (falling back to mock)", err)
			cm = container.NewMockManager()
		} else {
			cm = realCM
		}
	} else {
		log.Println("[agent] Docker disabled, using mock manager")
		cm = container.NewMockManager()
	}

	// Create task executor (no HTTP listener in pull mode)
	executor := agentsvc.NewTaskExecutor(nodeID, cm, *agentImage)

	// Start heartbeat loop with task processing
	go heartbeatLoop(config.Server, nodeID, agentKey, config.Interval, executor)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Println("shutting down...")
}

func register(server, name string, res model.Resources, token string) (string, string, error) {
	body := map[string]interface{}{
		"name":      name,
		"token":     token,
		"resources": res,
		"labels":    map[string]string{"os": runtime.GOOS, "arch": runtime.GOARCH},
	}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", server+"/api/register", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("register request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", "", fmt.Errorf("register failed: status %d", resp.StatusCode)
	}

	var response struct {
		NodeID   string     `json:"node_id"`
		Node     model.Node `json:"node"`
		AgentKey string     `json:"agent_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", "", fmt.Errorf("decode response: %w", err)
	}
	if response.NodeID != "" {
		return response.NodeID, response.AgentKey, nil
	}
	return response.Node.ID, response.AgentKey, nil
}

func heartbeatLoop(server, nodeID, agentKey string, interval time.Duration, executor *agentsvc.TaskExecutor) {
	log.Printf("[heartbeat] starting loop (interval=%v, nodeID=%s)", interval, nodeID)

	// Send first heartbeat immediately
	doHeartbeat(server, nodeID, agentKey, executor)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		doHeartbeat(server, nodeID, agentKey, executor)
	}
}

func doHeartbeat(server, nodeID, agentKey string, executor *agentsvc.TaskExecutor) {
	log.Printf("[heartbeat] sending to %s", server)
	res := collectResources()
	res.AgentVersion = version.Version
	res.AgentCommit = version.Commit
	res.AgentBuildTime = version.BuildTime
	log.Printf("[heartbeat] resources collected: cpu=%d cores, key=%s...", res.CPUCores, agentKey[:8])
	body := map[string]interface{}{
		"node_id":   nodeID,
		"resources": res,
	}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", server+"/api/heartbeat", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Key", agentKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[heartbeat] error: %v", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[heartbeat] server returned status %d: %s", resp.StatusCode, string(body))
		resp.Body.Close()
		return
	}

	var heartbeatResponse struct {
		Status string       `json:"status"`
		Tasks  []model.Task `json:"tasks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResponse); err != nil {
		log.Printf("[heartbeat] decode error: %v", err)
		resp.Body.Close()
		return
	}
	resp.Body.Close()

	log.Printf("[heartbeat] ok, tasks=%d", len(heartbeatResponse.Tasks))

	// Process tasks
	for _, task := range heartbeatResponse.Tasks {
		go processTask(server, agentKey, executor, &task)
	}
}

func processTask(server, agentKey string, executor *agentsvc.TaskExecutor, task *model.Task) {
	log.Printf("[task] processing %s: %s", task.ID, task.Action)

	var result map[string]interface{}
	var status string

	switch task.Action {
	case "deploy", "deploy_agent":
		result, status = executor.ExecuteDeploy(task.Payload)
	case "stop", "stop_agent":
		result, status = executor.ExecuteStop(task.Payload)
	case "remove", "remove_agent":
		result, status = executor.ExecuteRemove(task.Payload)
	case "restart", "restart_agent":
		result, status = executor.ExecuteRestart(task.Payload)
	case "reload_persona":
		result, status = executor.ExecuteReloadPersona(task.Payload)
	case "debug_agent":
		result, status = executor.ExecuteDebugAgent(task.Payload)
	default:
		log.Printf("[task] unknown action: %s", task.Action)
		status = "failed"
		result = map[string]interface{}{"error": "unknown action"}
	}

	// Report completion back to control plane
	reportTaskCompletion(server, agentKey, task.ID, status, result)
}

func reportTaskCompletion(server, agentKey, taskID, status string, result map[string]interface{}) {
	body := map[string]interface{}{
		"task_id": taskID,
		"status":  status,
		"result":  result,
	}
	data, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", server+"/api/task/complete", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Key", agentKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[task-complete] error: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[task-complete] server returned status %d", resp.StatusCode)
		return
	}

	log.Printf("[task-complete] task %s completed with status %s", taskID, status)
}

func collectResources() model.Resources {
	res := model.Resources{
		CPUCores: runtime.NumCPU(),
	}
	if v, err := mem.VirtualMemory(); err == nil {
		res.MemoryMB = int(v.Total / 1024 / 1024)
		res.MemoryUsage = v.UsedPercent
	}
	if d, err := disk.Usage("/"); err == nil {
		res.DiskMB = int(d.Total / 1024 / 1024)
	}
	if pct, err := cpu.Percent(time.Second, false); err == nil && len(pct) > 0 {
		res.CPUUsage = pct[0]
	}
	return res
}
