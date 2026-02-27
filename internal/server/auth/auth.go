package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Config holds authentication configuration.
type Config struct {
	mu               sync.RWMutex
	AdminUser        string // admin username
	AdminPassHash    []byte // bcrypt hash of admin password
	AgentKey         string // shared secret for node agents
	JWTSecret        []byte // HMAC-SHA256 key for JWT tokens
	TokenTTL         time.Duration
	nodeKeyValidator NodeKeyValidator
}

// NewConfig creates auth config from plaintext credentials.
func NewConfig(adminUser, adminPass, agentKey string) (*Config, error) {
	if adminUser == "" || adminPass == "" {
		return nil, errors.New("admin-user and admin-pass are required")
	}
	if agentKey == "" {
		return nil, errors.New("agent-key is required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPass), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate jwt secret: %w", err)
	}

	return &Config{
		AdminUser:     adminUser,
		AdminPassHash: hash,
		AgentKey:      agentKey,
		JWTSecret:     secret,
		TokenTTL:      24 * time.Hour,
	}, nil
}

// --- Simple JWT (HMAC-SHA256, no external deps) ---

type Claims struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

func (c *Config) GenerateToken(username string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := time.Now()
	claims := Claims{
		Sub: username,
		Iat: now.Unix(),
		Exp: now.Add(c.TokenTTL).Unix(),
	}
	claimsJSON, _ := json.Marshal(claims)

	// header.payload.signature (simplified: no base64url header, just payload+sig)
	header := base64url([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload := base64url(claimsJSON)
	sigInput := header + "." + payload
	sig := hmacSHA256([]byte(sigInput), c.JWTSecret)

	return sigInput + "." + base64url(sig), nil
}

func (c *Config) ValidateToken(token string) (*Claims, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token format")
	}

	sigInput := parts[0] + "." + parts[1]
	expectedSig := hmacSHA256([]byte(sigInput), c.JWTSecret)
	actualSig, err := base64urlDecode(parts[2])
	if err != nil {
		return nil, errors.New("invalid signature encoding")
	}
	if !hmac.Equal(expectedSig, actualSig) {
		return nil, errors.New("invalid signature")
	}

	payloadBytes, err := base64urlDecode(parts[1])
	if err != nil {
		return nil, errors.New("invalid payload encoding")
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, errors.New("invalid claims")
	}

	if time.Now().Unix() > claims.Exp {
		return nil, errors.New("token expired")
	}

	return &claims, nil
}

// CheckPassword verifies the admin password.
func (c *Config) CheckPassword(user, pass string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if user != c.AdminUser {
		return false
	}
	return bcrypt.CompareHashAndPassword(c.AdminPassHash, []byte(pass)) == nil
}

// Snapshot returns a consistent copy of current admin credentials + secret.
func (c *Config) Snapshot() (user string, passHash []byte, jwtSecret []byte) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ph := make([]byte, len(c.AdminPassHash))
	copy(ph, c.AdminPassHash)
	s := make([]byte, len(c.JWTSecret))
	copy(s, c.JWTSecret)
	return c.AdminUser, ph, s
}

// SetPassHash replaces the admin password hash.
func (c *Config) SetPassHash(hash []byte) error {
	if len(hash) == 0 {
		return errors.New("pass hash required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AdminPassHash = hash
	return nil
}

// UpdateCredentials updates the admin username/password hash.
// If newPass is non-empty, it will be hashed and stored.
func (c *Config) UpdateCredentials(newUser, newPass string) error {
	if newUser == "" {
		return errors.New("username required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.AdminUser = newUser
	if newPass != "" {
		h, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		c.AdminPassHash = h
	}
	return nil
}

// SetSecret replaces the JWT secret (invalidates existing tokens).
func (c *Config) SetSecret(secret []byte) error {
	if len(secret) < 16 {
		return errors.New("secret too short")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.JWTSecret = secret
	return nil
}

// CheckAgentKey verifies the shared agent key.
func (c *Config) CheckAgentKey(key string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return hmac.Equal([]byte(c.AgentKey), []byte(key))
}

// NodeKeyValidator is a function that checks if a key belongs to any registered node.
type NodeKeyValidator func(key string) bool

// SetNodeKeyValidator sets the per-node key validator callback.
func (c *Config) SetNodeKeyValidator(v NodeKeyValidator) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodeKeyValidator = v
}

// CheckAnyAgentKey checks global key first, then per-node keys.
func (c *Config) CheckAnyAgentKey(key string) bool {
	if c.CheckAgentKey(key) {
		return true
	}
	c.mu.RLock()
	v := c.nodeKeyValidator
	c.mu.RUnlock()
	if v != nil {
		return v(key)
	}
	return false
}

// --- Middleware ---

// RequireJWT returns middleware that validates JWT from Authorization header or cookie.
func (c *Config) RequireJWT(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		if _, err := c.ValidateToken(token); err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// RequireAgentKey returns middleware that validates X-Agent-Key header.
func (c *Config) RequireAgentKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Agent-Key")
		if !c.CheckAnyAgentKey(key) {
			http.Error(w, `{"error":"invalid agent key"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func extractToken(r *http.Request) string {
	// Check Authorization header first
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	// Check cookie
	if cookie, err := r.Cookie("token"); err == nil {
		return cookie.Value
	}
	// Check query param (for WebSocket etc)
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	return ""
}

// --- base64url helpers (no padding, URL-safe) ---

func base64url(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func base64urlDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

func hmacSHA256(data, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
