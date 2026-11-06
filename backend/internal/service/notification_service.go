package service

import (
	"fmt"
	"time"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/dto/response"
	"github.com/reqmango/backend/internal/model"
	"gorm.io/gorm"
)

// NotificationService handles notification business logic.
type NotificationService struct {
	db *gorm.DB
}

// NewNotificationService creates a new NotificationService.
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

// List returns notifications for a recipient, with optional unread filter.
func (s *NotificationService) List(recipientID uint64, unreadOnly bool, limit, offset int) ([]response.NotificationResponse, error) {
	var notifications []model.Notification
	q := s.db.Where("recipient_id = ?", recipientID)
	if unreadOnly {
		q = q.Where("is_read = ?", false)
	}
	if limit <= 0 {
		limit = 50
	}
	if err := q.Order("created_at DESC").Limit(limit).Offset(offset).Find(&notifications).Error; err != nil {
		return nil, common.Internal("Failed to fetch notifications")
	}

	responses := make([]response.NotificationResponse, len(notifications))
	for i, n := range notifications {
		responses[i] = notificationToResponse(&n)
	}
	return responses, nil
}

// Get returns a single notification by ID, scoped to the recipient.
func (s *NotificationService) Get(id, recipientID uint64) (*response.NotificationResponse, error) {
	var n model.Notification
	if err := s.db.Where("id = ? AND recipient_id = ?", id, recipientID).First(&n).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, common.NotificationNotFound()
		}
		return nil, common.Internal("Failed to fetch notification")
	}
	resp := notificationToResponse(&n)
	return &resp, nil
}

// Create creates a single notification.
func (s *NotificationService) Create(req *request.NotificationCreateRequest) (*response.NotificationResponse, error) {
	n := buildNotification(req.Title, req.Message, req.Type, req.Priority, req.ActionURL, req.RecipientID, req.SenderID, req.ProjectID, req.IssueID)
	if err := s.db.Create(n).Error; err != nil {
		return nil, common.Internal("Failed to create notification")
	}
	resp := notificationToResponse(n)
	return &resp, nil
}

// CreateBulk creates notifications for multiple recipients.
func (s *NotificationService) CreateBulk(req *request.NotificationBulkCreateRequest) ([]response.NotificationResponse, error) {
	notifications := make([]*model.Notification, len(req.RecipientIDs))
	for i, rid := range req.RecipientIDs {
		notifications[i] = buildNotification(req.Title, req.Message, req.Type, req.Priority, req.ActionURL, rid, req.SenderID, req.ProjectID, req.IssueID)
	}
	if err := s.db.Create(&notifications).Error; err != nil {
		return nil, common.Internal("Failed to create notifications")
	}

	responses := make([]response.NotificationResponse, len(notifications))
	for i, n := range notifications {
		responses[i] = notificationToResponse(n)
	}
	return responses, nil
}

// MarkRead marks a notification as read.
func (s *NotificationService) MarkRead(id, recipientID uint64) (*response.NotificationResponse, error) {
	var n model.Notification
	if err := s.db.Where("id = ? AND recipient_id = ?", id, recipientID).First(&n).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, common.NotificationNotFound()
		}
		return nil, common.Internal("Failed to fetch notification")
	}
	now := gorm.Expr("NOW()")
	if err := s.db.Model(&n).Updates(map[string]interface{}{"is_read": true, "read_at": now}).Error; err != nil {
		return nil, common.Internal("Failed to mark as read")
	}
	n.IsRead = true
	resp := notificationToResponse(&n)
	return &resp, nil
}

// MarkAllRead marks all notifications as read for a recipient.
func (s *NotificationService) MarkAllRead(recipientID uint64) (*response.MarkAllReadResponse, error) {
	result := s.db.Model(&model.Notification{}).
		Where("recipient_id = ? AND is_read = ?", recipientID, false).
		Updates(map[string]interface{}{"is_read": true, "read_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return nil, common.Internal("Failed to mark all as read")
	}
	return &response.MarkAllReadResponse{MarkedCount: result.RowsAffected}, nil
}

// GetSummary returns notification counts for a recipient.
func (s *NotificationService) GetSummary(recipientID uint64) (*response.NotificationSummaryResponse, error) {
	var total, unread int64
	if err := s.db.Model(&model.Notification{}).Where("recipient_id = ?", recipientID).Count(&total).Error; err != nil {
		return nil, common.Internal("Failed to count notifications")
	}
	if err := s.db.Model(&model.Notification{}).Where("recipient_id = ? AND is_read = ?", recipientID, false).Count(&unread).Error; err != nil {
		return nil, common.Internal("Failed to count unread notifications")
	}

	// Unread by type
	type typeCount struct {
		Type  string
		Count int
	}
	var typeCounts []typeCount
	s.db.Model(&model.Notification{}).
		Select("type, COUNT(*) as count").
		Where("recipient_id = ? AND is_read = ?", recipientID, false).
		Group("type").Scan(&typeCounts)

	unreadByType := make(map[string]int)
	for _, tc := range typeCounts {
		unreadByType[tc.Type] = tc.Count
	}

	return &response.NotificationSummaryResponse{
		Total:        int(total),
		Unread:       int(unread),
		UnreadByType: unreadByType,
	}, nil
}

// Delete deletes a notification.
func (s *NotificationService) Delete(id, recipientID uint64) error {
	result := s.db.Where("id = ? AND recipient_id = ?", id, recipientID).Delete(&model.Notification{})
	if result.Error != nil {
		return common.Internal("Failed to delete notification")
	}
	if result.RowsAffected == 0 {
		return common.NotFound("Notification not found")
	}
	return nil
}

// ==================== Helpers ====================

func buildNotification(title, message, nType, priority string, actionURL *string, recipientID uint64, senderID, projectID, issueID *uint64) *model.Notification {
	if nType == "" {
		nType = "info"
	}
	if priority == "" {
		priority = "medium"
	}
	return &model.Notification{
		Title:       title,
		Message:     message,
		Type:        nType,
		Priority:    priority,
		ActionURL:   actionURL,
		RecipientID: recipientID,
		SenderID:    senderID,
		ProjectID:   projectID,
		IssueID:     issueID,
	}
}

func notificationToResponse(n *model.Notification) response.NotificationResponse {
	return response.NotificationResponse{
		ID:          n.ID,
		Title:       n.Title,
		Message:     n.Message,
		Type:        n.Type,
		Priority:    n.Priority,
		IsRead:      n.IsRead,
		ReadAt:      n.ReadAt,
		ActionURL:   n.ActionURL,
		RecipientID: n.RecipientID,
		SenderID:    n.SenderID,
		ProjectID:   n.ProjectID,
		IssueID:     n.IssueID,
		CreatedAt:   n.CreatedAt,
		UpdatedAt:   n.UpdatedAt,
	}
}

// TriggerNotification creates a notification for a specific event.
// This is used internally to auto-trigger notifications on business events.
func (s *NotificationService) TriggerNotification(db *gorm.DB, event, title, message string, recipientID uint64, senderID, projectID, issueID *uint64) error {
	if skipSelfNotify(recipientID, senderID) {
		return nil
	}
	nType := "info"
	priority := "medium"

	switch event {
	case "issue_assigned", "issue_mentioned":
		nType = "assignment"
		priority = "high"
	case "issue_state_changed":
		nType = "status"
	case "issue_commented":
		nType = "comment"
	case "issue_created", "issue_updated":
		nType = "info"
	case "issue_due_soon":
		nType = "reminder"
		priority = "high"
	case "cycle_created":
		nType = "info"
	}

	n := &model.Notification{
		Title:       title,
		Message:     message,
		Type:        nType,
		Priority:    priority,
		RecipientID: recipientID,
		SenderID:    senderID,
		ProjectID:   projectID,
		IssueID:     issueID,
	}

	return db.Create(n).Error
}

// TriggerNotificationsBulk creates notifications for multiple recipients.
func (s *NotificationService) TriggerNotificationsBulk(db *gorm.DB, event, title, message string, recipientIDs []uint64, senderID, projectID, issueID *uint64) error {
	recipientIDs = filterRecipientsForSender(recipientIDs, senderID)
	if len(recipientIDs) == 0 {
		return nil
	}
	nType := "info"
	priority := "medium"

	switch event {
	case "issue_assigned", "issue_mentioned":
		nType = "assignment"
		priority = "high"
	case "issue_state_changed":
		nType = "status"
	case "issue_commented":
		nType = "comment"
	case "issue_created", "issue_updated":
		nType = "info"
	case "issue_due_soon":
		nType = "reminder"
		priority = "high"
	case "cycle_created":
		nType = "info"
	}

	notifications := make([]*model.Notification, len(recipientIDs))
	for i, rid := range recipientIDs {
		notifications[i] = &model.Notification{
			Title:       title,
			Message:     message,
			Type:        nType,
			Priority:    priority,
			RecipientID: rid,
			SenderID:    senderID,
			ProjectID:   projectID,
			IssueID:     issueID,
		}
	}

	return db.Create(&notifications).Error
}

// CheckDueDateReminders finds issues due within 24 hours and sends reminder notifications.
func (s *NotificationService) CheckDueDateReminders() error {
	var issues []model.Issue
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)

	s.db.Where("target_date <= ? AND target_date > ? AND archived_at IS NULL", tomorrow, now).
		Preload("AssigneeLinks.User").Find(&issues)

	for _, issue := range issues {
		// Check if reminder already sent (by issue_id and type = reminder)
		var count int64
		s.db.Model(&model.Notification{}).
			Where("issue_id = ? AND type = ? AND title LIKE ?", issue.ID, "reminder", "%到期%").
			Count(&count)
		if count > 0 {
			continue
		}

		for _, link := range issue.AssigneeLinks {
			projID := issue.ProjectID
			issueID := issue.ID
			_ = s.TriggerNotification(s.db,
				"issue_due_soon",
				fmt.Sprintf("工作项即将到期: %s", issue.Name),
				"工作项将在24小时内到期",
				link.UserID,
				nil,
				&projID,
				&issueID,
			)
		}
	}
	return nil
}
