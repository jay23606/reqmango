package service

import (
	"testing"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
)

func mergeTestMigrate(t *testing.T, db interface{ AutoMigrate(...interface{}) error }) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&model.Issue{}, &model.IssueType{}, &model.State{},
		&model.IssueActivity{}, &model.IssueAssignee{}, &model.IssueLabel{}, &model.Label{},
		&model.IssueTypeImport{}, &model.Webhook{},
		&model.RelationType{}, &model.IssueRelation{},
		&model.IssueCycle{}, &model.Cycle{}, &model.ModuleIssue{}, &model.Module{},
	))
}

func TestMergeDuplicates_SoftMergesSameProject(t *testing.T) {
	db := setupTestDB(t)
	mergeTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	cancelled := &model.State{
		Name: "Cancelled", Color: "#EF4444", Group: "cancelled",
		ProjectID: &projID, WorkspaceID: wsID,
	}
	require.NoError(t, db.Create(cancelled).Error)

	label := &model.Label{Name: "bug", ProjectID: projID, WorkspaceID: wsID}
	require.NoError(t, db.Create(label).Error)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	src, err := svc.Create(&request.IssueCreateRequest{
		Name: "dup source", DescriptionHTML: "<p>source body</p>",
	}, projID, wsID, userID)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.IssueLabel{IssueID: src.ID, LabelID: label.ID}).Error)

	tgt, err := svc.Create(&request.IssueCreateRequest{
		Name: "canonical", DescriptionHTML: "<p>target</p>",
	}, projID, wsID, userID)
	require.NoError(t, err)

	out, err := svc.MergeDuplicates(&request.MergeDuplicatesRequest{
		TargetIssueID:  tgt.ID,
		SourceIssueIDs: []uint64{src.ID},
	}, userID)
	require.NoError(t, err)
	require.Equal(t, tgt.ID, out.ID)

	var srcRow model.Issue
	require.NoError(t, db.First(&srcRow, src.ID).Error)
	require.Equal(t, cancelled.ID, srcRow.StateID, "source should move to cancelled")

	var relCount int64
	require.NoError(t, db.Model(&model.IssueRelation{}).
		Where("issue_id = ? AND related_issue_id = ?", src.ID, tgt.ID).Count(&relCount).Error)
	require.Equal(t, int64(1), relCount)

	var labelCount int64
	require.NoError(t, db.Model(&model.IssueLabel{}).
		Where("issue_id = ? AND label_id = ?", tgt.ID, label.ID).Count(&labelCount).Error)
	require.Equal(t, int64(1), labelCount)

	var tgtRow model.Issue
	require.NoError(t, db.First(&tgtRow, tgt.ID).Error)
	require.Contains(t, tgtRow.DescriptionHTML, "Merged from")
	require.Contains(t, tgtRow.DescriptionHTML, "dup source")

	// Idempotent
	_, err = svc.MergeDuplicates(&request.MergeDuplicatesRequest{
		TargetIssueID:  tgt.ID,
		SourceIssueIDs: []uint64{src.ID},
	}, userID)
	require.NoError(t, err)
}

func TestMergeDuplicates_RejectsCrossProject(t *testing.T) {
	db := setupTestDB(t)
	mergeTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	other := &model.Project{Name: "Other", Identifier: "OTH", WorkspaceID: wsID}
	require.NoError(t, db.Create(other).Error)
	st := &model.State{Name: "Backlog", Color: "#999", Group: "backlog", ProjectID: &other.ID, WorkspaceID: wsID, IsDefault: true}
	require.NoError(t, db.Create(st).Error)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	src, err := svc.Create(&request.IssueCreateRequest{Name: "a"}, projID, wsID, userID)
	require.NoError(t, err)
	tgt, err := svc.Create(&request.IssueCreateRequest{Name: "b"}, other.ID, wsID, userID)
	require.NoError(t, err)

	_, err = svc.MergeDuplicates(&request.MergeDuplicatesRequest{
		TargetIssueID: tgt.ID, SourceIssueIDs: []uint64{src.ID},
	}, userID)
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok)
	require.Equal(t, 400, appErr.Code)
}

func TestMergeDuplicates_RejectsSelf(t *testing.T) {
	db := setupTestDB(t)
	mergeTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)
	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	iss, err := svc.Create(&request.IssueCreateRequest{Name: "solo"}, projID, wsID, userID)
	require.NoError(t, err)

	_, err = svc.MergeDuplicates(&request.MergeDuplicatesRequest{
		TargetIssueID: iss.ID, SourceIssueIDs: []uint64{iss.ID},
	}, userID)
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok)
	require.Equal(t, 400, appErr.Code)
}

func TestMergeDuplicates_ConflictWhenAlreadyMergedElsewhere(t *testing.T) {
	db := setupTestDB(t)
	mergeTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)
	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)

	src, err := svc.Create(&request.IssueCreateRequest{Name: "src"}, projID, wsID, userID)
	require.NoError(t, err)
	t1, err := svc.Create(&request.IssueCreateRequest{Name: "t1"}, projID, wsID, userID)
	require.NoError(t, err)
	t2, err := svc.Create(&request.IssueCreateRequest{Name: "t2"}, projID, wsID, userID)
	require.NoError(t, err)

	_, err = svc.MergeDuplicates(&request.MergeDuplicatesRequest{
		TargetIssueID: t1.ID, SourceIssueIDs: []uint64{src.ID},
	}, userID)
	require.NoError(t, err)

	_, err = svc.MergeDuplicates(&request.MergeDuplicatesRequest{
		TargetIssueID: t2.ID, SourceIssueIDs: []uint64{src.ID},
	}, userID)
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok)
	require.Equal(t, 409, appErr.Code)
}
