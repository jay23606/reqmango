package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/dto/response"
	"github.com/reqmango/backend/internal/model"
	"gorm.io/gorm"
)

type ApprovalService struct {
	db       *gorm.DB
	notifier *NotificationService
}

func NewApprovalService(db *gorm.DB, notifier *NotificationService) *ApprovalService {
	return &ApprovalService{db: db, notifier: notifier}
}

// Create submits a new approval request.
func (s *ApprovalService) Create(issueID, requesterID, transitionID uint64, requestNote string) (*model.Approval, error) {
	// 1. Check no other pending approval exists on this issue
	var pendingCount int64
	if err := s.db.Model(&model.Approval{}).Where("issue_id = ? AND status = ?", issueID, "pending").Count(&pendingCount).Error; err != nil {
		return nil, common.Internal("Failed to check pending approvals")
	}
	if pendingCount > 0 {
		return nil, common.BadRequest("issue_already_pending_approval")
	}

	// 2. Get the transition
	var transition model.StateTransition
	if err := s.db.First(&transition, transitionID).Error; err != nil {
		return nil, common.NotFound("Transition not found")
	}
	if transition.RuleType != "approval" {
		return nil, common.BadRequest("transition_not_approval_type")
	}

	// 3. Get the issue
	var issue model.Issue
	if err := s.db.First(&issue, issueID).Error; err != nil {
		return nil, common.NotFound("Issue not found")
	}

	// 4. Resolve approve/reject target states
	// Approve: go to target_state_id (main target state)
	approveTargetID := transition.TargetStateID

	// Reject: go to reject_target_state_id if specified, otherwise stay at source_state_id
	rejectTargetID := transition.SourceStateID
	if transition.RejectTargetStateID != nil {
		rejectTargetID = *transition.RejectTargetStateID
	}

	// 5. Normalize approver_ids to JSON array string
	approverIDsJSON := normalizeApproverIDs(transition.ApproverIDs)

	// 6. Create approval in a transaction
	approval := model.Approval{
		IssueID:              issueID,
		WorkflowID:           transition.WorkflowID,
		TransitionID:         transitionID,
		ProjectID:            issue.ProjectID,
		WorkspaceID:          issue.WorkspaceID,
		RequesterID:          requesterID,
		RequestNote:          requestNote,
		SourceStateID:        issue.StateID,
		ApproveTargetStateID: approveTargetID,
		RejectTargetStateID:  rejectTargetID,
		ApproverIDs:          approverIDsJSON,
		Status:               "pending",
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&approval).Error; err != nil {
			return err
		}
		// Update issue
		pendingStatus := "pending"
		if err := tx.Model(&model.Issue{}).Where("id = ?", issueID).
			Updates(map[string]interface{}{
				"approval_status":    pendingStatus,
				"active_approval_id": approval.ID,
			}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, common.Internal("Failed to create approval")
	}

	// 7. Notify approvers (best-effort)
	if s.notifier != nil {
		approverIDs := filterRecipientsForSender(parseUint64Array(approverIDsJSON), &approval.RequesterID)
		if len(approverIDs) > 0 {
			title := "Approval request"
			message := fmt.Sprintf("Issue #%d requires your approval", issueID)
			_ = s.notifier.TriggerNotificationsBulk(s.db, "approval_requested", title, message, approverIDs, &approval.RequesterID, &approval.ProjectID, &approval.IssueID)
		}
	}

	// 8. Record issue activity (best-effort)
	s.recordActivity(issueID, requesterID, "approval_submitted", nil, nil, nil, strPtrOrNil(approverIDsJSON))

	return &approval, nil
}

// List returns approvals matching the filter.
func (s *ApprovalService) List(req request.ApprovalListQuery, workspaceID uint64) ([]response.ApprovalResponse, error) {
	query := s.db.Model(&model.Approval{}).Where("workspace_id = ?", workspaceID)
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.ProjectID != 0 {
		query = query.Where("project_id = ?", req.ProjectID)
	}
	if req.ApproverID != 0 {
		query = query.Where("approver_ids @> ?::jsonb", fmt.Sprintf("[%d]", req.ApproverID))
	}
	query = query.Order("created_at DESC")

	var approvals []model.Approval
	if err := query.Find(&approvals).Error; err != nil {
		return nil, common.Internal("Failed to list approvals")
	}

	res := make([]response.ApprovalResponse, 0, len(approvals))
	for _, a := range approvals {
		res = append(res, s.toResponse(a))
	}
	return res, nil
}

// Get returns an approval by ID with related data.
func (s *ApprovalService) Get(id uint64) (*response.ApprovalResponse, error) {
	var approval model.Approval
	if err := s.db.First(&approval, id).Error; err != nil {
		return nil, common.NotFound("Approval not found")
	}
	resp := s.toResponse(approval)

	var records []model.ApprovalRecord
	s.db.Where("approval_id = ?", id).Order("decided_at DESC").Find(&records)
	for _, r := range records {
		resp.Records = append(resp.Records, response.ApprovalRecordResponse{
			ID:         r.ID,
			ApproverID: r.ApproverID,
			Decision:   r.Decision,
			Note:       r.Note,
			DecidedAt:  r.DecidedAt,
		})
	}
	return &resp, nil
}

// Decide records an approver's decision and updates the issue state.
func (s *ApprovalService) Decide(approvalID, approverID uint64, decision, note string) (*model.Approval, error) {
	if decision != "approved" && decision != "rejected" {
		return nil, common.BadRequest("invalid_decision")
	}

	// 1. Load approval
	var approval model.Approval
	if err := s.db.First(&approval, approvalID).Error; err != nil {
		return nil, common.NotFound("Approval not found")
	}
	if approval.Status != "pending" {
		return nil, common.BadRequest("approval_already_decided")
	}

	// 2. Validate approver
	approverIDs := parseUint64Array(approval.ApproverIDs)
	if !containsUint64(approverIDs, approverID) {
		return nil, common.Forbidden("not_an_approver")
	}

	// 3. Determine target state
	var targetStateID uint64
	if decision == "approved" {
		targetStateID = approval.ApproveTargetStateID
	} else {
		targetStateID = approval.RejectTargetStateID
	}

	// 4. Transaction: create record, update approval, update issue
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Create approval_record
		record := model.ApprovalRecord{
			ApprovalID: approvalID,
			ApproverID: approverID,
			Decision:   decision,
			Note:       note,
			DecidedAt:  time.Now(),
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}

		// Update approval
		now := time.Now()
		updates := map[string]interface{}{
			"status":        decision,
			"decided_by":    approverID,
			"decided_at":    now,
			"decision_note": note,
		}
		if err := tx.Model(&model.Approval{}).Where("id = ? AND status = ?", approvalID, "pending").
			Updates(updates).Error; err != nil {
			return err
		}

		// Update issue: state_id + approval_status
		if err := tx.Model(&model.Issue{}).Where("id = ?", approval.IssueID).
			Updates(map[string]interface{}{
				"state_id":           targetStateID,
				"approval_status":    decision,
				"active_approval_id": nil,
			}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, common.Internal("Failed to decide approval")
	}

	// Reload
	s.db.First(&approval, approvalID)

	// Notify requester (best-effort)
	if s.notifier != nil && approval.RequesterID != 0 {
		title := "Approval " + decision
		message := fmt.Sprintf("Your approval request on issue #%d has been %s", approval.IssueID, decision)
		actorID := approverID
		_ = s.notifier.TriggerNotification(s.db, "approval_decided", title, message, approval.RequesterID, &actorID, &approval.ProjectID, &approval.IssueID)
	}

	// Record issue activity (best-effort)
	s.recordActivity(approval.IssueID, approverID, "approval_"+decision, strPtr("state"), strPtr(fmt.Sprintf("%d", approval.SourceStateID)), strPtr(fmt.Sprintf("%d", targetStateID)), strPtrOrNil(note))

	return &approval, nil
}

// Cancel cancels a pending approval. Only the requester can cancel.
func (s *ApprovalService) Cancel(approvalID, userID uint64) (*model.Approval, error) {
	var approval model.Approval
	if err := s.db.First(&approval, approvalID).Error; err != nil {
		return nil, common.NotFound("Approval not found")
	}
	if approval.Status != "pending" {
		return nil, common.BadRequest("approval_not_pending")
	}
	if approval.RequesterID != userID {
		return nil, common.Forbidden("not_requester")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Approval{}).Where("id = ?", approvalID).
			Update("status", "cancelled").Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Issue{}).Where("id = ?", approval.IssueID).
			Updates(map[string]interface{}{
				"approval_status":    nil,
				"active_approval_id": nil,
			}).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, common.Internal("Failed to cancel approval")
	}

	s.db.First(&approval, approvalID)

	// Record issue activity (best-effort)
	s.recordActivity(approval.IssueID, userID, "approval_cancelled", nil, nil, nil, nil)

	return &approval, nil
}

// CountPending returns the number of pending approvals for a user (as approver) in a workspace.
func (s *ApprovalService) CountPending(workspaceID, userID uint64) (int64, error) {
	var count int64
	err := s.db.Model(&model.Approval{}).
		Where("workspace_id = ? AND status = ? AND approver_ids @> ?::jsonb",
			workspaceID, "pending", fmt.Sprintf("[%d]", userID)).
		Count(&count).Error
	if err != nil {
		return 0, common.Internal("Failed to count pending approvals")
	}
	return count, nil
}

// toResponse converts an Approval model to a response DTO (without records).
func (s *ApprovalService) toResponse(a model.Approval) response.ApprovalResponse {
	resp := response.ApprovalResponse{
		ID:                   a.ID,
		IssueID:              a.IssueID,
		WorkflowID:           a.WorkflowID,
		TransitionID:         a.TransitionID,
		ProjectID:            a.ProjectID,
		WorkspaceID:          a.WorkspaceID,
		RequesterID:          a.RequesterID,
		RequestNote:          a.RequestNote,
		SourceStateID:        a.SourceStateID,
		ApproveTargetStateID: a.ApproveTargetStateID,
		RejectTargetStateID:  a.RejectTargetStateID,
		ApproverIDs:          parseUint64Array(a.ApproverIDs),
		Status:               a.Status,
		DecidedBy:            a.DecidedBy,
		DecidedAt:            a.DecidedAt,
		DecisionNote:         a.DecisionNote,
		CreatedAt:            a.CreatedAt,
		Records:              []response.ApprovalRecordResponse{},
	}

	// Enrich with names (best-effort, ignore errors)
	var requester model.User
	if err := s.db.Select("id, display_name").First(&requester, a.RequesterID).Error; err == nil {
		resp.RequesterName = requester.DisplayName
	}
	if a.DecidedBy != nil {
		var decider model.User
		if err := s.db.Select("id, display_name").First(&decider, *a.DecidedBy).Error; err == nil {
			resp.DecidedByName = decider.DisplayName
		}
	}
	var issue model.Issue
	if err := s.db.First(&issue, a.IssueID).Error; err == nil {
		resp.IssueTitle = issue.Name
	}
	// Load project (with identifier) to build IssueKey like "CORE-52" and ProjectName
	var project model.Project
	if err := s.db.Select("id, name, identifier").First(&project, a.ProjectID).Error; err == nil {
		resp.ProjectName = project.Name
		if issue.ID != 0 {
			resp.IssueKey = fmt.Sprintf("%s-%d", project.Identifier, issue.SequenceID)
		}
	}
	var srcState model.State
	if err := s.db.Select("id, name").First(&srcState, a.SourceStateID).Error; err == nil {
		resp.SourceStateName = srcState.Name
	}
	var approveState model.State
	if err := s.db.Select("id, name").First(&approveState, a.ApproveTargetStateID).Error; err == nil {
		resp.ApproveStateName = approveState.Name
	}
	var rejectState model.State
	if err := s.db.Select("id, name").First(&rejectState, a.RejectTargetStateID).Error; err == nil {
		resp.RejectStateName = rejectState.Name
	}
	// Approver names
	for _, id := range resp.ApproverIDs {
		var u model.User
		if err := s.db.Select("id, display_name").First(&u, id).Error; err == nil {
			resp.ApproverNames = append(resp.ApproverNames, u.DisplayName)
		} else {
			resp.ApproverNames = append(resp.ApproverNames, fmt.Sprintf("#%d", id))
		}
	}
	return resp
}

// normalizeApproverIDs ensures approver_ids is stored as a JSON array string.
// Accepts both "[1,2,3]" and "1,2,3" formats.
func normalizeApproverIDs(s *string) string {
	if s == nil || *s == "" {
		return "[]"
	}
	str := *s
	// Already JSON array
	if len(str) > 0 && str[0] == '[' {
		return str
	}
	// Comma-separated -> JSON array
	ids := parseUint64Array(str)
	b, _ := json.Marshal(ids)
	return string(b)
}

func parseUint64Array(s string) []uint64 {
	if s == "" {
		return nil
	}
	// Try JSON first
	var ids []uint64
	if err := json.Unmarshal([]byte(s), &ids); err == nil {
		return ids
	}
	// Fallback: comma-separated
	var result []uint64
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var id uint64
		if _, err := fmt.Sscanf(part, "%d", &id); err == nil {
			result = append(result, id)
		}
	}
	return result
}

func containsUint64(arr []uint64, v uint64) bool {
	for _, x := range arr {
		if x == v {
			return true
		}
	}
	return false
}

// recordActivity creates an IssueActivity record (best-effort, ignores errors).
func (s *ApprovalService) recordActivity(issueID, actorID uint64, verb string, field, oldValue, newValue, comment *string) {
	activity := &model.IssueActivity{
		IssueID:  &issueID,
		Verb:     verb,
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
		Comment:  comment,
		ActorID:  &actorID,
	}
	s.db.Create(activity)
}

func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
