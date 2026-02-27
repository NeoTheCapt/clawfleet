package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/util"

	"github.com/NeoTheCapt/clawfleet/internal/persona"
	"github.com/NeoTheCapt/clawfleet/internal/server/auth"
	"github.com/NeoTheCapt/clawfleet/internal/server/comms"
	"github.com/NeoTheCapt/clawfleet/internal/server/scheduler"
	"github.com/NeoTheCapt/clawfleet/internal/store"
	"github.com/NeoTheCapt/clawfleet/internal/sync"
)

// Server is the HTTP API server for the control plane.
type Server struct {
	store      *store.Store
	scheduler  *scheduler.Scheduler
	broker     *comms.Broker
	auth       *auth.Config
	mux        *http.ServeMux
	addr       string
	personaGen *persona.Generator
	syncEngine *sync.PersonaSyncEngine
}

// New creates a new API server.
func New(addr string, s *store.Store, sched *scheduler.Scheduler, broker *comms.Broker, authCfg *auth.Config) *Server {
	// 初始化人设生成器
	personaGen := persona.NewGenerator(s)

	// 初始化同步引擎
	syncEngine := sync.NewPersonaSyncEngine(s, 3)

	srv := &Server{
		store:      s,
		scheduler:  sched,
		broker:     broker,
		auth:       authCfg,
		mux:        http.NewServeMux(),
		addr:       addr,
		personaGen: personaGen,
		syncEngine: syncEngine,
	}
	// Wire per-node key validation
	authCfg.SetNodeKeyValidator(func(key string) bool {
		_, err := s.FindNodeByAgentKey(key)
		return err == nil
	})

	srv.routes()
	return srv
}

// Start begins serving HTTP.
func (s *Server) Start() error {
	log.Printf("[api] listening on %s", s.addr)
	return http.ListenAndServe(s.addr, s.mux)
}

func (s *Server) routes() {
	jwt := s.auth.RequireJWT
	agentAuth := s.auth.RequireAgentKey

	// Static files (Web UI) — SPA mode, serve from web/dist/
	s.mux.HandleFunc("/", s.handleSPA)

	// Public endpoints
	s.mux.HandleFunc("/api/health", s.handleHealth)
	s.mux.HandleFunc("/api/version", s.handleVersion)
	s.mux.HandleFunc("/api/login", s.handleLogin)
	s.mux.HandleFunc("/api/im/conversations", s.requireIMAuth(s.handleIMConversations))
	s.mux.HandleFunc("/api/im/conversations/", s.requireIMAuth(s.handleIMConversationByID))
	// IM outbox (agent-only)
	s.mux.HandleFunc("/api/im/outbox", s.requireIMAuth(s.handleIMOutbox))
	s.mux.HandleFunc("/api/im/outbox/", s.requireIMAuth(s.handleIMOutbox))
	s.mux.HandleFunc("/api/im/sync-org", s.requireIMAuth(s.handleIMSyncOrg))
	s.mux.HandleFunc("/api/im/bootstrap", s.requireIMAuth(s.handleIMBootstrap))

	// JWT-protected (Web UI)
	s.mux.HandleFunc("/api/nodes", jwt(s.handleNodes))
	s.mux.HandleFunc("/api/nodes/", jwt(s.handleNodeByID))
	s.mux.HandleFunc("/api/agents", jwt(s.handleAgents))
	s.mux.HandleFunc("/api/agents/", jwt(s.handleAgentByID))
	s.mux.HandleFunc("/api/tasks/", jwt(s.handleTaskByID))
	// Company Management
	s.mux.HandleFunc("/api/companies", jwt(s.handleCompanies))
	s.mux.HandleFunc("/api/companies/", jwt(s.handleCompanyByID))
	s.mux.HandleFunc("/api/kpi/trigger-daily", jwt(s.handleKPITriggerDaily))
	s.mux.HandleFunc("/api/kpi/reviews/", jwt(s.handleKPIReviewSubmit))
	s.mux.HandleFunc("/api/tokens", jwt(s.handleTokens))
	s.mux.HandleFunc("/api/tokens/", jwt(s.handleTokenByID))
	s.mux.HandleFunc("/api/work", jwt(s.handleWorkList))
	s.mux.HandleFunc("/api/work/plan", jwt(s.handleWorkPlan))
	s.mux.HandleFunc("/api/work/approve", jwt(s.handleWorkApprove))
	s.mux.HandleFunc("/api/work/", s.handleWork)
	// Persona Templates
	s.mux.HandleFunc("/api/persona-templates", jwt(s.handlePersonaTemplates))
	s.mux.HandleFunc("/api/persona-templates/", jwt(s.handlePersonaTemplateByID))
	// Persona Generation and Management
	s.mux.HandleFunc("/api/persona/generate", jwt(s.handlePersonaGenerate))
	// Persona Synchronization
	s.mux.HandleFunc("/api/positions/", jwt(s.handlePositionWithSubpath)) // This will handle /positions/{id}/sync-persona

	// Admin settings
	s.mux.HandleFunc("/api/admin/credentials", jwt(s.handleAdminCredentials))

	// Public endpoints for install
	s.mux.HandleFunc("/api/install.sh", s.handleInstallScript)
	s.mux.HandleFunc("/dl/fleet-node", s.handleBinaryDownload)

	// Agent-key protected (node agents)
	s.mux.HandleFunc("/api/register", s.handleRegister) // No longer requires agent-key, uses tokens
	s.mux.HandleFunc("/api/heartbeat", agentAuth(s.handleHeartbeat))
	s.mux.HandleFunc("/api/task/complete", agentAuth(s.handleTaskComplete))
}

// --- Login ---

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !s.auth.CheckPassword(req.Username, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := s.auth.GenerateToken(req.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token generation failed")
		return
	}
	// Set cookie for browser
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})
	writeJSON(w, http.StatusOK, map[string]string{
		"token": token,
		"user":  req.Username,
	})
}

// --- Health ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "clawfleet-server",
		"time":    time.Now().Format(time.RFC3339),
	})
}

// --- Helpers ---

// handleSPA serves the Vue SPA from web/dist/.
// For non-API GET requests where the file doesn't exist, it falls back to index.html (history mode).
func (s *Server) handleSPA(w http.ResponseWriter, r *http.Request) {
	distDir := "web/dist"
	path := r.URL.Path

	// Never let SPA fallback swallow unknown API routes.
	if strings.HasPrefix(path, "/api/") {
		writeNotFound(w, "not found")
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if path == "/" {
		path = "/index.html"
	}

	// Try to serve the file directly
	filePath := distDir + path
	if f, err := http.Dir(distDir).Open(path); err == nil {
		_ = f.Close()
		if path == "/index.html" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
		}
		http.ServeFile(w, r, filePath)
		return
	}

	// Fallback to index.html for SPA routing
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	http.ServeFile(w, r, distDir+"/index.html")
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("[api] failed to encode JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeBadRequest(w http.ResponseWriter, msg string) { writeError(w, http.StatusBadRequest, msg) }
func writeNotFound(w http.ResponseWriter, msg string)   { writeError(w, http.StatusNotFound, msg) }
func writeServerError(w http.ResponseWriter, err error) {
	log.Printf("[api] internal server error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func writeStatus(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"status": msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeBadRequest(w, "invalid json")
		return false
	}
	return true
}

func generateToken() string           { return util.GenerateToken() }
func generateID(prefix string) string { return util.GenerateID(prefix) }
func randomSuffix() string            { return util.RandomSuffix() }
