package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/NeoTheCapt/clawfleet/internal/server/api"
	"github.com/NeoTheCapt/clawfleet/internal/server/auth"
	"github.com/NeoTheCapt/clawfleet/internal/server/comms"
	"github.com/NeoTheCapt/clawfleet/internal/server/scheduler"
	"github.com/NeoTheCapt/clawfleet/internal/store"
	"github.com/NeoTheCapt/clawfleet/internal/version"
)

func main() {
	addr := flag.String("addr", ":8090", "API listen address")
	dbPath := flag.String("db", "clawfleet.db", "SQLite database path")
	adminUser := flag.String("admin-user", "", "Admin username (required)")
	adminPass := flag.String("admin-pass", "", "Admin password (required)")
	agentKey := flag.String("agent-key", "", "Shared secret for node agent auth (required)")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.String())
		return
	}

	// Allow env overrides
	if *adminUser == "" {
		*adminUser = os.Getenv("CLAWFLEET_ADMIN_USER")
	}
	if *adminPass == "" {
		*adminPass = os.Getenv("CLAWFLEET_ADMIN_PASS")
	}
	if *agentKey == "" {
		*agentKey = os.Getenv("CLAWFLEET_AGENT_KEY")
	}

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("🚢 Clawfleet Control Plane starting... (%s)", version.String())

	// Initialize auth (will be overridden by DB if present)
	authCfg, err := auth.NewConfig(*adminUser, *adminPass, *agentKey)
	if err != nil {
		log.Fatalf("auth config error: %v", err)
	}
	log.Println("[auth] authentication enabled")

	// Initialize store
	s, err := store.New(*dbPath)
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}
	defer s.Close()
	log.Println("[store] SQLite initialized")

	// Load persisted admin credentials (if any), else persist env/flag credentials.
	if creds, err := s.GetAdminCredentials(); err != nil {
		log.Printf("[auth] warning: failed to load admin credentials from db: %v", err)
	} else if creds != nil {
		// Override runtime creds + secret
		_ = authCfg.UpdateCredentials(creds.Username, "")
		_ = authCfg.SetSecret(creds.JWTSecret)
		_ = authCfg.SetPassHash(creds.PassHash)
		log.Printf("[auth] loaded admin credentials from db (user=%s)", creds.Username)
	} else {
		// Persist initial creds to DB (password hash + jwt secret)
		u, ph, sec := authCfg.Snapshot()
		if err := s.UpsertAdminCredentials(u, ph, sec); err != nil {
			log.Printf("[auth] warning: failed to persist admin credentials to db: %v", err)
		}
	}

	// Initialize message broker
	broker := comms.NewBroker()

	// Initialize scheduler
	sched := scheduler.New(s, broker)
	if sched != nil {
		sched.StartKPICron()
		sched.StartIMOutboxWorker()
	}

	// Start API server
	srv := api.New(*addr, s, sched, broker, authCfg)

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		os.Exit(0)
	}()

	if err := srv.Start(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
