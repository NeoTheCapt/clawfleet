package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server string `yaml:"server"`
	Token  string `yaml:"token"`
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "version", "--version", "-v":
		handleVersion(args)
	case "login":
		handleLogin(args)
	case "nodes":
		handleNodes(args)
	case "agents":
		handleAgents(args)
	case "companies":
		handleCompanies(args)
	case "deploy":
		handleDeploy(args)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`fleetctl - Clawfleet CLI

USAGE:
  fleetctl <command> [args...]

COMMANDS:
  version [--server <url>]                            Show fleetctl + server version
  login --server <url> --user <user> --pass <pass>    Login and save token

  nodes list                                          List all nodes
  nodes describe <node-id|name>                       Node details + agents
  nodes remove <node-id|name>                         Remove node
  nodes token create [--name label]                   Generate install token
  nodes token list                                     List tokens
  nodes token revoke <token-id>                       Revoke token

  deploy <name> --node <node> --type <type> [options] Deploy agent to node
    --role <role>        Agent role (optional)
    --model <model>      AI model (optional)

  agents list [--node <node>] [--cluster <cluster>]   List agents
  agents describe <agent-id|name>                     Agent details
  agents stop <agent-id|name>                         Stop agent
  agents remove <agent-id|name>                       Remove agent
  agents logs <agent-id|name>                         Agent logs

  companies list                                       List companies
  companies create <name>                              Create company
  companies show <company-id>                          Show company org chart

ENVIRONMENT:
  FLEETCTL_SERVER     Server URL (alternative to --server)
  FLEETCTL_TOKEN      JWT token (alternative to login)

CONFIG FILE:
  ~/.config/fleetctl/config.yaml

`)
}

func handleLogin(args []string) {
	var server, user, pass string

	for i := 0; i < len(args)-1; i += 2 {
		switch args[i] {
		case "--server":
			server = args[i+1]
		case "--user":
			user = args[i+1]
		case "--pass":
			pass = args[i+1]
		}
	}

	if server == "" || user == "" || pass == "" {
		fmt.Fprintf(os.Stderr, "Usage: fleetctl login --server <url> --user <user> --pass <pass>\n")
		os.Exit(1)
	}

	// Make login request
	body := map[string]string{
		"username": user,
		"password": pass,
	}
	data, _ := json.Marshal(body)

	resp, err := http.Post(server+"/api/login", "application/json", bytes.NewReader(data))
	if err != nil {
		log.Fatalf("Login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Login failed: status %d", resp.StatusCode)
	}

	var loginResp struct {
		Token string `json:"token"`
		User  string `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		log.Fatalf("Parse login response failed: %v", err)
	}

	// Save config
	config := Config{
		Server: server,
		Token:  loginResp.Token,
	}

	configDir := filepath.Join(os.Getenv("HOME"), ".config", "fleetctl")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		log.Fatalf("Create config dir failed: %v", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")
	configData, _ := yaml.Marshal(config)
	if err := os.WriteFile(configFile, configData, 0600); err != nil {
		log.Fatalf("Save config failed: %v", err)
	}

	fmt.Printf("Logged in as %s\n", loginResp.User)
	fmt.Printf("Token saved to %s\n", configFile)
}

func loadConfig() (*Config, error) {
	config := &Config{}

	// Try environment variables first
	if server := os.Getenv("FLEETCTL_SERVER"); server != "" {
		config.Server = server
	}
	if token := os.Getenv("FLEETCTL_TOKEN"); token != "" {
		config.Token = token
	}

	// Try config file
	configFile := filepath.Join(os.Getenv("HOME"), ".config", "fleetctl", "config.yaml")
	if data, err := os.ReadFile(configFile); err == nil {
		var fileConfig Config
		if yaml.Unmarshal(data, &fileConfig) == nil {
			if config.Server == "" {
				config.Server = fileConfig.Server
			}
			if config.Token == "" {
				config.Token = fileConfig.Token
			}
		}
	}

	if config.Server == "" || config.Token == "" {
		return nil, fmt.Errorf("server and token required - run 'fleetctl login' or set FLEETCTL_SERVER/FLEETCTL_TOKEN")
	}

	return config, nil
}

func makeRequest(method, path string, body interface{}) (*http.Response, error) {
	config, err := loadConfig()
	if err != nil {
		return nil, err
	}

	var reqBody *bytes.Buffer
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(data)
	}

	var req *http.Request
	if reqBody != nil {
		req, err = http.NewRequest(method, config.Server+path, reqBody)
	} else {
		req, err = http.NewRequest(method, config.Server+path, nil)
	}
	if err != nil {
		return nil, err
	}

	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", "Bearer "+config.Token)

	return http.DefaultClient.Do(req)
}

// Utility functions

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key]; ok {
		if f, ok := v.(float64); ok {
			return f
		}
	}
	return 0
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func formatStatus(status string) string {
	switch status {
	case "online", "running":
		return "🟢 " + status
	case "offline", "error", "failed":
		return "🔴 " + status
	case "creating", "provisioning":
		return "🟡 " + status
	default:
		return status
	}
}

func formatTime(timeStr string) string {
	if timeStr == "" {
		return ""
	}
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t.Format("2006-01-02 15:04")
	}
	return timeStr
}
