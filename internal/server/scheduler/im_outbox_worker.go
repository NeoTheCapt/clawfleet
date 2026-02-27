package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
	"github.com/NeoTheCapt/clawfleet/internal/store"
	"github.com/NeoTheCapt/clawfleet/internal/util"
)

// StartIMOutboxWorker starts a background loop that consumes IM outbox items and generates
// agent replies via OpenRouter. This is the "fast path" to make Web IM work without relying
// on OpenClaw channel plugins.
func (s *Scheduler) StartIMOutboxWorker() {
	go func() {
		log.Println("[im-outbox] worker started (interval=1s)")
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.processIMOutboxOnce()
		}
	}()
}

func (s *Scheduler) processIMOutboxOnce() {
	items, err := s.store.ListPendingIMOutbox(20)
	if err != nil {
		log.Printf("[im-outbox] list error: %v", err)
		return
	}
	for _, it := range items {
		if it == nil {
			continue
		}
		// Best-effort claim to avoid double processing.
		claimed, err := s.store.ClaimIMOutbox(it.ID)
		if err != nil {
			log.Printf("[im-outbox] claim %s error: %v", it.ID, err)
			continue
		}
		if !claimed {
			continue
		}

		if err := s.handleIMOutboxItem(it); err != nil {
			log.Printf("[im-outbox] handle %s error: %v", it.ID, err)
			_ = s.store.FailIMOutbox(it.ID)
		}
	}
}

func (s *Scheduler) handleIMOutboxItem(it *store.IMOutboxItem) error {
	agentID := strings.TrimSpace(it.ToAgentID)
	if agentID == "" {
		return s.store.FailIMOutbox(it.ID)
	}

	agent, err := s.store.GetAgentInstance(agentID)
	if err != nil {
		return err
	}

	apiKey := strings.TrimSpace(agent.Config.EnvVars["OPENROUTER_API_KEY"])
	if apiKey == "" {
		apiKey = strings.TrimSpace(agent.Config.EnvVars["OPENROUTER_KEY"])
	}
	if apiKey == "" {
		return s.store.FailIMOutbox(it.ID)
	}

	modelName := strings.TrimSpace(agent.Config.Model)
	if modelName == "" {
		modelName = "minimax/minimax-m2.5"
	}
	// OpenRouter expects model ids like "minimax/minimax-m2.5" (no leading "openrouter/").
	modelName = strings.TrimPrefix(modelName, "openrouter/")
	provider := strings.TrimSpace(agent.Config.Provider)
	if provider != "" && provider != "openrouter" {
		if !strings.HasPrefix(modelName, provider+"/") {
			modelName = provider + "/" + modelName
		}
	}

	systemPrompt := strings.TrimSpace(agent.Config.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = "You are an AI agent in an internal company chat. Be helpful, concise, and act according to your role."
	}

	msgs, err := s.store.ListIMMessages(it.ConversationID)
	if err != nil {
		return err
	}

	chatMsgs := make([]map[string]string, 0, len(msgs)+2)
	chatMsgs = append(chatMsgs, map[string]string{"role": "system", "content": systemPrompt})

	// Use last N messages to keep prompt bounded.
	start := 0
	if len(msgs) > 20 {
		start = len(msgs) - 20
	}
	for _, m := range msgs[start:] {
		if m == nil {
			continue
		}
		content := strings.TrimSpace(m.Body)
		if content == "" {
			continue
		}
		role := "user"
		if m.SenderType == string(model.IMMemberTypeAgent) {
			role = "assistant"
		}
		chatMsgs = append(chatMsgs, map[string]string{"role": role, "content": content})
	}

	replyText, err := openRouterChat(apiKey, modelName, chatMsgs)
	if err != nil {
		return err
	}
	replyText = strings.TrimSpace(replyText)
	if replyText == "" {
		return s.store.FailIMOutbox(it.ID)
	}

	// Write agent reply into conversation
	now := time.Now()
	msg := &model.IMMessage{
		ID:             util.GenerateID("immsg"),
		ConversationID: it.ConversationID,
		SenderType:     string(model.IMMemberTypeAgent),
		SenderID:       agentID,
		Body:           replyText,
		Meta:           `{"kind":"auto_reply"}`,
		Status:         "sent",
		CreatedAt:      now,
	}
	if err := s.store.CreateIMMessage(msg); err != nil {
		return err
	}
	if err := s.store.AckIMOutbox(it.ID); err != nil {
		return err
	}
	return nil
}

func openRouterChat(apiKey, modelName string, messages []map[string]string) (string, error) {
	reqBody := map[string]interface{}{
		"model":       modelName,
		"messages":    messages,
		"temperature": 0.4,
	}
	buf, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(buf))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://clawfleet.local")
	httpReq.Header.Set("X-Title", "Clawfleet Web IM")

	client := &http.Client{Timeout: 35 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil || resp == nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openrouter non-2xx: %d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openrouter parse failed")
	}
	return parsed.Choices[0].Message.Content, nil
}
