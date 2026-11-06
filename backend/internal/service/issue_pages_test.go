package service

import (
	"testing"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupIssuePagesTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.IssuePage{}))
	return db
}

func issuePagesTestStateID(t *testing.T, db *gorm.DB, projectID uint64) uint64 {
	t.Helper()
	var st model.State
	require.NoError(t, db.Where("project_id = ?", projectID).First(&st).Error)
	return st.ID
}

func TestIssuePages_AddPage_IdempotentAndSameProject(t *testing.T) {
	db := setupIssuePagesTestDB(t)
	wsID, projID, _ := seedTestData(t, db)
	stateID := issuePagesTestStateID(t, db, projID)

	issue := &model.Issue{
		Name:            "linked",
		DescriptionHTML: "<p></p>",
		ProjectID:       projID,
		WorkspaceID:     wsID,
		StateID:         stateID,
	}
	require.NoError(t, db.Create(issue).Error)

	page := &model.Page{Title: "Doc A", Content: "x", ProjectID: projID, WorkspaceID: wsID}
	require.NoError(t, db.Create(page).Error)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	require.NoError(t, svc.AddPage(issue.ID, page.ID, 1))
	require.NoError(t, svc.AddPage(issue.ID, page.ID, 1))

	var n int64
	require.NoError(t, db.Model(&model.IssuePage{}).Where("issue_id = ? AND page_id = ?", issue.ID, page.ID).Count(&n).Error)
	assert.EqualValues(t, 1, n)

	pages, err := svc.ListPages(issue.ID)
	require.NoError(t, err)
	require.Len(t, pages, 1)
	assert.Equal(t, page.ID, pages[0].ID)
}

func TestIssuePages_AddPage_RejectOtherProjectPage(t *testing.T) {
	db := setupIssuePagesTestDB(t)
	wsID, projID, _ := seedTestData(t, db)
	stateID := issuePagesTestStateID(t, db, projID)

	other := &model.Project{Name: "Other", Identifier: "OTH", WorkspaceID: wsID}
	require.NoError(t, db.Create(other).Error)

	issue := &model.Issue{
		Name:            "i1",
		DescriptionHTML: "<p></p>",
		ProjectID:       projID,
		WorkspaceID:     wsID,
		StateID:         stateID,
	}
	require.NoError(t, db.Create(issue).Error)

	foreignPage := &model.Page{Title: "Foreign", Content: "", ProjectID: other.ID, WorkspaceID: wsID}
	require.NoError(t, db.Create(foreignPage).Error)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	err := svc.AddPage(issue.ID, foreignPage.ID, 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "same project")
	if ae, ok := err.(*common.AppError); assert.True(t, ok) {
		assert.Equal(t, 403, ae.Code)
	}
}

func TestIssuePages_RemovePage_NotLinked(t *testing.T) {
	db := setupIssuePagesTestDB(t)
	wsID, projID, _ := seedTestData(t, db)
	stateID := issuePagesTestStateID(t, db, projID)

	issue := &model.Issue{
		Name:            "i1",
		DescriptionHTML: "<p></p>",
		ProjectID:       projID,
		WorkspaceID:     wsID,
		StateID:         stateID,
	}
	require.NoError(t, db.Create(issue).Error)

	page := &model.Page{Title: "P", Content: "", ProjectID: projID, WorkspaceID: wsID}
	require.NoError(t, db.Create(page).Error)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	err := svc.RemovePage(issue.ID, page.ID, 1)
	require.Error(t, err)
	if ae, ok := err.(*common.AppError); assert.True(t, ok) {
		assert.Equal(t, 404, ae.Code)
	}
}
