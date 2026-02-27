package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/NeoTheCapt/clawfleet/internal/version"
)

func handleVersion(args []string) {
	var server string
	for i := 0; i < len(args)-1; i += 2 {
		if args[i] == "--server" {
			server = args[i+1]
		}
	}
	if server == "" {
		server = os.Getenv("FLEETCTL_SERVER")
	}

	fmt.Println("fleetctl:", version.String())

	if server == "" {
		return
	}

	req, _ := http.NewRequest("GET", server+"/api/version", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("server:  error:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("server:  http", resp.StatusCode)
		return
	}
	var info version.Info
	_ = json.NewDecoder(resp.Body).Decode(&info)
	fmt.Printf("server:  %s %s (%s) %s\n", info.Component, info.Version, short(info.Commit), info.BuildTime)
}

func short(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}
