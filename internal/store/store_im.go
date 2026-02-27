package store

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/NeoTheCapt/clawfleet/internal/model"
	"github.com/NeoTheCapt/clawfleet/internal/util"
)

func parseSQLiteTime(v string) (time.Time, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, false
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02T15:04:05.999999999Z07:00",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, v); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

type IMOrgSyncResult struct {
	Conversations int `json:"conversations"`
	Created       int `json:"created"`
	Updated       int `json:"updated"`
	Deleted       int `json:"deleted"`
}

type IMConversationMemberView struct {
	ID               string    `json:"id"`
	ConversationID   string    `json:"conversation_id"`
	MemberType       string    `json:"member_type"`
	MemberID         string    `json:"member_id"`
	Role             string    `json:"role,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	AgentDisplayName string    `json:"agent_display_name,omitempty"`
	AgentStatus      string    `json:"agent_status,omitempty"`
}

func (s *Store) ListIMConversations(companyID string) ([]*model.IMConversation, error) {
	query := `SELECT c.id,c.type,c.company_id,c.ref_id,c.title,c.created_at,c.updated_at,
		COUNT(DISTINCT m.id) AS member_count,
		MAX(msg.created_at) AS last_message_at
	FROM im_conversations c
	LEFT JOIN im_members m ON m.conversation_id=c.id
	LEFT JOIN im_messages msg ON msg.conversation_id=c.id
	WHERE (?='' OR c.company_id=?)
	GROUP BY c.id,c.type,c.company_id,c.ref_id,c.title,c.created_at,c.updated_at
	ORDER BY COALESCE(MAX(msg.created_at), c.updated_at) DESC`
	rows, err := s.db.Query(query, companyID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*model.IMConversation, 0)
	for rows.Next() {
		var c model.IMConversation
		var last sql.NullString
		if err := rows.Scan(&c.ID, &c.Type, &c.CompanyID, &c.RefID, &c.Title, &c.CreatedAt, &c.UpdatedAt, &c.MemberCount, &last); err != nil {
			return nil, err
		}
		if last.Valid {
			if t, ok := parseSQLiteTime(last.String); ok {
				c.LastMessageAt = &t
			}
		}
		out = append(out, &c)
	}
	return out, nil
}

func (s *Store) ListIMConversationsForMember(memberType, memberID, companyID string) ([]*model.IMConversation, error) {
	query := `SELECT c.id,c.type,c.company_id,c.ref_id,c.title,c.created_at,c.updated_at,
		COUNT(DISTINCT allm.id) AS member_count,
		MAX(msg.created_at) AS last_message_at
	FROM im_conversations c
	INNER JOIN im_members own ON own.conversation_id=c.id AND own.member_type=? AND own.member_id=?
	LEFT JOIN im_members allm ON allm.conversation_id=c.id
	LEFT JOIN im_messages msg ON msg.conversation_id=c.id
	WHERE (?='' OR c.company_id=?)
	GROUP BY c.id,c.type,c.company_id,c.ref_id,c.title,c.created_at,c.updated_at
	ORDER BY COALESCE(MAX(msg.created_at), c.updated_at) DESC`
	rows, err := s.db.Query(query, memberType, memberID, companyID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*model.IMConversation, 0)
	for rows.Next() {
		var c model.IMConversation
		var last sql.NullString
		if err := rows.Scan(&c.ID, &c.Type, &c.CompanyID, &c.RefID, &c.Title, &c.CreatedAt, &c.UpdatedAt, &c.MemberCount, &last); err != nil {
			return nil, err
		}
		if last.Valid {
			if t, ok := parseSQLiteTime(last.String); ok {
				c.LastMessageAt = &t
			}
		}
		out = append(out, &c)
	}
	return out, nil
}

func (s *Store) GetIMConversation(id string) (*model.IMConversation, error) {
	var c model.IMConversation
	err := s.db.QueryRow(`SELECT id,type,company_id,ref_id,title,created_at,updated_at FROM im_conversations WHERE id=?`, id).
		Scan(&c.ID, &c.Type, &c.CompanyID, &c.RefID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) ListIMMessages(conversationID string) ([]*model.IMMessage, error) {
	return queryIMMessages(s.db, `SELECT id,conversation_id,sender_type,sender_id,body,meta,status,created_at FROM im_messages WHERE conversation_id=? ORDER BY created_at ASC`, conversationID)
}

func (s *Store) IsIMConversationMember(conversationID, memberType, memberID string) (bool, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM im_members WHERE conversation_id=? AND member_type=? AND member_id=?`, conversationID, memberType, memberID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) CreateIMMessage(msg *model.IMMessage) error {
	_, err := s.db.Exec(`INSERT INTO im_messages (id,conversation_id,sender_type,sender_id,body,meta,status,created_at) VALUES (?,?,?,?,?,?,?,?)`,
		msg.ID, msg.ConversationID, msg.SenderType, msg.SenderID, msg.Body, msg.Meta, msg.Status, msg.CreatedAt)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE im_conversations SET updated_at=? WHERE id=?`, time.Now(), msg.ConversationID)
	return err
}

func (s *Store) GetIMConversationByTypeRef(convType, companyID, refID string) (*model.IMConversation, error) {
	var c model.IMConversation
	err := s.db.QueryRow(`SELECT id,type,company_id,ref_id,title,created_at,updated_at FROM im_conversations WHERE type=? AND company_id=? AND ref_id=? LIMIT 1`, convType, companyID, refID).
		Scan(&c.ID, &c.Type, &c.CompanyID, &c.RefID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) UpsertIMConversation(convType, companyID, refID, title string) (*model.IMConversation, error) {
	existing, err := s.GetIMConversationByTypeRef(convType, companyID, refID)
	if err == nil {
		if existing.Title != title {
			if _, uerr := s.db.Exec(`UPDATE im_conversations SET title=?,updated_at=? WHERE id=?`, title, time.Now(), existing.ID); uerr != nil {
				return nil, uerr
			}
			existing.Title = title
		}
		return existing, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	now := time.Now()
	conv := &model.IMConversation{
		ID:        util.GenerateID("imc"),
		Type:      convType,
		CompanyID: companyID,
		RefID:     refID,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := s.db.Exec(`INSERT INTO im_conversations (id,type,company_id,ref_id,title,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
		conv.ID, conv.Type, conv.CompanyID, conv.RefID, conv.Title, conv.CreatedAt, conv.UpdatedAt); err != nil {
		return nil, err
	}
	return conv, nil
}

func (s *Store) AddIMMemberIfMissing(conversationID, memberType, memberID, role string) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO im_members (id,conversation_id,member_type,member_id,role,created_at) VALUES (?,?,?,?,?,?)`,
		util.GenerateID("imm"), conversationID, memberType, memberID, role, time.Now())
	return err
}

func (s *Store) RemoveIMMember(conversationID, memberType, memberID string) error {
	_, err := s.db.Exec(`DELETE FROM im_members WHERE conversation_id=? AND member_type=? AND member_id=?`, conversationID, memberType, memberID)
	return err
}

func (s *Store) ListIMConversationMembers(conversationID string) ([]*IMConversationMemberView, error) {
	rows, err := s.db.Query(`SELECT m.id,m.conversation_id,m.member_type,m.member_id,m.role,m.created_at,
		COALESCE(ai.name,''),COALESCE(ai.status,'')
		FROM im_members m
		LEFT JOIN agent_instances ai ON m.member_type='agent' AND ai.id=m.member_id
		WHERE m.conversation_id=?
		ORDER BY CASE m.member_type WHEN 'user' THEN 0 ELSE 1 END, m.created_at ASC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*IMConversationMemberView, 0)
	for rows.Next() {
		var m IMConversationMemberView
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.MemberType, &m.MemberID, &m.Role, &m.CreatedAt, &m.AgentDisplayName, &m.AgentStatus); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, nil
}

func (s *Store) ListIMConversationAgentMemberIDs(conversationID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT member_id FROM im_members WHERE conversation_id=? AND member_type='agent' ORDER BY created_at ASC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func (s *Store) EnsureDirectConversation(agentA, agentB, title string) (*model.IMConversation, error) {
	refID := "dm:" + CanonicalDMRefID(agentA, agentB)
	conv, err := s.UpsertIMConversation("direct", "", refID, title)
	if err != nil {
		return nil, err
	}
	if err := s.AddIMMemberIfMissing(conv.ID, string(model.IMMemberTypeAgent), agentA, "participant"); err != nil {
		return nil, err
	}
	if err := s.AddIMMemberIfMissing(conv.ID, string(model.IMMemberTypeAgent), agentB, "participant"); err != nil {
		return nil, err
	}
	return conv, nil
}

func CanonicalDMRefID(a, b string) string {
	ids := []string{a, b}
	sort.Strings(ids)
	return ids[0] + ":" + ids[1]
}

func (s *Store) AddAgentToAllIMConversations(agentID string, role string) (int, error) {
	rows, err := s.db.Query(`SELECT id FROM im_conversations`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var conversationID string
		if err := rows.Scan(&conversationID); err != nil {
			return count, err
		}
		if _, err := s.db.Exec(`INSERT OR IGNORE INTO im_members (id,conversation_id,member_type,member_id,role,created_at) VALUES (?,?,?,?,?,?)`,
			util.GenerateID("imm"), conversationID, string(model.IMMemberTypeAgent), agentID, role, time.Now()); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

func (s *Store) RecomputeIMOrg(companyID, ceoAgentID string) (*IMOrgSyncResult, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	type existingConv struct {
		ID    string
		Title string
	}
	existing := map[string]existingConv{}
	rows, err := tx.Query(`SELECT id,type,ref_id,title FROM im_conversations WHERE company_id=? AND type IN ('department','line')`, companyID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, t, refID, title string
		if err := rows.Scan(&id, &t, &refID, &title); err != nil {
			rows.Close()
			return nil, err
		}
		existing[t+"|"+refID] = existingConv{ID: id, Title: title}
	}
	rows.Close()

	type department struct {
		ID   string
		Name string
	}
	departments := make([]department, 0)
	rows, err = tx.Query(`SELECT id,name FROM departments WHERE company_id=? ORDER BY created_at ASC`, companyID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var d department
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			rows.Close()
			return nil, err
		}
		departments = append(departments, d)
	}
	rows.Close()

	type position struct {
		ID           string
		DepartmentID string
		Title        string
		ReportsTo    string
	}
	positions := make([]position, 0)
	positionsByID := map[string]position{}
	positionsByDepartment := map[string][]position{}
	rows, err = tx.Query(`SELECT p.id,p.department_id,p.title,p.reports_to FROM positions p JOIN departments d ON d.id=p.department_id WHERE d.company_id=? ORDER BY p.created_at ASC`, companyID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p position
		if err := rows.Scan(&p.ID, &p.DepartmentID, &p.Title, &p.ReportsTo); err != nil {
			rows.Close()
			return nil, err
		}
		positions = append(positions, p)
		positionsByID[p.ID] = p
		positionsByDepartment[p.DepartmentID] = append(positionsByDepartment[p.DepartmentID], p)
	}
	rows.Close()

	assignmentsByPosition := map[string][]string{}
	rows, err = tx.Query(`SELECT aa.position_id,aa.agent_id
		FROM agent_assignments aa
		JOIN positions p ON p.id=aa.position_id
		JOIN departments d ON d.id=p.department_id
		WHERE d.company_id=? AND (aa.status='active' OR aa.status='') ORDER BY aa.created_at ASC`, companyID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var positionID, agentID string
		if err := rows.Scan(&positionID, &agentID); err != nil {
			rows.Close()
			return nil, err
		}
		if strings.TrimSpace(positionID) == "" || strings.TrimSpace(agentID) == "" {
			continue
		}
		assignmentsByPosition[positionID] = append(assignmentsByPosition[positionID], agentID)
	}
	rows.Close()

	type desiredConversation struct {
		Type    string
		RefID   string
		Title   string
		Members map[string]string // memberKey -> role
	}
	desired := map[string]desiredConversation{}

	for _, d := range departments {
		m := map[string]string{string(model.IMMemberTypeUser) + ":" + model.IMChairmanUserID: "chairman"}
		if ceoAgentID != "" {
			m[string(model.IMMemberTypeAgent)+":"+ceoAgentID] = "ceo"
		}
		for _, p := range positionsByDepartment[d.ID] {
			for _, agentID := range assignmentsByPosition[p.ID] {
				m[string(model.IMMemberTypeAgent)+":"+agentID] = "member"
			}
		}
		key := "department|" + d.ID
		desired[key] = desiredConversation{
			Type:    "department",
			RefID:   d.ID,
			Title:   d.Name + " Department",
			Members: m,
		}
	}

	for _, p := range positions {
		if strings.TrimSpace(p.ReportsTo) == "" {
			continue
		}
		manager, ok := positionsByID[p.ReportsTo]
		if !ok {
			continue
		}
		m := map[string]string{string(model.IMMemberTypeUser) + ":" + model.IMChairmanUserID: "chairman"}
		if ceoAgentID != "" {
			m[string(model.IMMemberTypeAgent)+":"+ceoAgentID] = "ceo"
		}
		for _, agentID := range assignmentsByPosition[manager.ID] {
			m[string(model.IMMemberTypeAgent)+":"+agentID] = "manager"
		}
		for _, agentID := range assignmentsByPosition[p.ID] {
			m[string(model.IMMemberTypeAgent)+":"+agentID] = "report"
		}

		refID := manager.ID + ":" + p.ID
		key := "line|" + refID
		desired[key] = desiredConversation{
			Type:    "line",
			RefID:   refID,
			Title:   manager.Title + " <> " + p.Title,
			Members: m,
		}
	}

	result := &IMOrgSyncResult{Conversations: len(desired)}
	for key, dc := range desired {
		now := time.Now()
		convID := ""
		if ex, ok := existing[key]; ok {
			convID = ex.ID
			if ex.Title != dc.Title {
				if _, err := tx.Exec(`UPDATE im_conversations SET title=?,updated_at=? WHERE id=?`, dc.Title, now, convID); err != nil {
					return nil, err
				}
				result.Updated++
			}
			delete(existing, key)
		} else {
			convID = util.GenerateID("imc")
			if _, err := tx.Exec(`INSERT INTO im_conversations (id,type,company_id,ref_id,title,created_at,updated_at) VALUES (?,?,?,?,?,?,?)`,
				convID, dc.Type, companyID, dc.RefID, dc.Title, now, now); err != nil {
				return nil, err
			}
			result.Created++
		}

		if _, err := tx.Exec(`DELETE FROM im_members WHERE conversation_id=?`, convID); err != nil {
			return nil, err
		}
		for mk, role := range dc.Members {
			parts := strings.SplitN(mk, ":", 2)
			if len(parts) != 2 {
				continue
			}
			if _, err := tx.Exec(`INSERT INTO im_members (id,conversation_id,member_type,member_id,role,created_at) VALUES (?,?,?,?,?,?)`,
				util.GenerateID("imm"), convID, parts[0], parts[1], role, now); err != nil {
				return nil, err
			}
		}
	}

	for _, stale := range existing {
		if _, err := tx.Exec(`DELETE FROM im_outbox WHERE conversation_id=?`, stale.ID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM im_messages WHERE conversation_id=?`, stale.ID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM im_members WHERE conversation_id=?`, stale.ID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM im_conversations WHERE id=?`, stale.ID); err != nil {
			return nil, err
		}
		result.Deleted++
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func scanIMMessage(row scannable) (*model.IMMessage, error) {
	var m model.IMMessage
	if err := row.Scan(&m.ID, &m.ConversationID, &m.SenderType, &m.SenderID, &m.Body, &m.Meta, &m.Status, &m.CreatedAt); err != nil {
		return nil, err
	}
	return &m, nil
}

func queryIMMessages(db *sql.DB, query string, args ...interface{}) ([]*model.IMMessage, error) {
	return queryRows(db, query, args, func(r *sql.Rows) (*model.IMMessage, error) { return scanIMMessage(r) })
}

func (s *Store) DeleteIMConversation(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.Exec(`DELETE FROM im_outbox WHERE conversation_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM im_messages WHERE conversation_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM im_members WHERE conversation_id=?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM im_conversations WHERE id=?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) GetAgentInstanceByRole(role string) (*model.AgentInstance, error) {
	row := s.db.QueryRow(`SELECT id,name,company_id,node_id,agent_type,role,status,container_id,deploy_mode,config,created_at,updated_at FROM agent_instances WHERE role=? ORDER BY created_at ASC LIMIT 1`, role)
	a, err := scanAgent(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return a, nil
}

func (s *Store) EnsureCEOAgentInstance() (*model.AgentInstance, bool, error) {
	a, err := s.GetAgentInstance(model.IMCEOAgentID)
	if err == nil {
		return a, false, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}

	nodeID := ""
	nodes, err := s.ListNodes()
	if err == nil && len(nodes) > 0 {
		nodeID = nodes[0].ID
	}
	now := time.Now()
	a = &model.AgentInstance{
		ID:         model.IMCEOAgentID,
		Name:       "CEO",
		CompanyID:  "",
		NodeID:     nodeID,
		AgentType:  model.AgentTypeOpenClaw,
		Role:       "ceo",
		Status:     model.AgentStatusStopped,
		DeployMode: "docker",
		Config: model.AgentConfig{
			PersonaName:  "CEO",
			SystemPrompt: "You are the CEO agent identity in internal IM.",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.CreateAgentInstance(a); err != nil {
		return nil, false, fmt.Errorf("create CEO agent: %w", err)
	}
	return a, true, nil
}
