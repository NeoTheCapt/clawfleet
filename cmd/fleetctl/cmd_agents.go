package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func handleAgents(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: fleetctl agents <subcommand>\n")
		fmt.Fprintf(os.Stderr, "Subcommands: list, describe, stop, remove, logs\n")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		handleAgentsList(args[1:])
	case "describe":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: fleetctl agents describe <agent-id|name>\n")
			os.Exit(1)
		}
		handleAgentDescribe(args[1])
	case "stop":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: fleetctl agents stop <agent-id|name>\n")
			os.Exit(1)
		}
		handleAgentStop(args[1])
	case "remove":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: fleetctl agents remove <agent-id|name>\n")
			os.Exit(1)
		}
		handleAgentRemove(args[1])
	case "logs":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: fleetctl agents logs <agent-id|name>\n")
			os.Exit(1)
		}
		handleAgentLogs(args[1])
	default:
		fmt.Fprintf(os.Stderr, "Unknown agents subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func handleAgentsList(args []string) {
	query := ""
	for i := 0; i < len(args)-1; i += 2 {
		switch args[i] {
		case "--node":
			query += "?node_id=" + args[i+1]
		case "--company":
			query += "?company_id=" + args[i+1]
		}
	}

	resp, err := makeRequest("GET", "/api/agents"+query, nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	var agents []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&agents); err != nil {
		log.Fatalf("Parse response failed: %v", err)
	}

	fmt.Printf("%-20s %-25s %-20s %-15s %-12s %-20s\n", "ID", "NAME", "NODE", "TYPE", "STATUS", "CREATED")
	fmt.Printf("%-20s %-25s %-20s %-15s %-12s %-20s\n", strings.Repeat("-", 20), strings.Repeat("-", 25), strings.Repeat("-", 20), strings.Repeat("-", 15), strings.Repeat("-", 12), strings.Repeat("-", 20))

	for _, agent := range agents {
		id := getString(agent, "id")
		name := getString(agent, "name")
		nodeID := getString(agent, "node_id")
		agentType := getString(agent, "agent_type")
		status := getString(agent, "status")
		created := formatTime(getString(agent, "created_at"))

		fmt.Printf("%-20s %-25s %-20s %-15s %-12s %-20s\n",
			truncate(id, 20),
			truncate(name, 25),
			truncate(nodeID, 20),
			agentType,
			formatStatus(status),
			created)
	}
}

func handleAgentDescribe(agentID string) {
	resp, err := makeRequest("GET", "/api/agents/"+agentID, nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	var agent map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&agent); err != nil {
		log.Fatalf("Parse response failed: %v", err)
	}

	fmt.Printf("Agent Details:\n")
	fmt.Printf("  ID: %s\n", getString(agent, "id"))
	fmt.Printf("  Name: %s\n", getString(agent, "name"))
	fmt.Printf("  Type: %s\n", getString(agent, "agent_type"))
	fmt.Printf("  Status: %s\n", formatStatus(getString(agent, "status")))
	fmt.Printf("  Node: %s\n", getString(agent, "node_id"))
	fmt.Printf("  Company: %s\n", getString(agent, "company_id"))
	fmt.Printf("  Role: %s\n", getString(agent, "role"))
	fmt.Printf("  Container: %s\n", getString(agent, "container_id"))
	fmt.Printf("  Created: %s\n", formatTime(getString(agent, "created_at")))
	fmt.Printf("  Updated: %s\n", formatTime(getString(agent, "updated_at")))

	if config, ok := agent["config"].(map[string]interface{}); ok {
		fmt.Printf("  Config:\n")
		fmt.Printf("    Model: %s\n", getString(config, "model"))
		fmt.Printf("    System Prompt: %s\n", truncate(getString(config, "system_prompt"), 60))
	}
}

func handleAgentStop(agentID string) {
	// For now, just call DELETE since we don't have a separate stop endpoint
	// In a full implementation, this would stop the container but keep the record
	resp, err := makeRequest("DELETE", "/api/agents/"+agentID, nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	fmt.Printf("Agent %s stopped\n", agentID)
}

func handleAgentRemove(agentID string) {
	resp, err := makeRequest("DELETE", "/api/agents/"+agentID, nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	fmt.Printf("Agent %s removed\n", agentID)
}

func handleAgentLogs(agentID string) {
	// This would need to be implemented in the API
	fmt.Printf("Agent logs for %s (not implemented yet)\n", agentID)
}
