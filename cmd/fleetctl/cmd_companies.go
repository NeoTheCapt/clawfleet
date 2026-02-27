package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func handleCompanies(args []string) {
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: fleetctl companies <subcommand>\n")
		fmt.Fprintf(os.Stderr, "Subcommands: list, create, show\n")
		os.Exit(1)
	}
	switch args[0] {
	case "list":
		resp, err := makeRequest("GET", "/api/companies", nil)
		if err != nil {
			log.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Request failed: status %d", resp.StatusCode)
		}
		var companies []map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&companies); err != nil {
			log.Fatalf("Parse response failed: %v", err)
		}
		fmt.Printf("%-20s %-30s %-10s %-20s\n", "ID", "NAME", "STATUS", "CREATED")
		fmt.Printf("%-20s %-30s %-10s %-20s\n", strings.Repeat("-", 20), strings.Repeat("-", 30), strings.Repeat("-", 10), strings.Repeat("-", 20))
		for _, c := range companies {
			fmt.Printf("%-20s %-30s %-10s %-20s\n",
				truncate(getString(c, "id"), 20),
				truncate(getString(c, "name"), 30),
				getString(c, "status"),
				formatTime(getString(c, "created_at")))
		}
	case "create":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: fleetctl companies create <name>\n")
			os.Exit(1)
		}
		body := map[string]string{"name": args[1]}
		resp, err := makeRequest("POST", "/api/companies", body)
		if err != nil {
			log.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			log.Fatalf("Request failed: status %d", resp.StatusCode)
		}
		var c map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
			log.Fatalf("Parse response failed: %v", err)
		}
		fmt.Printf("Company created: %s (%s)\n", getString(c, "name"), getString(c, "id"))
	case "show":
		if len(args) < 2 {
			fmt.Fprintf(os.Stderr, "Usage: fleetctl companies show <company-id>\n")
			os.Exit(1)
		}
		resp, err := makeRequest("GET", "/api/companies/"+args[1]+"/org-chart", nil)
		if err != nil {
			log.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Request failed: status %d", resp.StatusCode)
		}
		var chart map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&chart); err != nil {
			log.Fatalf("Parse response failed: %v", err)
		}
		data, _ := json.MarshalIndent(chart, "", "  ")
		fmt.Println(string(data))
	default:
		fmt.Fprintf(os.Stderr, "Unknown companies subcommand: %s\n", args[0])
		os.Exit(1)
	}
}
