package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
	"github.com/NeoTheCapt/clawfleet/internal/server/auth"
	"github.com/NeoTheCapt/clawfleet/internal/store"
)

type imPrincipal struct {
	SenderType string
	SenderID   string
}

type imPrincipalContextKey struct{}

func (s *Server) requireIMAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// JWT wins if both JWT and X-IM-Key are present.
		if token := auth.ExtractToken(r); token != "" {
			claims, err := s.auth.ValidateToken(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}
			p := imPrincipal{SenderType: string(model.IMMemberTypeUser), SenderID: claims.Sub}
			next(w, r.WithContext(context.WithValue(r.Context(), imPrincipalContextKey{}, p)))
			return
		}

		key := strings.TrimSpace(r.Header.Get("X-IM-Key"))
		if key == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		agentID, err := s.store.FindAgentIDByIMKey(key)
		if err != nil {
			if err == sql.ErrNoRows {
				writeError(w, http.StatusForbidden, "invalid im key")
				return
			}
			writeServerError(w, err)
			return
		}
		p := imPrincipal{SenderType: string(model.IMMemberTypeAgent), SenderID: agentID}
		next(w, r.WithContext(context.WithValue(r.Context(), imPrincipalContextKey{}, p)))
	}
}

func getIMPrincipal(r *http.Request) (imPrincipal, bool) {
	p, ok := r.Context().Value(imPrincipalContextKey{}).(imPrincipal)
	return p, ok
}

func (s *Server) ensureIMConversationAccess(pr imPrincipal, conversationID string) error {
	if pr.SenderType != string(model.IMMemberTypeAgent) {
		return nil
	}
	ok, err := s.store.IsIMConversationMember(conversationID, pr.SenderType, pr.SenderID)
	if err != nil {
		return err
	}
	if !ok {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Server) handleIMConversations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pr, ok := getIMPrincipal(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var (
		conversations []*model.IMConversation
		err           error
	)
	companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	if pr.SenderType == string(model.IMMemberTypeAgent) {
		conversations, err = s.store.ListIMConversationsForMember(pr.SenderType, pr.SenderID, companyID)
	} else {
		conversations, err = s.store.ListIMConversations(companyID)
	}
	if err != nil {
		writeServerError(w, err)
		return
	}
	if conversations == nil {
		conversations = []*model.IMConversation{}
	}
	writeJSON(w, http.StatusOK, conversations)
}

func (s *Server) handleIMConversationByID(w http.ResponseWriter, r *http.Request) {
	pr, ok := getIMPrincipal(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, sub := splitSubpath(r.URL.Path, "/api/im/conversations/")
	if id == "" {
		writeBadRequest(w, "missing conversation id")
		return
	}
	if _, err := s.store.GetIMConversation(id); err != nil {
		if err == sql.ErrNoRows {
			writeNotFound(w, "conversation not found")
			return
		}
		writeServerError(w, err)
		return
	}
	if sub == "" {
		if r.Method != http.MethodDelete {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		if pr.SenderType != string(model.IMMemberTypeUser) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		if err := s.store.DeleteIMConversation(id); err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}
	if strings.HasPrefix(sub, "members") {
		s.handleIMConversationMembers(w, r, pr, id, strings.TrimPrefix(strings.TrimPrefix(sub, "members"), "/"))
		return
	}
	if sub != "messages" {
		writeNotFound(w, "subroute not found")
		return
	}
	if err := s.ensureIMConversationAccess(pr, id); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeServerError(w, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		messages, err := s.store.ListIMMessages(id)
		if err != nil {
			writeServerError(w, err)
			return
		}
		if messages == nil {
			messages = []*model.IMMessage{}
		}
		writeJSON(w, http.StatusOK, messages)
	case http.MethodPost:
		var req struct {
			Body string `json:"body"`
			Meta string `json:"meta"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		if strings.TrimSpace(req.Body) == "" {
			writeBadRequest(w, "body required")
			return
		}
		now := time.Now()
		msg := &model.IMMessage{
			ID:             generateID("immsg"),
			ConversationID: id,
			SenderType:     pr.SenderType,
			SenderID:       pr.SenderID,
			Body:           req.Body,
			Meta:           req.Meta,
			Status:         "sent",
			CreatedAt:      now,
		}
		if err := s.store.CreateIMMessage(msg); err != nil {
			writeServerError(w, err)
			return
		}
		targets, err := s.resolveIMMessageTargets(id, pr, req.Meta)
		if err != nil {
			writeServerError(w, err)
			return
		}
		for _, agentID := range targets {
			if err := s.store.CreateIMOutbox(id, agentID, req.Body); err != nil {
				writeServerError(w, err)
				return
			}
		}
		writeJSON(w, http.StatusCreated, msg)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleIMConversationMembers(w http.ResponseWriter, r *http.Request, pr imPrincipal, conversationID, sub string) {
	if err := s.ensureIMConversationAccess(pr, conversationID); err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		writeServerError(w, err)
		return
	}

	if sub == "" {
		switch r.Method {
		case http.MethodGet:
			members, err := s.store.ListIMConversationMembers(conversationID)
			if err != nil {
				writeServerError(w, err)
				return
			}
			if members == nil {
				members = []*store.IMConversationMemberView{}
			}
			writeJSON(w, http.StatusOK, members)
		case http.MethodPost:
			if !isIMChairman(pr) {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}
			var req struct {
				MemberType string `json:"member_type"`
				MemberID   string `json:"member_id"`
				Role       string `json:"role"`
			}
			if !decodeJSON(w, r, &req) {
				return
			}
			req.MemberType = strings.TrimSpace(req.MemberType)
			req.MemberID = strings.TrimSpace(req.MemberID)
			req.Role = strings.TrimSpace(req.Role)
			if req.MemberType != string(model.IMMemberTypeAgent) {
				writeBadRequest(w, "member_type must be agent")
				return
			}
			if req.MemberID == "" {
				writeBadRequest(w, "member_id required")
				return
			}
			if req.Role == "" {
				req.Role = "member"
			}
			if _, err := s.store.GetAgentInstance(req.MemberID); err != nil {
				if err == sql.ErrNoRows {
					writeNotFound(w, "agent not found")
					return
				}
				writeServerError(w, err)
				return
			}
			if err := s.store.AddIMMemberIfMissing(conversationID, req.MemberType, req.MemberID, req.Role); err != nil {
				writeServerError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if r.Method == http.MethodDelete && strings.HasPrefix(sub, "agent/") {
		if !isIMChairman(pr) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
		agentID := strings.TrimSpace(strings.TrimPrefix(sub, "agent/"))
		if agentID == "" {
			writeBadRequest(w, "missing agent id")
			return
		}
		if err := s.store.RemoveIMMember(conversationID, string(model.IMMemberTypeAgent), agentID); err != nil {
			writeServerError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}
	writeNotFound(w, "subroute not found")
}

func isIMChairman(pr imPrincipal) bool {
	if pr.SenderType != string(model.IMMemberTypeUser) {
		return false
	}
	id := strings.TrimSpace(pr.SenderID)
	return strings.EqualFold(id, model.IMChairmanUserID) || strings.EqualFold(id, "brian")
}

func (s *Server) resolveIMMessageTargets(conversationID string, pr imPrincipal, meta string) ([]string, error) {
	memberAgents, err := s.store.ListIMConversationAgentMemberIDs(conversationID)
	if err != nil {
		return nil, err
	}
	if len(memberAgents) == 0 {
		return nil, nil
	}
	targets := memberAgents
	if mentioned := parseMentionAgents(meta); len(mentioned) > 0 {
		allowed := map[string]struct{}{}
		for _, id := range mentioned {
			allowed[id] = struct{}{}
		}
		filtered := make([]string, 0, len(memberAgents))
		for _, id := range memberAgents {
			if _, ok := allowed[id]; ok {
				filtered = append(filtered, id)
			}
		}
		targets = filtered
	}
	if pr.SenderType == string(model.IMMemberTypeAgent) {
		targets = slices.DeleteFunc(targets, func(id string) bool {
			return id == pr.SenderID
		})
	}
	return targets, nil
}

func parseMentionAgents(meta string) []string {
	meta = strings.TrimSpace(meta)
	if meta == "" {
		return nil
	}
	var payload struct {
		Mentions struct {
			Agents []string `json:"agents"`
		} `json:"mentions"`
	}
	if err := json.Unmarshal([]byte(meta), &payload); err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(payload.Mentions.Agents))
	for _, raw := range payload.Mentions.Agents {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func (s *Server) handleIMSyncOrg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pr, ok := getIMPrincipal(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if pr.SenderType != string(model.IMMemberTypeUser) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req struct {
		CompanyID string `json:"company_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.CompanyID = strings.TrimSpace(req.CompanyID)
	if req.CompanyID == "" {
		writeBadRequest(w, "company_id required")
		return
	}
	if _, err := s.store.GetCompany(req.CompanyID); err != nil {
		if err == sql.ErrNoRows {
			writeNotFound(w, "company not found")
			return
		}
		writeServerError(w, err)
		return
	}

	ceoID := ""
	if _, err := s.store.GetAgentInstance(model.IMCEOAgentID); err == nil {
		ceoID = model.IMCEOAgentID
	} else if err != nil && err != sql.ErrNoRows {
		writeServerError(w, err)
		return
	}

	result, err := s.store.RecomputeIMOrg(req.CompanyID, ceoID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleIMBootstrap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pr, ok := getIMPrincipal(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if pr.SenderType != string(model.IMMemberTypeUser) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	agent, created, err := s.store.EnsureCEOAgentInstance()
	if err != nil {
		writeServerError(w, err)
		return
	}
	joined, err := s.store.AddAgentToAllIMConversations(agent.ID, "ceo")
	if err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"agent":          sanitizeAgentForResponse(agent),
		"created":        created,
		"joined_convs":   joined,
		"agent_identity": "ceo",
	})
}
