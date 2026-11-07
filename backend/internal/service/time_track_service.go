package service

import (
	"time"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/dto/response"
	"github.com/reqmango/backend/internal/model"
	"gorm.io/gorm"
)

type TimeTrackService struct{ db *gorm.DB }

func NewTimeTrackService(db *gorm.DB) *TimeTrackService { return &TimeTrackService{db: db} }

// Create logs a completed time entry manually. Duration is seconds.
func (s *TimeTrackService) Create(issueID, userID uint64, req *request.TimeTrackCreateRequest) (*response.TimeTrackResponse, error) {
	if req.Duration < 1 {
		return nil, common.BadRequest("duration must be at least 1 second")
	}
	ended := time.Now()
	if req.EndedAt != nil {
		ended = *req.EndedAt
	}
	started := ended.Add(-time.Duration(req.Duration) * time.Second)
	t := &model.TimeTrack{
		IssueID:     issueID,
		UserID:      userID,
		Description: req.Description,
		StartedAt:   started,
		EndedAt:     &ended,
		Duration:    req.Duration,
	}
	if err := s.db.Create(t).Error; err != nil {
		return nil, common.Internal("Failed to create time entry")
	}
	return toResponse(t), nil
}

// Start begins a timer for an issue.
func (s *TimeTrackService) Start(issueID, userID uint64, req *request.TimeTrackStartRequest) (*response.TimeTrackResponse, error) {
	// Stop any running timer for this user
	s.db.Model(&model.TimeTrack{}).
		Where("user_id = ? AND ended_at IS NULL", userID).
		Updates(map[string]interface{}{"ended_at": gorm.Expr("NOW()"), "duration": gorm.Expr("EXTRACT(EPOCH FROM NOW() - started_at)")})

	t := &model.TimeTrack{
		IssueID:     issueID,
		UserID:      userID,
		Description: req.Description,
		StartedAt:   time.Now(),
	}
	if err := s.db.Create(t).Error; err != nil {
		return nil, common.Internal("Failed to start timer")
	}
	return toResponse(t), nil
}

// Stop stops the running timer for a user on an issue.
func (s *TimeTrackService) Stop(issueID, userID uint64) (*response.TimeTrackResponse, error) {
	var t model.TimeTrack
	if err := s.db.Where("issue_id = ? AND user_id = ? AND ended_at IS NULL", issueID, userID).First(&t).Error; err != nil {
		return nil, common.NotFound("No running timer found")
	}
	now := time.Now()
	t.EndedAt = &now
	t.Duration = int64(now.Sub(t.StartedAt).Seconds())
	if err := s.db.Save(&t).Error; err != nil {
		return nil, common.Internal("Failed to stop timer")
	}
	return toResponse(&t), nil
}

// List returns time entries for an issue.
func (s *TimeTrackService) List(issueID uint64) ([]response.TimeTrackResponse, error) {
	var entries []model.TimeTrack
	if err := s.db.Where("issue_id = ?", issueID).Order("started_at DESC").Find(&entries).Error; err != nil {
		return nil, common.Internal("Failed to list time entries")
	}
	result := make([]response.TimeTrackResponse, len(entries))
	for i, e := range entries {
		result[i] = *toResponse(&e)
	}
	return result, nil
}

// Summary returns aggregated time stats for an issue.
func (s *TimeTrackService) Summary(issueID uint64) (*response.TimeTrackSummary, error) {
	var total struct {
		Seconds int64
		Count   int
	}
	s.db.Model(&model.TimeTrack{}).
		Select("COALESCE(SUM(duration),0) as seconds, COUNT(*) as count").
		Where("issue_id = ? AND ended_at IS NOT NULL", issueID).Scan(&total)

	var running int64
	s.db.Model(&model.TimeTrack{}).Where("issue_id = ? AND ended_at IS NULL", issueID).Count(&running)

	return &response.TimeTrackSummary{
		IssueID:      issueID,
		TotalSeconds: total.Seconds,
		TotalHours:   float64(total.Seconds) / 3600,
		EntryCount:   total.Count,
		IsRunning:    running > 0,
	}, nil
}

// Delete removes a time entry.
func (s *TimeTrackService) Delete(id, userID uint64) error {
	r := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&model.TimeTrack{})
	if r.RowsAffected == 0 {
		return common.NotFound("Time entry not found")
	}
	return r.Error
}

func toResponse(t *model.TimeTrack) *response.TimeTrackResponse {
	return &response.TimeTrackResponse{
		ID: t.ID, IssueID: t.IssueID, UserID: t.UserID,
		Description: t.Description, StartedAt: t.StartedAt,
		EndedAt: t.EndedAt, Duration: t.Duration, CreatedAt: t.CreatedAt,
	}
}

// ProjectSummary returns aggregated time tracking stats for a project.
func (s *TimeTrackService) ProjectSummary(projectID uint64) (map[string]interface{}, error) {
	var total struct {
		Seconds int64
		Count   int
	}
	s.db.Model(&model.TimeTrack{}).
		Select("COALESCE(SUM(t.duration),0) as seconds, COUNT(*) as count").
		Joins("JOIN issues i ON i.id = t.issue_id").
		Where("i.project_id = ? AND t.ended_at IS NOT NULL", projectID).Scan(&total)

	// Per-user breakdown
	type userStats struct {
		UserID   uint64 `json:"user_id"`
		Username string `json:"username"`
		Seconds  int64  `json:"seconds"`
		Count    int    `json:"count"`
	}
	var userBreakdown []userStats
	s.db.Model(&model.TimeTrack{}).
		Select("t.user_id, u.username, COALESCE(SUM(t.duration),0) as seconds, COUNT(*) as count").
		Joins("JOIN issues i ON i.id = t.issue_id").
		Joins("JOIN users u ON u.id = t.user_id").
		Where("i.project_id = ? AND t.ended_at IS NOT NULL", projectID).
		Group("t.user_id, u.username").Order("seconds DESC").Scan(&userBreakdown)

	return map[string]interface{}{
		"project_id":    projectID,
		"total_hours":   float64(total.Seconds) / 3600,
		"total_seconds": total.Seconds,
		"entry_count":   total.Count,
		"by_user":       userBreakdown,
	}, nil
}
