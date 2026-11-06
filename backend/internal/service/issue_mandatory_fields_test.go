package service

import (
	"testing"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedRequiredCF(t *testing.T, db *gorm.DB, wsID, typeID uint64, name string) uint64 {
	t.Helper()
	f := &model.CustomField{Name: name, FieldType: "text", WorkspaceID: wsID}
	require.NoError(t, db.Create(f).Error)
	require.NoError(t, db.Create(&model.IssueTypeField{
		TypeID: typeID, FieldID: f.ID, IsRequired: true,
	}).Error)
	return f.ID
}

// ensureDefaultIssueState marks the project's backlog state as default so IssueService.Create can resolve state_id.
func ensureDefaultIssueState(t *testing.T, db *gorm.DB, projectID uint64) {
	t.Helper()
	res := db.Model(&model.State{}).
		Where("project_id = ? AND name = ?", projectID, "Backlog").
		Updates(map[string]interface{}{"is_default": true, "group": "backlog"})
	require.NoError(t, res.Error)
	require.Greater(t, res.RowsAffected, int64(0), "expected a Backlog state for the project")
}

func TestCreate_RejectsMissingRequiredCustomField(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.CustomField{},
		&model.IssueTypeField{},
		&model.IssueTypeImport{},
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
	))
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	bugType := &model.IssueType{Name: "Bug", WorkspaceID: wsID}
	require.NoError(t, db.Create(bugType).Error)
	fieldName := "复现步骤"
	fieldID := seedRequiredCF(t, db, wsID, bugType.ID, fieldName)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	typeID := bugType.ID

	t.Run("omitted_custom_field_values", func(t *testing.T) {
		_, err := svc.Create(&request.IssueCreateRequest{
			Name:   "missing cf",
			TypeID: &typeID,
		}, projID, wsID, userID)
		require.Error(t, err)
		appErr, ok := err.(*common.AppError)
		require.True(t, ok, "want *common.AppError, got %T: %v", err, err)
		require.Equal(t, common.ErrBadRequest, appErr.ErrorCode)
		require.Contains(t, appErr.Message, "缺少必填字段")
		require.Contains(t, appErr.Message, fieldName)

		var n int64
		require.NoError(t, db.Model(&model.Issue{}).Where("project_id = ?", projID).Count(&n).Error)
		require.Zero(t, n, "issue must not persist when mandatory CF is missing")
	})

	t.Run("empty_string_value", func(t *testing.T) {
		_, err := svc.Create(&request.IssueCreateRequest{
			Name:   "empty cf",
			TypeID: &typeID,
			CustomFieldValues: map[uint64]interface{}{
				fieldID: "",
			},
		}, projID, wsID, userID)
		require.Error(t, err)
		appErr, ok := err.(*common.AppError)
		require.True(t, ok, "want *common.AppError, got %T: %v", err, err)
		require.Equal(t, common.ErrBadRequest, appErr.ErrorCode)
		require.Contains(t, appErr.Message, "缺少必填字段")
		require.Contains(t, appErr.Message, fieldName)

		var n int64
		require.NoError(t, db.Model(&model.Issue{}).Where("project_id = ?", projID).Count(&n).Error)
		require.Zero(t, n, "issue must not persist when mandatory CF is empty string")
	})
}

func TestCreate_WithRequiredCustomFieldValue_Succeeds(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.CustomField{},
		&model.IssueTypeField{},
		&model.IssueTypeImport{},
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
	))
	wsID, projID, userID := seedTestData(t, db)
	ensureDefaultIssueState(t, db, projID)

	bugType := &model.IssueType{Name: "Bug", WorkspaceID: wsID}
	require.NoError(t, db.Create(bugType).Error)
	fieldName := "复现步骤"
	fieldID := seedRequiredCF(t, db, wsID, bugType.ID, fieldName)

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	typeID := bugType.ID
	value := "打开应用，点击登录"

	issue, err := svc.Create(&request.IssueCreateRequest{
		Name:   "has required cf",
		TypeID: &typeID,
		CustomFieldValues: map[uint64]interface{}{
			fieldID: value,
		},
	}, projID, wsID, userID)
	require.NoError(t, err)
	require.NotNil(t, issue)

	var stored model.IssueCustomFieldValue
	require.NoError(t, db.Where("issue_id = ? AND field_id = ?", issue.ID, fieldID).First(&stored).Error)
	require.Equal(t, value, stored.Value)

	var n int64
	require.NoError(t, db.Model(&model.Issue{}).Where("project_id = ? AND id = ?", projID, issue.ID).Count(&n).Error)
	require.Equal(t, int64(1), n)
}
