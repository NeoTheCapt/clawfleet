package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// StartKPICron starts a background goroutine that checks for auto-review cycles
// and creates pending KPI reviews when due.
func (s *Scheduler) StartKPICron() {
	go func() {
		log.Println("[kpi-cron] started, checking every 15 minutes")
		// Check on startup after a short delay
		time.Sleep(30 * time.Second)
		s.checkAutoReviews()

		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			s.checkAutoReviews()
		}
	}()
}

func (s *Scheduler) checkAutoReviews() {
	cycles, err := s.store.ListAutoReviewCycles()
	if err != nil {
		log.Printf("[kpi-cron] error listing auto review cycles: %v", err)
		return
	}

	now := time.Now()
	for _, cycle := range cycles {
		if !s.isDue(cycle, now) {
			continue
		}
		log.Printf("[kpi-cron] cycle %q (%s) is due, triggering reviews", cycle.Name, cycle.Frequency)
		s.triggerReviews(cycle, now)
	}
}

// isDue checks if a review cycle needs to trigger based on frequency and last_triggered.
func (s *Scheduler) isDue(cycle *model.ReviewCycle, now time.Time) bool {
	interval := frequencyToDuration(cycle.Frequency)
	if interval == 0 {
		return false
	}

	// Never triggered before
	if cycle.LastTriggered.IsZero() {
		// Check if enough time has passed since creation
		return now.Sub(cycle.CreatedAt) >= interval
	}

	return now.Sub(cycle.LastTriggered) >= interval
}

// triggerReviews creates pending KPI review entries for all agents assigned to the company.
func (s *Scheduler) triggerReviews(cycle *model.ReviewCycle, now time.Time) {
	// Get all agent assignments for this company
	assignments, err := s.store.ListAssignments(cycle.CompanyID)
	if err != nil {
		log.Printf("[kpi-cron] error listing assignments for company %s: %v", cycle.CompanyID, err)
		return
	}

	if len(assignments) == 0 {
		log.Printf("[kpi-cron] no assignments for company %s, skipping", cycle.CompanyID)
		return
	}

	period := computePeriod(cycle.Frequency, now)
	created := 0

	for _, assign := range assignments {
		review := &model.KPIReview{
			ID:         fmt.Sprintf("kpi_%s_%09d", now.Format("20060102150405"), now.UnixNano()%1000000000),
			AgentID:    assign.AgentID,
			CycleID:    cycle.ID,
			Period:     period,
			Status:     "pending",
			SelfReport: cycle.ReportTemplate, // Pre-fill with template as prompt
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := s.store.CreateKPIReview(review); err != nil {
			log.Printf("[kpi-cron] error creating review for agent %s: %v", assign.AgentID, err)
			continue
		}
		created++

		// Send review task to agent via broker
		if s.broker != nil {
			s.broker.Send(assign.AgentID, map[string]interface{}{
				"type":      "kpi_review_request",
				"review_id": review.ID,
				"cycle":     cycle.Name,
				"period":    period,
				"metrics":   cycle.Metrics,
				"template":  cycle.ReportTemplate,
				"message":   fmt.Sprintf("请完成 %s 的绩效自评 (周期: %s)。请根据以下指标评估你的表现：%v", cycle.Name, period, cycle.Metrics),
			})
		}
	}

	// Update last triggered
	if err := s.store.UpdateReviewCycleLastTriggered(cycle.ID, now); err != nil {
		log.Printf("[kpi-cron] warning: failed to update last_triggered for cycle %s: %v", cycle.ID, err)
	}
	log.Printf("[kpi-cron] created %d pending reviews for cycle %q (period: %s)", created, cycle.Name, period)
}

func frequencyToDuration(freq string) time.Duration {
	switch freq {
	case "daily":
		return 24 * time.Hour
	case "weekly":
		return 7 * 24 * time.Hour
	case "biweekly":
		return 14 * 24 * time.Hour
	case "monthly":
		return 30 * 24 * time.Hour
	case "quarterly":
		return 90 * 24 * time.Hour
	default:
		return 0
	}
}

func computePeriod(freq string, now time.Time) string {
	switch freq {
	case "daily":
		return now.Format("2006-01-02")
	case "weekly":
		y, w := now.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	case "biweekly":
		y, w := now.ISOWeek()
		return fmt.Sprintf("%d-W%02d", y, w)
	case "monthly":
		return now.Format("2006-01")
	case "quarterly":
		q := (now.Month()-1)/3 + 1
		return fmt.Sprintf("%d-Q%d", now.Year(), q)
	default:
		return now.Format("2006-01-02")
	}
}

// removed: randomSuffix inlined above
