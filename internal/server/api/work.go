package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
)

type workPlanItem struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	OwnerPositionID string `json:"owner_position_id,omitempty"`
	Priority        string `json:"priority,omitempty"`
	DueAt           string `json:"due_at,omitempty"`
}

type workPlanPayload struct {
	Items []workPlanItem `json:"items"`
}

func (s *Server) handleWork(w http.ResponseWriter, r *http.Request) {
	id, sub := splitSubpath(r.URL.Path, "/api/work/")
	if id == "" {
		writeNotFound(w, "not found")
		return
	}
	switch sub {
	case "status":
		s.auth.RequireJWT(func(w http.ResponseWriter, r *http.Request) {
			s.handleWorkStatus(w, r, id)
		})(w, r)
	case "update":
		s.requireIMAuth(func(w http.ResponseWriter, r *http.Request) {
			s.handleWorkUpdate(w, r, id)
		})(w, r)
	default:
		writeNotFound(w, "not found")
	}
}

func (s *Server) handleWorkList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	companyID := strings.TrimSpace(r.URL.Query().Get("company_id"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if companyID == "" {
		writeBadRequest(w, "company_id required")
		return
	}
	list, err := s.store.ListWorkItems(companyID, status)
	if err != nil {
		writeServerError(w, err)
		return
	}
	if list == nil {
		list = []*model.WorkItem{}
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleWorkPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		CompanyID string `json:"company_id"`
		GoalText  string `json:"goal_text"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.CompanyID = strings.TrimSpace(req.CompanyID)
	req.GoalText = strings.TrimSpace(req.GoalText)
	if req.CompanyID == "" || req.GoalText == "" {
		writeBadRequest(w, "company_id and goal_text required")
		return
	}
	positions, err := s.store.ListAllPositions(req.CompanyID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	plan, warning := s.generateWorkPlan(req.GoalText, positions)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"plan":    plan,
		"warning": warning,
	})
}

func (s *Server) handleWorkApprove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		CompanyID string `json:"company_id"`
		GoalText  string `json:"goal_text"`
		Plan      struct {
			Items []workPlanItem `json:"items"`
		} `json:"plan"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.CompanyID = strings.TrimSpace(req.CompanyID)
	req.GoalText = strings.TrimSpace(req.GoalText)
	if req.CompanyID == "" || req.GoalText == "" {
		writeBadRequest(w, "company_id and goal_text required")
		return
	}
	if len(req.Plan.Items) == 0 {
		writeBadRequest(w, "plan.items required")
		return
	}
	for i, item := range req.Plan.Items {
		if strings.TrimSpace(item.Title) == "" {
			writeBadRequest(w, fmt.Sprintf("plan.items[%d].title required", i))
			return
		}
	}
	if _, err := s.store.GetCompany(req.CompanyID); err != nil {
		writeNotFound(w, "company not found")
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
	departments, err := s.store.ListDepartments(req.CompanyID)
	if err != nil {
		writeServerError(w, err)
		return
	}
	positionByID := map[string]*model.Position{}
	for _, p := range positions {
		positionByID[p.ID] = p
	}
	departmentByID := map[string]*model.Department{}
	for _, d := range departments {
		departmentByID[d.ID] = d
	}
	assignmentsByPosition := map[string][]*model.AgentAssignment{}
	for _, a := range assignments {
		if a.Status != "" && a.Status != "active" {
			continue
		}
		assignmentsByPosition[a.PositionID] = append(assignmentsByPosition[a.PositionID], a)
	}

	ceoID := ""
	if ceo, _, err := s.store.EnsureCEOAgentInstance(); err == nil {
		ceoID = ceo.ID
	}

	now := time.Now()
	parentMeta, _ := json.Marshal(map[string]interface{}{
		"goal_text": req.GoalText,
		"plan":      req.Plan,
	})
	parent := &model.WorkItem{
		ID:              generateID("work"),
		CompanyID:       req.CompanyID,
		Title:           "Execution Plan",
		Description:     req.GoalText,
		Status:          string(model.WorkItemStatusApproved),
		CreatedByType:   "user",
		CreatedByID:     model.IMChairmanUserID,
		OwnerAgentID:    ceoID,
		ReviewerAgentID: ceoID,
		Priority:        "high",
		Meta:            string(parentMeta),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.store.CreateWorkItem(parent); err != nil {
		writeServerError(w, err)
		return
	}

	childIDs := make([]string, 0, len(req.Plan.Items))
	for _, item := range req.Plan.Items {
		title := strings.TrimSpace(item.Title)
		priority := normalizePriority(item.Priority)
		var dueAt *time.Time
		if t, ok := parseDueAt(item.DueAt); ok {
			dueAt = &t
		}

		ownerAgentID := ""
		if item.OwnerPositionID != "" {
			if owners := assignmentsByPosition[item.OwnerPositionID]; len(owners) > 0 {
				ownerAgentID = owners[0].AgentID
			}
		}
		if ownerAgentID == "" {
			ownerAgentID = ceoID
		}

		meta, _ := json.Marshal(map[string]interface{}{
			"owner_position_id": strings.TrimSpace(item.OwnerPositionID),
			"source":            "work.approve",
		})
		child := &model.WorkItem{
			ID:              generateID("work"),
			CompanyID:       req.CompanyID,
			Title:           title,
			Description:     strings.TrimSpace(item.Description),
			Status:          string(model.WorkItemStatusApproved),
			CreatedByType:   "user",
			CreatedByID:     model.IMChairmanUserID,
			OwnerAgentID:    ownerAgentID,
			ReviewerAgentID: ceoID,
			ParentID:        parent.ID,
			Priority:        priority,
			DueAt:           dueAt,
			Meta:            string(meta),
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if err := s.store.CreateWorkItem(child); err != nil {
			writeServerError(w, err)
			return
		}
		childIDs = append(childIDs, child.ID)

		if ownerAgentID != "" && ceoID != "" {
			dm, err := s.store.EnsureDirectConversation(ceoID, ownerAgentID, "CEO <> Owner")
			if err == nil {
				_ = s.store.CreateIMMessage(&model.IMMessage{
					ID:             generateID("immsg"),
					ConversationID: dm.ID,
					SenderType:     "system",
					SenderID:       "work",
					Body:           fmt.Sprintf("New work item approved: %s", child.Title),
					Meta:           `{"kind":"work_dispatch","work_item_id":"` + child.ID + `"}`,
					Status:         "sent",
					CreatedAt:      time.Now(),
				})
			}
		}

		pos := positionByID[item.OwnerPositionID]
		if pos != nil {
			if dept := departmentByID[pos.DepartmentID]; dept != nil {
				deptConv, err := s.store.UpsertIMConversation("department", req.CompanyID, dept.ID, dept.Name+" Department")
				if err == nil {
					_ = s.store.AddIMMemberIfMissing(deptConv.ID, string(model.IMMemberTypeUser), model.IMChairmanUserID, "chairman")
					if ownerAgentID != "" {
						_ = s.store.AddIMMemberIfMissing(deptConv.ID, string(model.IMMemberTypeAgent), ownerAgentID, "owner")
					}
					if ceoID != "" {
						_ = s.store.AddIMMemberIfMissing(deptConv.ID, string(model.IMMemberTypeAgent), ceoID, "ceo")
					}
					_ = s.store.CreateIMMessage(&model.IMMessage{
						ID:             generateID("immsg"),
						ConversationID: deptConv.ID,
						SenderType:     "system",
						SenderID:       "work",
						Body:           fmt.Sprintf("Work dispatched to %s: %s", pos.Title, child.Title),
						Meta:           `{"kind":"work_dispatch","work_item_id":"` + child.ID + `"}`,
						Status:         "sent",
						CreatedAt:      time.Now(),
					})
				}
			}
			if strings.TrimSpace(pos.ReportsTo) != "" {
				managerPos := positionByID[pos.ReportsTo]
				if managerPos != nil {
					lineRef := managerPos.ID + ":" + pos.ID
					lineConv, err := s.store.UpsertIMConversation("line", req.CompanyID, lineRef, managerPos.Title+" <> "+pos.Title)
					if err == nil {
						_ = s.store.AddIMMemberIfMissing(lineConv.ID, string(model.IMMemberTypeUser), model.IMChairmanUserID, "chairman")
						if ceoID != "" {
							_ = s.store.AddIMMemberIfMissing(lineConv.ID, string(model.IMMemberTypeAgent), ceoID, "ceo")
						}
						if ownerAgentID != "" {
							_ = s.store.AddIMMemberIfMissing(lineConv.ID, string(model.IMMemberTypeAgent), ownerAgentID, "report")
						}
						for _, ma := range assignmentsByPosition[managerPos.ID] {
							_ = s.store.AddIMMemberIfMissing(lineConv.ID, string(model.IMMemberTypeAgent), ma.AgentID, "manager")
						}
						_ = s.store.CreateIMMessage(&model.IMMessage{
							ID:             generateID("immsg"),
							ConversationID: lineConv.ID,
							SenderType:     "system",
							SenderID:       "work",
							Body:           fmt.Sprintf("Line dispatch: %s", child.Title),
							Meta:           `{"kind":"work_dispatch","work_item_id":"` + child.ID + `"}`,
							Status:         "sent",
							CreatedAt:      time.Now(),
						})
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"parent_id": parent.ID,
		"item_ids":  childIDs,
	})
}

func (s *Server) handleWorkStatus(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	if !isValidWorkStatus(req.Status) {
		writeBadRequest(w, "invalid status")
		return
	}
	if _, err := s.store.GetWorkItem(id); err != nil {
		if err == sql.ErrNoRows {
			writeNotFound(w, "work item not found")
			return
		}
		writeServerError(w, err)
		return
	}
	if err := s.store.UpdateWorkItemStatus(id, req.Status); err != nil {
		writeServerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (s *Server) handleWorkUpdate(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	pr, ok := getIMPrincipal(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if pr.SenderType != string(model.IMMemberTypeAgent) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	var req struct {
		Text string `json:"text"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		writeBadRequest(w, "text required")
		return
	}
	item, err := s.store.GetWorkItem(id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeNotFound(w, "work item not found")
			return
		}
		writeServerError(w, err)
		return
	}
	if item.OwnerAgentID != pr.SenderID {
		writeError(w, http.StatusForbidden, "only owner can update")
		return
	}

	entry := map[string]interface{}{
		"at":          time.Now().Format(time.RFC3339),
		"sender_type": "agent",
		"sender_id":   pr.SenderID,
		"text":        req.Text,
	}
	if err := s.store.AppendWorkItemLog(id, entry); err != nil {
		writeServerError(w, err)
		return
	}

	reviewerID := strings.TrimSpace(item.ReviewerAgentID)
	if reviewerID == "" {
		if ceo, _, err := s.store.EnsureCEOAgentInstance(); err == nil {
			reviewerID = ceo.ID
		}
	}
	if reviewerID != "" {
		dm, err := s.store.EnsureDirectConversation(reviewerID, item.OwnerAgentID, "CEO <> Owner")
		if err == nil {
			_ = s.store.CreateIMMessage(&model.IMMessage{
				ID:             generateID("immsg"),
				ConversationID: dm.ID,
				SenderType:     string(model.IMMemberTypeAgent),
				SenderID:       pr.SenderID,
				Body:           req.Text,
				Meta:           `{"kind":"work_update","work_item_id":"` + item.ID + `"}`,
				Status:         "sent",
				CreatedAt:      time.Now(),
			})
		}
	}

	if item.CompanyID != "" {
		assignments, err := s.store.ListAssignments(item.CompanyID)
		if err == nil {
			var ownerAssignment *model.AgentAssignment
			for _, a := range assignments {
				if (a.Status == "" || a.Status == "active") && a.AgentID == item.OwnerAgentID {
					ownerAssignment = a
					break
				}
			}
			if ownerAssignment != nil {
				ownerPos, err := s.store.GetPosition(ownerAssignment.PositionID)
				if err == nil && strings.TrimSpace(ownerPos.ReportsTo) != "" {
					managerPos, err := s.store.GetPosition(ownerPos.ReportsTo)
					if err == nil {
						lineConv, err := s.store.UpsertIMConversation("line", item.CompanyID, managerPos.ID+":"+ownerPos.ID, managerPos.Title+" <> "+ownerPos.Title)
						if err == nil {
							_ = s.store.AddIMMemberIfMissing(lineConv.ID, string(model.IMMemberTypeUser), model.IMChairmanUserID, "chairman")
							_ = s.store.AddIMMemberIfMissing(lineConv.ID, string(model.IMMemberTypeAgent), item.OwnerAgentID, "report")
							_ = s.store.CreateIMMessage(&model.IMMessage{
								ID:             generateID("immsg"),
								ConversationID: lineConv.ID,
								SenderType:     string(model.IMMemberTypeAgent),
								SenderID:       item.OwnerAgentID,
								Body:           req.Text,
								Meta:           `{"kind":"work_update","work_item_id":"` + item.ID + `"}`,
								Status:         "sent",
								CreatedAt:      time.Now(),
							})
						}
					}
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) generateWorkPlan(goalText string, positions []*model.Position) (workPlanPayload, string) {
	apiKey := strings.TrimSpace(os.Getenv("PLANNER_OPENROUTER_API_KEY"))
	if apiKey == "" {
		return s.heuristicPlan(goalText, positions), "PLANNER_OPENROUTER_API_KEY not set; using heuristic fallback plan"
	}
	modelName := strings.TrimSpace(os.Getenv("PLANNER_MODEL"))
	if modelName == "" {
		modelName = "openrouter/minimax/minimax-m2.5"
	}

	type positionDescriptor struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	posList := make([]positionDescriptor, 0, len(positions))
	for _, p := range positions {
		posList = append(posList, positionDescriptor{ID: p.ID, Title: p.Title})
	}
	posJSON, _ := json.Marshal(posList)

	reqBody := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a COO planner. Return JSON only with shape: {\"items\":[{\"title\":\"\",\"description\":\"\",\"owner_position_id\":\"\",\"priority\":\"low|medium|high\",\"due_at\":\"RFC3339 or empty\"}]}. Keep 3-8 actionable items.",
			},
			{
				"role":    "user",
				"content": fmt.Sprintf("Goal: %s\nAvailable positions (id,title): %s", goalText, string(posJSON)),
			},
		},
		"temperature": 0.2,
	}
	buf, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequest(http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(buf))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("HTTP-Referer", "https://clawfleet.local")
	httpReq.Header.Set("X-Title", "Clawfleet Planner")

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil || resp == nil {
		return s.heuristicPlan(goalText, positions), "planner request failed; using heuristic fallback plan"
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return s.heuristicPlan(goalText, positions), "planner returned non-2xx; using heuristic fallback plan"
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil || len(parsed.Choices) == 0 {
		return s.heuristicPlan(goalText, positions), "planner response parse failed; using heuristic fallback plan"
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	content = extractJSONObject(content)
	var out workPlanPayload
	if err := json.Unmarshal([]byte(content), &out); err != nil || len(out.Items) == 0 {
		return s.heuristicPlan(goalText, positions), "planner output invalid; using heuristic fallback plan"
	}
	for i := range out.Items {
		out.Items[i].Priority = normalizePriority(out.Items[i].Priority)
	}
	return out, ""
}

func (s *Server) heuristicPlan(goalText string, positions []*model.Position) workPlanPayload {
	return workPlanPayload{
		Items: []workPlanItem{
			{
				Title:           "Execute goal",
				Description:     goalText,
				OwnerPositionID: findCEOPositionID(positions),
				Priority:        "high",
			},
		},
	}
}

func findCEOPositionID(positions []*model.Position) string {
	for _, p := range positions {
		title := strings.ToLower(strings.TrimSpace(p.Title))
		if strings.Contains(title, "ceo") || strings.Contains(title, "chief executive") {
			return p.ID
		}
	}
	for _, p := range positions {
		if p.Level == model.PositionLevelCSuite {
			return p.ID
		}
	}
	if len(positions) > 0 {
		return positions[0].ID
	}
	return ""
}

func normalizePriority(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "low":
		return "low"
	case "high":
		return "high"
	default:
		return "medium"
	}
}

func parseDueAt(v string) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func extractJSONObject(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return text
}

func isValidWorkStatus(v string) bool {
	switch v {
	case string(model.WorkItemStatusDraft),
		string(model.WorkItemStatusProposed),
		string(model.WorkItemStatusApproved),
		string(model.WorkItemStatusInProgress),
		string(model.WorkItemStatusDone),
		string(model.WorkItemStatusCancelled):
		return true
	default:
		return false
	}
}
