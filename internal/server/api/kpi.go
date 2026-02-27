package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

func (s *Server) handleKPITriggerDaily(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		CompanyID         string `json:"company_id"`
		Scope             string `json:"scope"`
		DepartmentID      string `json:"department_id"`
		ManagerPositionID string `json:"manager_position_id"`
		ManagerAgentID    string `json:"manager_agent_id"`
		AgentID           string `json:"agent_id"`
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
		writeNotFound(w, "company not found")
		return
	}

	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = "company"
	}
	now := time.Now()
	period := now.Format("2006-01-02")

	cycle, err := s.store.GetReviewCycleByFrequency(req.CompanyID, "daily")
	if err == sql.ErrNoRows {
		cycle = &model.ReviewCycle{
			ID:         generateID("cycle"),
			CompanyID:  req.CompanyID,
			Name:       "Daily Review",
			Frequency:  "daily",
			AutoReview: true,
			Status:     "active",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := s.store.CreateReviewCycle(cycle); err != nil {
			writeServerError(w, err)
			return
		}
	} else if err != nil {
		writeServerError(w, err)
		return
	}

	assignments, err := s.store.ListAssignments(req.CompanyID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	positions, err := s.store.ListAllPositions(req.CompanyID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	positionByID := map[string]*model.Position{}
	for _, p := range positions {
		positionByID[p.ID] = p
	}

	activeAssignments := make([]*model.AgentAssignment, 0, len(assignments))
	assignmentsByPosition := map[string][]*model.AgentAssignment{}
	assignmentsByAgent := map[string][]*model.AgentAssignment{}
	for _, a := range assignments {
		if a.Status != "" && a.Status != "active" {
			continue
		}
		activeAssignments = append(activeAssignments, a)
		assignmentsByPosition[a.PositionID] = append(assignmentsByPosition[a.PositionID], a)
		assignmentsByAgent[a.AgentID] = append(assignmentsByAgent[a.AgentID], a)
	}

	targets, err := selectDailyScopeTargets(scope, &req, activeAssignments, assignmentsByAgent, positionByID)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}

	var ceo *model.AgentInstance
	ensureCEO := func() (string, error) {
		if ceo != nil {
			return ceo.ID, nil
		}
		agent, _, err := s.store.EnsureCEOAgentInstance()
		if err != nil {
			return "", err
		}
		ceo = agent
		return ceo.ID, nil
	}

	created := 0
	existing := 0
	reviews := make([]*model.KPIReview, 0, len(targets))
	for _, a := range targets {
		review, err := s.store.GetKPIReviewByCycleAgentPeriod(cycle.ID, a.AgentID, period)
		if err == nil {
			reviews = append(reviews, review)
			existing++
			continue
		}
		if err != sql.ErrNoRows {
			writeServerError(w, err)
			return
		}

		reviewerID := ""
		if p := positionByID[a.PositionID]; p != nil && strings.TrimSpace(p.ReportsTo) != "" {
			if managers := assignmentsByPosition[p.ReportsTo]; len(managers) > 0 {
				reviewerID = managers[0].AgentID
			}
		}
		if reviewerID == "" {
			fallback, err := ensureCEO()
			if err != nil {
				writeServerError(w, err)
				return
			}
			reviewerID = fallback
		}

		review = &model.KPIReview{
			ID:         generateID("kpi"),
			AgentID:    a.AgentID,
			CycleID:    cycle.ID,
			Period:     period,
			SelfReport: cycle.ReportTemplate,
			ReviewerID: reviewerID,
			Status:     "pending",
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := s.store.CreateKPIReview(review); err != nil {
			writeServerError(w, err)
			return
		}
		reviews = append(reviews, review)
		created++
	}

	if err := s.store.UpdateReviewCycleLastTriggered(cycle.ID, now); err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"cycle":          cycle,
		"period":         period,
		"scope":          scope,
		"created_count":  created,
		"existing_count": existing,
		"reviews":        reviews,
	})
}

func selectDailyScopeTargets(
	scope string,
	req *struct {
		CompanyID         string `json:"company_id"`
		Scope             string `json:"scope"`
		DepartmentID      string `json:"department_id"`
		ManagerPositionID string `json:"manager_position_id"`
		ManagerAgentID    string `json:"manager_agent_id"`
		AgentID           string `json:"agent_id"`
	},
	activeAssignments []*model.AgentAssignment,
	assignmentsByAgent map[string][]*model.AgentAssignment,
	positionByID map[string]*model.Position,
) ([]*model.AgentAssignment, error) {
	switch scope {
	case "company":
		return activeAssignments, nil
	case "department":
		if req.DepartmentID == "" {
			return nil, fmt.Errorf("department_id required for department scope")
		}
		out := make([]*model.AgentAssignment, 0)
		for _, a := range activeAssignments {
			if p := positionByID[a.PositionID]; p != nil && p.DepartmentID == req.DepartmentID {
				out = append(out, a)
			}
		}
		return out, nil
	case "manager":
		managerPositionID := strings.TrimSpace(req.ManagerPositionID)
		if managerPositionID == "" {
			managerAgentID := strings.TrimSpace(req.ManagerAgentID)
			if managerAgentID == "" {
				return nil, fmt.Errorf("manager_position_id or manager_agent_id required for manager scope")
			}
			assigns := assignmentsByAgent[managerAgentID]
			if len(assigns) == 0 {
				return nil, fmt.Errorf("manager agent assignment not found in company")
			}
			managerPositionID = assigns[0].PositionID
		}
		out := make([]*model.AgentAssignment, 0)
		for _, a := range activeAssignments {
			if p := positionByID[a.PositionID]; p != nil && p.ReportsTo == managerPositionID {
				out = append(out, a)
			}
		}
		return out, nil
	case "agent":
		agentID := strings.TrimSpace(req.AgentID)
		if agentID == "" {
			return nil, fmt.Errorf("agent_id required for agent scope")
		}
		assigns := assignmentsByAgent[agentID]
		if len(assigns) == 0 {
			return nil, fmt.Errorf("agent assignment not found in company")
		}
		return []*model.AgentAssignment{assigns[0]}, nil
	default:
		return nil, fmt.Errorf("invalid scope, expected company|department|manager|agent")
	}
}

func (s *Server) handleKPIReviewSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	id, sub := splitSubpath(r.URL.Path, "/api/kpi/reviews/")
	if id == "" || sub != "submit" {
		writeNotFound(w, "not found")
		return
	}
	var req struct {
		Feedback string `json:"feedback"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Feedback = strings.TrimSpace(req.Feedback)
	if req.Feedback == "" {
		writeBadRequest(w, "feedback required")
		return
	}

	review, err := s.store.GetKPIReview(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeNotFound(w, "kpi review not found")
			return
		}
		writeServerError(w, err)
		return
	}
	cycle, err := s.store.GetReviewCycle(review.CycleID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	review.Feedback = req.Feedback
	review.Status = "reviewed"
	if err := s.store.UpdateKPIReview(review); err != nil {
		writeServerError(w, err)
		return
	}

	reviewerID := strings.TrimSpace(review.ReviewerID)
	if reviewerID == "" {
		ceo, _, err := s.store.EnsureCEOAgentInstance()
		if err != nil {
			writeServerError(w, err)
			return
		}
		reviewerID = ceo.ID
	}

	// DM to reviewee
	dmRef := fmt.Sprintf("dm:%s", canonicalDMRefID(reviewerID, review.AgentID))
	dmTitle := "KPI Review DM"
	dmConv, err := s.store.UpsertIMConversation("direct", "", dmRef, dmTitle)
	if err != nil {
		writeServerError(w, err)
		return
	}
	if err := s.store.AddIMMemberIfMissing(dmConv.ID, string(model.IMMemberTypeAgent), reviewerID, "reviewer"); err != nil {
		writeServerError(w, err)
		return
	}
	if err := s.store.AddIMMemberIfMissing(dmConv.ID, string(model.IMMemberTypeAgent), review.AgentID, "reviewee"); err != nil {
		writeServerError(w, err)
		return
	}
	dmBody := fmt.Sprintf("KPI feedback for %s (%s): %s", cycle.Name, review.Period, review.Feedback)
	if err := s.store.CreateIMMessage(&model.IMMessage{
		ID:             generateID("immsg"),
		ConversationID: dmConv.ID,
		SenderType:     string(model.IMMemberTypeAgent),
		SenderID:       reviewerID,
		Body:           dmBody,
		Meta:           "{}",
		Status:         "sent",
		CreatedAt:      time.Now(),
	}); err != nil {
		writeServerError(w, err)
		return
	}

	// Manager line conversation summary
	assignments, err := s.store.ListAssignments(cycle.CompanyID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	var reviewAssignment *model.AgentAssignment
	for _, a := range assignments {
		if (a.Status == "" || a.Status == "active") && a.AgentID == review.AgentID {
			reviewAssignment = a
			break
		}
	}
	if reviewAssignment != nil {
		position, err := s.store.GetPosition(reviewAssignment.PositionID)
		if err == nil && strings.TrimSpace(position.ReportsTo) != "" {
			managerPos, err := s.store.GetPosition(position.ReportsTo)
			if err == nil {
				lineRefID := managerPos.ID + ":" + position.ID
				lineTitle := managerPos.Title + " <> " + position.Title
				lineConv, err := s.store.UpsertIMConversation("line", cycle.CompanyID, lineRefID, lineTitle)
				if err != nil {
					writeServerError(w, err)
					return
				}
				_ = s.store.AddIMMemberIfMissing(lineConv.ID, string(model.IMMemberTypeUser), model.IMChairmanUserID, "chairman")
				managerAssignments := make([]*model.AgentAssignment, 0)
				for _, a := range assignments {
					if (a.Status == "" || a.Status == "active") && a.PositionID == managerPos.ID {
						managerAssignments = append(managerAssignments, a)
					}
				}
				for _, ma := range managerAssignments {
					_ = s.store.AddIMMemberIfMissing(lineConv.ID, string(model.IMMemberTypeAgent), ma.AgentID, "manager")
				}
				if ceo, err := s.store.GetAgentInstance(model.IMCEOAgentID); err == nil {
					_ = s.store.AddIMMemberIfMissing(lineConv.ID, string(model.IMMemberTypeAgent), ceo.ID, "ceo")
				}
				_ = s.store.AddIMMemberIfMissing(lineConv.ID, string(model.IMMemberTypeAgent), review.AgentID, "report")
				if err := s.store.CreateIMMessage(&model.IMMessage{
					ID:             generateID("immsg"),
					ConversationID: lineConv.ID,
					SenderType:     string(model.IMMemberTypeAgent),
					SenderID:       reviewerID,
					Body:           fmt.Sprintf("KPI summary (%s, %s): %s", cycle.Name, review.Period, review.Feedback),
					Meta:           "{}",
					Status:         "sent",
					CreatedAt:      time.Now(),
				}); err != nil {
					writeServerError(w, err)
					return
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, review)
}

func canonicalDMRefID(a, b string) string {
	if a <= b {
		return a + ":" + b
	}
	return b + ":" + a
}
