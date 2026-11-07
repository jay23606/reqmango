package service

import (
	"testing"
	"time"

	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
)

func TestTimeTrackCreate_AddsToSummary(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.TimeTrack{}, &model.Issue{}, &model.State{}, &model.IssueType{},
		&model.IssueTypeImport{}, &model.IssueActivity{}, &model.IssueAssignee{},
		&model.IssueLabel{}, &model.Label{}, &model.Webhook{},
		&model.IssueCycle{}, &model.Cycle{}, &model.ModuleIssue{}, &model.Module{},
	))
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	issueSvc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	iss, err := issueSvc.Create(&request.IssueCreateRequest{Name: "timed"}, projID, wsID, userID)
	require.NoError(t, err)

	ttSvc := NewTimeTrackService(db)
	ended := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	desc := "manual"
	entry, err := ttSvc.Create(iss.ID, userID, &request.TimeTrackCreateRequest{
		Duration:    3600,
		Description: &desc,
		EndedAt:     &ended,
	})
	require.NoError(t, err)
	require.Equal(t, int64(3600), entry.Duration)
	require.NotNil(t, entry.EndedAt)

	sum, err := ttSvc.Summary(iss.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3600), sum.TotalSeconds)
	require.Equal(t, 1, sum.EntryCount)
	require.InDelta(t, 1.0, sum.TotalHours, 0.001)
}
