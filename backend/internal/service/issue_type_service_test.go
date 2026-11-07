package service

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupIssueTypeServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.IssueType{}))
	return db
}

// TestIssueTypeService_AllowedChildCreateUpdateGetRoundTrip covers API-style
// create (with allowed_child_type_ids), update, and get for AC-05.1.
func TestIssueTypeService_AllowedChildCreateUpdateGetRoundTrip(t *testing.T) {
	db := setupIssueTypeServiceDB(t)
	const wsID uint64 = 1
	const userID uint64 = 42

	childA := &model.IssueType{Name: "Child A", WorkspaceID: wsID, Sequence: 1, IsActive: true}
	childB := &model.IssueType{Name: "Child B", WorkspaceID: wsID, Sequence: 2, IsActive: true}
	childC := &model.IssueType{Name: "Child C", WorkspaceID: wsID, Sequence: 3, IsActive: true}
	require.NoError(t, db.Create(childA).Error)
	require.NoError(t, db.Create(childB).Error)
	require.NoError(t, db.Create(childC).Error)

	svc := NewIssueTypeService(db)

	created, err := svc.Create(wsID, userID, request.IssueTypeCreate{
		Name:                "Parent",
		AllowedChildTypeIDs: []uint64{childA.ID, childB.ID},
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	require.Equal(t, []uint64{childA.ID, childB.ID}, created.AllowedChildTypeIDs)

	got, err := svc.Get(created.ID)
	require.NoError(t, err)
	require.Equal(t, []uint64{childA.ID, childB.ID}, got.AllowedChildTypeIDs)

	updated, err := svc.Update(created.ID, userID, request.IssueTypeUpdate{
		AllowedChildTypeIDs: []uint64{childB.ID, childC.ID},
	})
	require.NoError(t, err)
	require.Equal(t, []uint64{childB.ID, childC.ID}, updated.AllowedChildTypeIDs)

	got2, err := svc.Get(created.ID)
	require.NoError(t, err)
	require.Equal(t, []uint64{childB.ID, childC.ID}, got2.AllowedChildTypeIDs)
}
