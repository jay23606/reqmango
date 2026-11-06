package service

import (
	"testing"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/dto/response"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// hierarchyTestMigrate extends setupTestDB with tables referenced by IssueService.Create (same set as issue_mandatory_fields_test).
func hierarchyTestMigrate(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.AutoMigrate(
		&model.IssueTypeImport{},
		&model.CustomField{},
		&model.ConditionalField{},
		&model.IssueTypeField{},
		&model.IssueCustomFieldValue{},
		&model.IssueActivity{},
		&model.IssueAssignee{},
		&model.IssueLabel{},
		&model.Label{},
		&model.IssueCycle{},
		&model.Cycle{},
		&model.ModuleIssue{},
		&model.Module{},
		&model.Webhook{},
		&model.ReleaseIssue{},
	))
}

func importWorkspaceTypesForProject(t *testing.T, db *gorm.DB, projectID, workspaceID uint64, typeIDs ...uint64) {
	t.Helper()
	for _, id := range typeIDs {
		require.NoError(t, db.Create(&model.IssueTypeImport{
			ProjectID:       projectID,
			WorkspaceTypeID: id,
			WorkspaceID:     workspaceID,
		}).Error)
	}
}

func requireBadRequestMessageContains(t *testing.T, err error, substr string) {
	t.Helper()
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok, "want *common.AppError, got %T: %v", err, err)
	require.Equal(t, common.ErrBadRequest, appErr.ErrorCode)
	require.Contains(t, appErr.Message, substr)
}

func countIssues(t *testing.T, db *gorm.DB, projectID uint64) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&model.Issue{}).Where("project_id = ?", projectID).Count(&n).Error)
	return n
}

// AC-01.1: parent at max depth (5) cannot get another child; stable message; no new row.
func TestIssueHierarchy_AC01_1_MaxDepthExceeded(t *testing.T) {
	db := setupTestDB(t)
	hierarchyTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	var taskType model.IssueType
	require.NoError(t, db.Where("workspace_id = ? AND name = ?", wsID, "Task").First(&taskType).Error)
	importWorkspaceTypesForProject(t, db, projID, wsID, taskType.ID)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)

	tid := taskType.ID
	var last *response.IssueResponse
	for i := 0; i < 6; i++ {
		var parentID *uint64
		if last != nil {
			pid := last.ID
			parentID = &pid
		}
		var resp *response.IssueResponse
		var err error
		if parentID == nil {
			resp, err = svc.Create(&request.IssueCreateRequest{
				Name:   "depth-chain-root",
				TypeID: &tid,
			}, projID, wsID, userID)
		} else {
			// Omit explicit type on nested rows — hierarchy depth still follows parent.
			resp, err = svc.Create(&request.IssueCreateRequest{
				Name:     "depth-chain-child",
				ParentID: parentID,
			}, projID, wsID, userID)
		}
		require.NoError(t, err)
		require.NotNil(t, resp)
		last = resp
	}

	var parentAtDepth5 model.Issue
	require.NoError(t, db.Where("project_id = ?", projID).Order("depth DESC").First(&parentAtDepth5).Error)
	require.Equal(t, 5, parentAtDepth5.Depth)

	before := countIssues(t, db, projID)
	pid := parentAtDepth5.ID
	_, err := svc.Create(&request.IssueCreateRequest{
		Name:     "too-deep",
		ParentID: &pid,
	}, projID, wsID, userID)
	requireBadRequestMessageContains(t, err, "Maximum hierarchy depth")
	require.Equal(t, before, countIssues(t, db, projID))
}

// AC-02.1: workspace root type (level 0, no parent_type_id) cannot be used under a parent issue.
func TestIssueHierarchy_AC02_1_RootTypeCannotHaveParent(t *testing.T) {
	db := setupTestDB(t)
	hierarchyTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	rootType := &model.IssueType{Name: "RootOnly", WorkspaceID: wsID, Level: 0, ParentTypeID: nil}
	require.NoError(t, db.Create(rootType).Error)
	childCapable := &model.IssueType{Name: "UnderRoot", WorkspaceID: wsID, Level: 1, ParentTypeID: &rootType.ID}
	require.NoError(t, db.Create(childCapable).Error)
	importWorkspaceTypesForProject(t, db, projID, wsID, rootType.ID, childCapable.ID)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)

	parentTypeID := childCapable.ID
	parent, err := svc.Create(&request.IssueCreateRequest{
		Name:   "parent-for-root-child-test",
		TypeID: &parentTypeID,
	}, projID, wsID, userID)
	require.NoError(t, err)

	before := countIssues(t, db, projID)
	rid := rootType.ID
	pid := parent.ID
	_, err = svc.Create(&request.IssueCreateRequest{
		Name:     "invalid-root-as-child",
		TypeID:   &rid,
		ParentID: &pid,
	}, projID, wsID, userID)
	requireBadRequestMessageContains(t, err, "root-level type cannot have a parent")
	require.Equal(t, before, countIssues(t, db, projID))
}

// AC-03.1: when parent type level > 0, child type level must not be shallower.
func TestIssueHierarchy_AC03_1_ChildTypeLevelNotShallowerThanParent(t *testing.T) {
	db := setupTestDB(t)
	hierarchyTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	typeL2 := &model.IssueType{Name: "L2Type", WorkspaceID: wsID, Level: 2, ParentTypeID: nil}
	require.NoError(t, db.Create(typeL2).Error)
	typeL1 := &model.IssueType{Name: "L1Type", WorkspaceID: wsID, Level: 1, ParentTypeID: nil}
	require.NoError(t, db.Create(typeL1).Error)
	importWorkspaceTypesForProject(t, db, projID, wsID, typeL2.ID, typeL1.ID)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)

	pt := typeL2.ID
	parent, err := svc.Create(&request.IssueCreateRequest{
		Name:   "parent-l2",
		TypeID: &pt,
	}, projID, wsID, userID)
	require.NoError(t, err)

	before := countIssues(t, db, projID)
	ct := typeL1.ID
	pid := parent.ID
	_, err = svc.Create(&request.IssueCreateRequest{
		Name:     "child-l1-under-l2-parent",
		TypeID:   &ct,
		ParentID: &pid,
	}, projID, wsID, userID)
	requireBadRequestMessageContains(t, err, "child type level must not be shallower than parent type level")
	require.Equal(t, before, countIssues(t, db, projID))
}

// AC-04.1: allowed_child_type_ids whitelist — child type must appear when list is non-empty.
func TestIssueHierarchy_AC04_1_AllowedChildWhitelist(t *testing.T) {
	db := setupTestDB(t)
	hierarchyTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	typeA := &model.IssueType{Name: "AllowedA", WorkspaceID: wsID, Level: 1, ParentTypeID: nil}
	require.NoError(t, db.Create(typeA).Error)
	typeB := &model.IssueType{Name: "BlockedB", WorkspaceID: wsID, Level: 2, ParentTypeID: nil}
	require.NoError(t, db.Create(typeB).Error)
	parentType := &model.IssueType{
		Name: "ParentWithAllowlist", WorkspaceID: wsID, Level: 1, ParentTypeID: nil,
		AllowedChildTypeIDs: model.JSONUint64Array{typeA.ID},
	}
	require.NoError(t, db.Create(parentType).Error)
	importWorkspaceTypesForProject(t, db, projID, wsID, parentType.ID, typeA.ID, typeB.ID)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)

	ptid := parentType.ID
	parent, err := svc.Create(&request.IssueCreateRequest{
		Name:   "parent-allowlist",
		TypeID: &ptid,
	}, projID, wsID, userID)
	require.NoError(t, err)

	before := countIssues(t, db, projID)
	bid := typeB.ID
	pid := parent.ID
	_, err = svc.Create(&request.IssueCreateRequest{
		Name:     "child-not-in-allowlist",
		TypeID:   &bid,
		ParentID: &pid,
	}, projID, wsID, userID)
	requireBadRequestMessageContains(t, err, "this child type is not allowed under the parent type")
	require.Equal(t, before, countIssues(t, db, projID))
}

// AC-04.2: child type's parent_type_id must match parent's issue type when both are set.
func TestIssueHierarchy_AC04_2_ParentTypeIDMustMatchParentIssueType(t *testing.T) {
	db := setupTestDB(t)
	hierarchyTestMigrate(t, db)
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	typeX := &model.IssueType{Name: "ExpectedParentX", WorkspaceID: wsID, Level: 1, ParentTypeID: nil}
	require.NoError(t, db.Create(typeX).Error)
	typeY := &model.IssueType{Name: "ActualParentY", WorkspaceID: wsID, Level: 1, ParentTypeID: nil}
	require.NoError(t, db.Create(typeY).Error)
	childType := &model.IssueType{
		Name: "ChildExpectsX", WorkspaceID: wsID, Level: 2, ParentTypeID: &typeX.ID,
	}
	require.NoError(t, db.Create(childType).Error)
	importWorkspaceTypesForProject(t, db, projID, wsID, typeX.ID, typeY.ID, childType.ID)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)

	yid := typeY.ID
	parent, err := svc.Create(&request.IssueCreateRequest{
		Name:   "parent-type-y",
		TypeID: &yid,
	}, projID, wsID, userID)
	require.NoError(t, err)

	before := countIssues(t, db, projID)
	cid := childType.ID
	pid := parent.ID
	_, err = svc.Create(&request.IssueCreateRequest{
		Name:     "child-wrong-parent-type-link",
		TypeID:   &cid,
		ParentID: &pid,
	}, projID, wsID, userID)
	requireBadRequestMessageContains(t, err, "parent type does not allow this child type")
	require.Equal(t, before, countIssues(t, db, projID))
}
