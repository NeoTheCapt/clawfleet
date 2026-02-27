package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

// ═══════════════════════════════════════════════════════════
// Review Cycle CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateReviewCycle(r *model.ReviewCycle) error {
	metrics, _ := json.Marshal(r.Metrics)
	_, err := s.db.Exec(`INSERT INTO review_cycles (id,company_id,name,frequency,metrics,report_template,auto_review,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		r.ID, r.CompanyID, r.Name, r.Frequency, string(metrics), r.ReportTemplate, r.AutoReview, r.Status, r.CreatedAt, r.UpdatedAt)
	return err
}

func (s *Store) ListReviewCycles(companyID string) ([]*model.ReviewCycle, error) {
	return queryReviewCycles(s.db, `SELECT id,company_id,name,frequency,metrics,report_template,auto_review,COALESCE(last_triggered,''),COALESCE(status,'active'),created_at,updated_at FROM review_cycles WHERE company_id=? ORDER BY created_at`, companyID)
}

func (s *Store) GetReviewCycle(id string) (*model.ReviewCycle, error) {
	return scanReviewCycle(s.db.QueryRow(`SELECT id,company_id,name,frequency,metrics,report_template,auto_review,COALESCE(last_triggered,''),COALESCE(status,'active'),created_at,updated_at FROM review_cycles WHERE id=?`, id))
}

func (s *Store) GetReviewCycleByFrequency(companyID, frequency string) (*model.ReviewCycle, error) {
	return scanReviewCycle(s.db.QueryRow(`SELECT id,company_id,name,frequency,metrics,report_template,auto_review,COALESCE(last_triggered,''),COALESCE(status,'active'),created_at,updated_at FROM review_cycles WHERE company_id=? AND frequency=? ORDER BY created_at ASC LIMIT 1`, companyID, frequency))
}

func (s *Store) ListAutoReviewCycles() ([]*model.ReviewCycle, error) {
	return queryReviewCycles(s.db, `SELECT id,company_id,name,frequency,metrics,report_template,auto_review,COALESCE(last_triggered,''),COALESCE(status,'active'),created_at,updated_at FROM review_cycles WHERE auto_review=1 AND status='active' ORDER BY created_at`)
}

func (s *Store) UpdateReviewCycle(r *model.ReviewCycle) error {
	metrics, _ := json.Marshal(r.Metrics)
	_, err := s.db.Exec(`UPDATE review_cycles SET name=?,frequency=?,metrics=?,report_template=?,auto_review=?,status=?,updated_at=? WHERE id=?`,
		r.Name, r.Frequency, string(metrics), r.ReportTemplate, r.AutoReview, r.Status, time.Now(), r.ID)
	return err
}

func (s *Store) UpdateReviewCycleLastTriggered(id string, t time.Time) error {
	_, err := s.db.Exec(`UPDATE review_cycles SET last_triggered=? WHERE id=?`, t, id)
	return err
}

func (s *Store) DeleteReviewCycle(id string) error {
	_, err := s.db.Exec(`DELETE FROM review_cycles WHERE id=?`, id)
	return err
}

func scanReviewCycle(row scannable) (*model.ReviewCycle, error) {
	var r model.ReviewCycle
	var metrics, lastTriggered string
	if err := row.Scan(&r.ID, &r.CompanyID, &r.Name, &r.Frequency, &metrics, &r.ReportTemplate, &r.AutoReview, &lastTriggered, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	if strings.TrimSpace(metrics) != "" {
		if err := json.Unmarshal([]byte(metrics), &r.Metrics); err != nil {
			return nil, err
		}
	}
	if lastTriggered != "" {
		r.LastTriggered, _ = time.Parse(time.RFC3339, lastTriggered)
	}
	return &r, nil
}

func queryReviewCycles(db *sql.DB, query string, args ...interface{}) ([]*model.ReviewCycle, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.ReviewCycle, error) { return scanReviewCycle(r) })
}

// ═══════════════════════════════════════════════════════════
// KPI Review CRUD
// ═══════════════════════════════════════════════════════════

func (s *Store) CreateKPIReview(k *model.KPIReview) error {
	metrics, _ := json.Marshal(k.Metrics)
	_, err := s.db.Exec(`INSERT INTO kpi_reviews (id,agent_id,cycle_id,period,metrics,score,self_report,reviewer_id,feedback,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		k.ID, k.AgentID, k.CycleID, k.Period, string(metrics), k.Score, k.SelfReport, k.ReviewerID, k.Feedback, k.Status, k.CreatedAt, k.UpdatedAt)
	return err
}

func (s *Store) ListKPIReviews(agentID string) ([]*model.KPIReview, error) {
	return queryKPIReviews(s.db, `SELECT id,agent_id,cycle_id,period,metrics,score,self_report,reviewer_id,feedback,status,created_at,updated_at FROM kpi_reviews WHERE agent_id=? ORDER BY created_at DESC`, agentID)
}

func (s *Store) ListKPIReviewsByCycle(cycleID string, period string) ([]*model.KPIReview, error) {
	if period != "" {
		return queryKPIReviews(s.db, `SELECT id,agent_id,cycle_id,period,metrics,score,self_report,reviewer_id,feedback,status,created_at,updated_at FROM kpi_reviews WHERE cycle_id=? AND period=? ORDER BY score DESC`, cycleID, period)
	}
	return queryKPIReviews(s.db, `SELECT id,agent_id,cycle_id,period,metrics,score,self_report,reviewer_id,feedback,status,created_at,updated_at FROM kpi_reviews WHERE cycle_id=? ORDER BY created_at DESC`, cycleID)
}

func (s *Store) ListKPIReviewsByCompanyPeriod(companyID, period string) ([]*model.KPIReview, error) {
	return queryKPIReviews(s.db, `SELECT k.id,k.agent_id,k.cycle_id,k.period,k.metrics,k.score,k.self_report,k.reviewer_id,k.feedback,k.status,k.created_at,k.updated_at
		FROM kpi_reviews k
		JOIN review_cycles c ON c.id=k.cycle_id
		WHERE c.company_id=? AND k.period=?
		ORDER BY k.created_at DESC`, companyID, period)
}

func (s *Store) GetKPIReview(id string) (*model.KPIReview, error) {
	return scanKPIReview(s.db.QueryRow(`SELECT id,agent_id,cycle_id,period,metrics,score,self_report,reviewer_id,feedback,status,created_at,updated_at FROM kpi_reviews WHERE id=?`, id))
}

func (s *Store) GetKPIReviewByCycleAgentPeriod(cycleID, agentID, period string) (*model.KPIReview, error) {
	return scanKPIReview(s.db.QueryRow(`SELECT id,agent_id,cycle_id,period,metrics,score,self_report,reviewer_id,feedback,status,created_at,updated_at FROM kpi_reviews WHERE cycle_id=? AND agent_id=? AND period=? ORDER BY created_at DESC LIMIT 1`, cycleID, agentID, period))
}

func (s *Store) UpdateKPIReview(k *model.KPIReview) error {
	metrics, _ := json.Marshal(k.Metrics)
	_, err := s.db.Exec(`UPDATE kpi_reviews SET metrics=?,score=?,self_report=?,reviewer_id=?,feedback=?,status=?,updated_at=? WHERE id=?`,
		string(metrics), k.Score, k.SelfReport, k.ReviewerID, k.Feedback, k.Status, time.Now(), k.ID)
	return err
}

func scanKPIReview(row scannable) (*model.KPIReview, error) {
	var k model.KPIReview
	var metrics string
	if err := row.Scan(&k.ID, &k.AgentID, &k.CycleID, &k.Period, &metrics, &k.Score, &k.SelfReport, &k.ReviewerID, &k.Feedback, &k.Status, &k.CreatedAt, &k.UpdatedAt); err != nil {
		return nil, err
	}
	if strings.TrimSpace(metrics) != "" {
		if err := json.Unmarshal([]byte(metrics), &k.Metrics); err != nil {
			return nil, err
		}
	}
	return &k, nil
}

func queryKPIReviews(db *sql.DB, query string, args ...interface{}) ([]*model.KPIReview, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.KPIReview, error) { return scanKPIReview(r) })
}
