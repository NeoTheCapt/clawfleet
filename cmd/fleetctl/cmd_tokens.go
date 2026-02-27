package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func handleTokenCreate(args []string) {
	var name string
	for i := 0; i < len(args)-1; i += 2 {
		if args[i] == "--name" {
			name = args[i+1]
		}
	}

	body := map[string]string{}
	if name != "" {
		body["name"] = name
	}

	resp, err := makeRequest("POST", "/api/tokens", body)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Fatalf("Parse response failed: %v", err)
	}

	installCommand := getString(result, "install_command")
	tokenValue := getString(result, "token")
	if tokenValue == "" {
		log.Fatalf("Parse response failed: missing token")
	}

	fmt.Printf("Token created:\n")
	fmt.Printf("  ID: %s\n", getString(result, "id"))
	fmt.Printf("  Token: %s\n", tokenValue)
	fmt.Printf("  Name: %s\n", getString(result, "name"))
	fmt.Printf("\nInstall command:\n")
	fmt.Printf("%s\n", installCommand)
}

func handleTokenList() {
	resp, err := makeRequest("GET", "/api/tokens", nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	var tokens []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		log.Fatalf("Parse response failed: %v", err)
	}

	fmt.Printf("%-20s %-30s %-10s %-20s %-20s\n", "ID", "NAME", "USED", "USED BY", "CREATED")
	fmt.Printf("%-20s %-30s %-10s %-20s %-20s\n", strings.Repeat("-", 20), strings.Repeat("-", 30), strings.Repeat("-", 10), strings.Repeat("-", 20), strings.Repeat("-", 20))

	for _, token := range tokens {
		id := getString(token, "id")
		name := getString(token, "name")
		used := "No"
		if getBool(token, "used") {
			used = "Yes"
		}
		usedBy := getString(token, "used_by_node")
		created := formatTime(getString(token, "created_at"))

		fmt.Printf("%-20s %-30s %-10s %-20s %-20s\n",
			truncate(id, 20),
			truncate(name, 30),
			used,
			truncate(usedBy, 20),
			created)
	}
}

func handleTokenRevoke(tokenID string) {
	resp, err := makeRequest("DELETE", "/api/tokens/"+tokenID, nil)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Request failed: status %d", resp.StatusCode)
	}

	fmt.Printf("Token %s revoked\n", tokenID)
}
