package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func handleDeploy(args []string) {
	if len(args) < 5 {
		fmt.Fprintf(os.Stderr, "Usage: fleetctl deploy <name> --node <node> --type <type> [--role <role>] [--model <model>]\n")
		os.Exit(1)
	}

	name := args[0]
	var node, agentType, role, model string

	for i := 1; i < len(args)-1; i += 2 {
		switch args[i] {
		case "--node":
			node = args[i+1]
		case "--type":
			agentType = args[i+1]
		case "--role":
			role = args[i+1]
		case "--model":
			model = args[i+1]
		}
	}

	if node == "" || agentType == "" {
		fmt.Fprintf(os.Stderr, "Usage: fleetctl deploy <name> --node <node> --type <type> [--role <role>] [--model <model>]\n")
		os.Exit(1)
	}

	body := map[string]interface{}{
		"name":       name,
		"node_id":    node,
		"agent_type": agentType,
		"config": map[string]interface{}{
			"model": model,
			"resources": map[string]string{
				"cpu_limit":    "2",
				"memory_limit": "4Gi",
			},
		},
	}

	if role != "" {
		body["role"] = role
	}

	resp, err := makeRequest("POST", "/api/agents?deploy=true", body)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	var agent map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&agent); err != nil {
		log.Fatalf("Parse response failed: %v", err)
	}

	fmt.Printf("Agent deployment started:\n")
	fmt.Printf("  ID: %s\n", getString(agent, "id"))
	fmt.Printf("  Name: %s\n", getString(agent, "name"))
	fmt.Printf("  Type: %s\n", getString(agent, "agent_type"))
	fmt.Printf("  Node: %s\n", getString(agent, "node_id"))
	fmt.Printf("  Status: %s\n", formatStatus(getString(agent, "status")))
}
