package scheduler

import (
	"github.com/NeoTheCapt/clawfleet/internal/server/comms"
	"github.com/NeoTheCapt/clawfleet/internal/store"
)

// Scheduler manages company-based deployments.
// The old cluster template engine has been replaced by company org structure.
type Scheduler struct {
	store  *store.Store
	broker *comms.Broker
}

func New(s *store.Store, broker *comms.Broker) *Scheduler {
	return &Scheduler{store: s, broker: broker}
}
