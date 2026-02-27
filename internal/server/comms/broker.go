package comms

import (
	"log"
	"sync"
)

// Broker handles inter-agent messaging.
// Currently a simple in-memory stub; will be replaced by NATS/Redis in production.
type Broker struct {
	mu    sync.Mutex
	inbox map[string][]interface{}
}

func NewBroker() *Broker { return &Broker{inbox: make(map[string][]interface{})} }

// Send queues a message to an agent's inbox.
func (b *Broker) Send(agentID string, msg interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.inbox[agentID] = append(b.inbox[agentID], msg)
	log.Printf("[broker] message queued for agent %s (%d in inbox)", agentID, len(b.inbox[agentID]))
}

// Receive returns and clears pending messages for an agent.
func (b *Broker) Receive(agentID string) []interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	msgs := b.inbox[agentID]
	delete(b.inbox, agentID)
	return msgs
}
