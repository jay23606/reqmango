package service

import (
	"testing"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
)

func estimateTestMigrate(t *testing.T, db interface{ AutoMigrate(...interface{}) error }) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&model.Issue{}, &model.IssueType{}, &model.State{},
		&model.IssueActivity{}, &model.IssueAssignee{}, &model.IssueLabel{}, &model.Label{},
		&model.IssueTypeImport{}, &model.Webhook{}, &model.WorkspaceMember{},
		&model.IssueCycle{}, &model.Cycle{}, &model.ModuleIssue{}, &model.Module{},
		&model.EstimatePoint{},
	))
}

func TestIssueEstimatePoint_CreateUpdateClear(t *testing.T) {
	db := setupTestDB(t)
	estimateTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	point := &model.EstimatePoint{
		Name: "3", Value: 3, Mode: model.EstimateModePoints,
		ProjectID: projID, WorkspaceID: wsID, Sequence: 1,
	}
	require.NoError(t, db.Create(point).Error)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	created, err := svc.Create(&request.IssueCreateRequest{
		Name: "with estimate", EstimatePointID: &point.ID,
	}, projID, wsID, userID)
	require.NoError(t, err)
	require.NotNil(t, created.EstimatePointID)
	require.Equal(t, point.ID, *created.EstimatePointID)

	got, err := svc.GetByID(created.ID)
	require.NoError(t, err)
	require.NotNil(t, got.EstimatePointID)
	require.Equal(t, point.ID, *got.EstimatePointID)

	clear := uint64(0)
	updated, err := svc.Update(created.ID, &request.IssueUpdateRequest{EstimatePointID: &clear}, userID)
	require.NoError(t, err)
	require.Nil(t, updated.EstimatePointID)

	foreign := uint64(99999)
	_, err = svc.Update(created.ID, &request.IssueUpdateRequest{EstimatePointID: &foreign}, userID)
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok)
	require.Equal(t, 400, appErr.Code)
}

func TestIssueEstimatePoint_RejectsForeignProjectPointOnCreate(t *testing.T) {
	db := setupTestDB(t)
	estimateTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	other := &model.Project{Name: "Other", Identifier: "OTH2", WorkspaceID: wsID}
	require.NoError(t, db.Create(other).Error)
	point := &model.EstimatePoint{
		Name: "5", Value: 5, Mode: model.EstimateModePoints,
		ProjectID: other.ID, WorkspaceID: wsID, Sequence: 1,
	}
	require.NoError(t, db.Create(point).Error)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	_, err := svc.Create(&request.IssueCreateRequest{
		Name: "bad point", EstimatePointID: &point.ID,
	}, projID, wsID, userID)
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok)
	require.Equal(t, 400, appErr.Code)
}

func TestImportMapsEstimatePointsToID(t *testing.T) {
	db := setupTestDB(t)
	estimateTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	point := &model.EstimatePoint{
		Name: "3", Value: 3, Mode: model.EstimateModePoints,
		ProjectID: projID, WorkspaceID: wsID, Sequence: 1,
	}
	require.NoError(t, db.Create(point).Error)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	est := 3.0
	result, err := svc.ImportFromJSON(projID, wsID, userID, []request.ImportIssueItem{
		{Name: "imported with estimate", EstimatePoints: &est},
	}, request.ImportOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, result.SuccessCount)
	require.Len(t, result.ImportedIDs, 1)

	got, err := svc.GetByID(result.ImportedIDs[0])
	require.NoError(t, err)
	require.NotNil(t, got.EstimatePointID)
	require.Equal(t, point.ID, *got.EstimatePointID)
}
