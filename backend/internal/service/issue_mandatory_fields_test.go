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

// seedIssueForMandatoryCFTests creates an issue with the given type (Backlog state from seedTestData).
func seedIssueForMandatoryCFTests(t *testing.T, db *gorm.DB, projectID, workspaceID, typeID uint64) *model.Issue {
	t.Helper()
	var st model.State
	require.NoError(t, db.Where("project_id = ? AND name = ?", projectID, "Backlog").First(&st).Error)
	issue := &model.Issue{
		Name:        "mandatory cf test issue",
		ProjectID:   projectID,
		WorkspaceID: workspaceID,
		StateID:     st.ID,
		IssueTypeID: &typeID,
	}
	require.NoError(t, db.Create(issue).Error)
	return issue
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

func TestSetIssueValue_RejectsEmptyRequired(t *testing.T) {
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
	wsID, projID, _ := seedTestData(t, db)

	bugType := &model.IssueType{Name: "Bug", WorkspaceID: wsID}
	require.NoError(t, db.Create(bugType).Error)
	fieldName := "复现步骤"
	fieldID := seedRequiredCF(t, db, wsID, bugType.ID, fieldName)

	issue := seedIssueForMandatoryCFTests(t, db, projID, wsID, bugType.ID)
	existing := "steps to reproduce"
	require.NoError(t, db.Create(&model.IssueCustomFieldValue{
		IssueID: issue.ID,
		FieldID: fieldID,
		Value:   existing,
	}).Error)

	svc := NewCustomFieldService(db)
	_, err := svc.SetIssueValue(issue.ID, request.IssueCustomFieldValueCreate{
		FieldID: fieldID,
		Value:   "",
	})
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok, "want *common.AppError, got %T: %v", err, err)
	require.Equal(t, common.ErrBadRequest, appErr.ErrorCode)
	require.Contains(t, appErr.Message, "缺少必填字段")
	require.Contains(t, appErr.Message, fieldName)

	var stored model.IssueCustomFieldValue
	require.NoError(t, db.Where("issue_id = ? AND field_id = ?", issue.ID, fieldID).First(&stored).Error)
	require.Equal(t, existing, stored.Value)
}

func TestDeleteIssueValue_RejectsRequired(t *testing.T) {
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
	wsID, projID, _ := seedTestData(t, db)

	bugType := &model.IssueType{Name: "Bug", WorkspaceID: wsID}
	require.NoError(t, db.Create(bugType).Error)
	fieldName := "复现步骤"
	fieldID := seedRequiredCF(t, db, wsID, bugType.ID, fieldName)

	issue := seedIssueForMandatoryCFTests(t, db, projID, wsID, bugType.ID)
	existing := "has value"
	require.NoError(t, db.Create(&model.IssueCustomFieldValue{
		IssueID: issue.ID,
		FieldID: fieldID,
		Value:   existing,
	}).Error)

	svc := NewCustomFieldService(db)
	err := svc.DeleteIssueValue(issue.ID, fieldID)
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok, "want *common.AppError, got %T: %v", err, err)
	require.Equal(t, common.ErrBadRequest, appErr.ErrorCode)
	require.Contains(t, appErr.Message, "缺少必填字段")
	require.Contains(t, appErr.Message, fieldName)

	var n int64
	require.NoError(t, db.Model(&model.IssueCustomFieldValue{}).
		Where("issue_id = ? AND field_id = ?", issue.ID, fieldID).Count(&n).Error)
	require.Equal(t, int64(1), n)
}

func TestBulkSetIssueValues_RejectsEmptyRequired(t *testing.T) {
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
	wsID, projID, _ := seedTestData(t, db)

	bugType := &model.IssueType{Name: "Bug", WorkspaceID: wsID}
	require.NoError(t, db.Create(bugType).Error)
	fieldName := "复现步骤"
	fieldID := seedRequiredCF(t, db, wsID, bugType.ID, fieldName)

	issue := seedIssueForMandatoryCFTests(t, db, projID, wsID, bugType.ID)
	existing := "bulk keep"
	require.NoError(t, db.Create(&model.IssueCustomFieldValue{
		IssueID: issue.ID,
		FieldID: fieldID,
		Value:   existing,
	}).Error)

	svc := NewCustomFieldService(db)
	_, err := svc.BulkSetIssueValues(request.BulkCustomFieldValueUpdate{
		IssueID: issue.ID,
		Values: []request.IssueCustomFieldValueUpdateItem{
			{FieldID: fieldID, Value: ""},
		},
	})
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok, "want *common.AppError, got %T: %v", err, err)
	require.Equal(t, common.ErrBadRequest, appErr.ErrorCode)
	require.Contains(t, appErr.Message, "缺少必填字段")
	require.Contains(t, appErr.Message, fieldName)

	var stored model.IssueCustomFieldValue
	require.NoError(t, db.Where("issue_id = ? AND field_id = ?", issue.ID, fieldID).First(&stored).Error)
	require.Equal(t, existing, stored.Value)
}

func TestConvertType_RejectsWhenTargetRequiresMissingCF(t *testing.T) {
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

	typeA := &model.IssueType{Name: "FeatureNoReq", WorkspaceID: wsID}
	require.NoError(t, db.Create(typeA).Error)
	bugType := &model.IssueType{Name: "Bug", WorkspaceID: wsID}
	require.NoError(t, db.Create(bugType).Error)
	fieldName := "复现步骤"
	_ = seedRequiredCF(t, db, wsID, bugType.ID, fieldName)

	issue := seedIssueForMandatoryCFTests(t, db, projID, wsID, typeA.ID)
	origTypeID := typeA.ID

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	_, err := svc.ConvertType(issue.ID, &request.ConvertTypeRequest{TargetTypeID: bugType.ID}, userID)
	require.Error(t, err)
	appErr, ok := err.(*common.AppError)
	require.True(t, ok, "want *common.AppError, got %T: %v", err, err)
	require.Equal(t, common.ErrBadRequest, appErr.ErrorCode)
	require.Contains(t, appErr.Message, "缺少必填字段")
	require.Contains(t, appErr.Message, fieldName)

	var reloaded model.Issue
	require.NoError(t, db.First(&reloaded, issue.ID).Error)
	require.NotNil(t, reloaded.IssueTypeID)
	require.Equal(t, origTypeID, *reloaded.IssueTypeID)
}

func TestConvertType_WithCustomFieldValues_SatisfiesRequiredAndPersists(t *testing.T) {
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

	typeA := &model.IssueType{Name: "FeatureNoReq", WorkspaceID: wsID}
	require.NoError(t, db.Create(typeA).Error)
	bugType := &model.IssueType{Name: "Bug", WorkspaceID: wsID}
	require.NoError(t, db.Create(bugType).Error)
	fieldID := seedRequiredCF(t, db, wsID, bugType.ID, "复现步骤")

	issue := seedIssueForMandatoryCFTests(t, db, projID, wsID, typeA.ID)
	value := "打开应用并复现"

	svc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	_, err := svc.ConvertType(issue.ID, &request.ConvertTypeRequest{
		TargetTypeID: bugType.ID,
		CustomFieldValues: map[uint64]interface{}{
			fieldID: value,
		},
	}, userID)
	require.NoError(t, err)

	var reloaded model.Issue
	require.NoError(t, db.First(&reloaded, issue.ID).Error)
	require.NotNil(t, reloaded.IssueTypeID)
	require.Equal(t, bugType.ID, *reloaded.IssueTypeID)

	var stored model.IssueCustomFieldValue
	require.NoError(t, db.Where("issue_id = ? AND field_id = ?", issue.ID, fieldID).First(&stored).Error)
	require.Equal(t, value, stored.Value)
}
