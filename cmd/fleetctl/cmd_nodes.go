package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func handleNodes(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: fleetctl nodes <subcommand>\n")
		fmt.Fprintf(os.Stderr, "Subcommands: list, describe, remove, token\n")
		os.Exit(1)
	}

	switch args[0] {
	case "list":
		handleNodesList()
	case "describe":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: fleetctl nodes describe <node-id|name>\n")
			os.Exit(1)
		}
		handleNodeDescribe(args[1])
	case "remove":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: fleetctl nodes remove <node-id|name>\n")
			os.Exit(1)
		}
		handleNodeRemove(args[1])
	case "token":
		handleNodeToken(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown nodes subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func handleNodesList() {
	resp, err := makeRequest("GET", "/api/nodes", nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	var nodes []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&nodes); err != nil {
		log.Fatalf("Parse response failed: %v", err)
	}

	// Print table
	fmt.Printf("%-20s %-30s %-12s %-20s\n", "ID", "NAME", "STATUS", "LAST SEEN")
	fmt.Printf("%-20s %-30s %-12s %-20s\n", strings.Repeat("-", 20), strings.Repeat("-", 30), strings.Repeat("-", 12), strings.Repeat("-", 20))

	for _, node := range nodes {
		id := getString(node, "id")
		name := getString(node, "name")
		status := getString(node, "status")
		lastSeen := getString(node, "last_seen")

		// Format last seen
		if t, err := time.Parse(time.RFC3339, lastSeen); err == nil {
			lastSeen = t.Format("2006-01-02 15:04:05")
		}

		// Color status
		statusStr := formatStatus(status)

		fmt.Printf("%-20s %-30s %-12s %-20s\n",
			truncate(id, 20),
			truncate(name, 30),
			statusStr,
			lastSeen)
	}
}

func handleNodeDescribe(nodeID string) {
	resp, err := makeRequest("GET", "/api/nodes/"+nodeID, nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	var node map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		log.Fatalf("Parse response failed: %v", err)
	}

	fmt.Printf("Node Details:\n")
	fmt.Printf("  ID: %s\n", getString(node, "id"))
	fmt.Printf("  Name: %s\n", getString(node, "name"))
	fmt.Printf("  Status: %s\n", formatStatus(getString(node, "status")))
	fmt.Printf("  Created: %s\n", formatTime(getString(node, "created_at")))
	fmt.Printf("  Last Seen: %s\n", formatTime(getString(node, "last_seen")))

	if resources, ok := node["resources"].(map[string]interface{}); ok {
		fmt.Printf("  Resources:\n")
		fmt.Printf("    CPU Cores: %.0f\n", getFloat(resources, "cpu_cores"))
		fmt.Printf("    Memory: %.0f MB\n", getFloat(resources, "memory_mb"))
		fmt.Printf("    CPU Usage: %.1f%%\n", getFloat(resources, "cpu_usage"))
		fmt.Printf("    Memory Usage: %.1f%%\n", getFloat(resources, "memory_usage"))
	}

	// Get agents on this node
	agentResp, err := makeRequest("GET", "/api/nodes/"+nodeID+"/agents", nil)
	if err == nil && agentResp.StatusCode == http.StatusOK {
		var agents []map[string]interface{}
		if json.NewDecoder(agentResp.Body).Decode(&agents) == nil {
			fmt.Printf("\nAgents (%d):\n", len(agents))
			if len(agents) > 0 {
				fmt.Printf("  %-20s %-20s %-15s %-12s\n", "ID", "NAME", "TYPE", "STATUS")
				fmt.Printf("  %-20s %-20s %-15s %-12s\n", strings.Repeat("-", 20), strings.Repeat("-", 20), strings.Repeat("-", 15), strings.Repeat("-", 12))
				for _, agent := range agents {
					fmt.Printf("  %-20s %-20s %-15s %-12s\n",
						truncate(getString(agent, "id"), 20),
						truncate(getString(agent, "name"), 20),
						getString(agent, "agent_type"),
						formatStatus(getString(agent, "status")),
					)
				}
			}
		}
		agentResp.Body.Close()
	}
}

func handleNodeRemove(nodeID string) {
	resp, err := makeRequest("DELETE", "/api/nodes/"+nodeID, nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	fmt.Printf("Node %s removed\n", nodeID)
}

func handleNodeToken(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: fleetctl nodes token <subcommand>\n")
		fmt.Fprintf(os.Stderr, "Subcommands: create, list, revoke\n")
		os.Exit(1)
	}

	switch args[0] {
	case "create":
		handleTokenCreate(args[1:])
	case "list":
		handleTokenList()
	case "revoke":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: fleetctl nodes token revoke <token-id>\n")
			os.Exit(1)
		}
		handleTokenRevoke(args[1])
	default:
		fmt.Fprintf(os.Stderr, "Unknown token subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
