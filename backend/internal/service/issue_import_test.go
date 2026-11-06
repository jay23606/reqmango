package service

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// importTestDB mirrors sequenceTestDB but migrates only the tables the import
// path reads, so the suite stays fast.
func importTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run issue import tests")
	}
	config := &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)}
	admin, err := gorm.Open(postgres.Open(dsn), config)
	require.NoError(t, err)
	adminSQL, err := admin.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, adminSQL.Close()) })
	schema := "import_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, admin.Exec("CREATE SCHEMA "+schema).Error)
	t.Cleanup(func() { require.NoError(t, admin.Exec("DROP SCHEMA "+schema+" CASCADE").Error) })

	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Contains(t, []string{"postgres", "postgresql"}, parsed.Scheme, "TEST_DATABASE_URL must be a PostgreSQL URL")
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(parsed.String()), config)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	require.NoError(t, db.AutoMigrate(
		&model.Project{}, &model.State{}, &model.IssueType{}, &model.IssueTypeImport{}, &model.Label{},
		&model.User{}, &model.WorkspaceMember{}, &model.ProjectMember{}, &model.Issue{},
		&model.CustomField{}, &model.IssueTypeField{}, &model.IssueCustomFieldValue{}, &model.IssueActivity{},
		&model.IssueAssignee{}, &model.IssueLabel{}, &model.IssueCycle{}, &model.Cycle{},
		&model.ModuleIssue{}, &model.Module{}, &model.Webhook{},
	))
	return db
}

func importTestProject(t *testing.T, db *gorm.DB, workspaceID uint64) model.Project {
	t.Helper()
	project := model.Project{Name: "Import test", Identifier: "IMP" + strings.ToUpper(uuid.NewString()[:6]), WorkspaceID: workspaceID}
	require.NoError(t, db.Create(&project).Error)
	require.NoError(t, db.Create(&model.State{Name: "Backlog", Group: "backlog", IsDefault: true, ProjectID: &project.ID, WorkspaceID: workspaceID}).Error)
	return project
}

// importTestProjectWithImportedBug seeds a workspace Bug type (imported into the project) with one required custom field.
func importTestProjectWithImportedBugAndRequiredCF(t *testing.T, db *gorm.DB, workspaceID uint64) (project model.Project, fieldDisplayName string) {
	t.Helper()
	project = importTestProject(t, db, workspaceID)
	bug := &model.IssueType{Name: "Bug", WorkspaceID: workspaceID}
	require.NoError(t, db.Create(bug).Error)
	require.NoError(t, db.Create(&model.IssueTypeImport{
		ProjectID:       project.ID,
		WorkspaceTypeID: bug.ID,
		WorkspaceID:     workspaceID,
	}).Error)
	fieldDisplayName = "复现步骤"
	f := &model.CustomField{Name: fieldDisplayName, FieldType: "text", WorkspaceID: workspaceID}
	require.NoError(t, db.Create(f).Error)
	require.NoError(t, db.Create(&model.IssueTypeField{
		TypeID: bug.ID, FieldID: f.ID, IsRequired: true,
	}).Error)
	return project, fieldDisplayName
}

func TestParseCSVRecordsAppliesExplicitMapping(t *testing.T) {
	records := [][]string{
		{"工作项名称", "负责人邮箱", "所处阶段"},
		{"登录页报错", "dev@example.com", "Todo"},
	}
	headers, applied, items, err := parseCSVRecords(records, map[string]string{
		"name":      "工作项名称",
		"assignees": "负责人邮箱",
		"state":     "所处阶段",
	})
	require.NoError(t, err)
	require.Equal(t, []string{"工作项名称", "负责人邮箱", "所处阶段"}, headers)
	require.Equal(t, "工作项名称", applied["name"])
	require.Equal(t, "所处阶段", applied["state"])
	require.Len(t, items, 1)
	require.Equal(t, "登录页报错", items[0].Name)
	require.Equal(t, []string{"dev@example.com"}, items[0].AssigneeEmails)
	require.Equal(t, "Todo", items[0].StateName)
}

func TestParseCSVRecordsFallsBackToAliases(t *testing.T) {
	records := [][]string{
		{"标题", "描述", "优先级", "状态", "类型", "标签", "开始日期", "截止日期", "估点"},
		{"首页空白", "打开即白屏", "high", "Todo", "Bug", "前端,回归", "2026-01-01", "2026-01-05", "3"},
	}
	_, applied, items, err := parseCSVRecords(records, nil)
	require.NoError(t, err)
	require.Equal(t, "标题", applied["name"])
	require.Len(t, items, 1)
	require.Equal(t, "首页空白", items[0].Name)
	require.Equal(t, "high", items[0].Priority)
	require.Equal(t, []string{"前端", "回归"}, items[0].LabelNames)
	require.NotNil(t, items[0].EstimatePoints)
	require.Equal(t, 3.0, *items[0].EstimatePoints)
}

func TestParseCSVRecordsRejectsUnknownMappedColumn(t *testing.T) {
	records := [][]string{{"标题"}, {"x"}}
	_, _, _, err := parseCSVRecords(records, map[string]string{"name": "不存在的列"})
	require.Error(t, err)
}

func TestParseCSVRecordsExplicitMappingDisablesAliasFallback(t *testing.T) {
	records := [][]string{
		{"name", "状态"},
		{"x", "Todo"},
	}
	_, applied, items, err := parseCSVRecords(records, map[string]string{"name": "name"})
	require.NoError(t, err)
	require.Equal(t, "name", applied["name"])
	_, hasState := applied["state"]
	require.False(t, hasState, "unmapped fields must stay unmapped")
	require.Empty(t, items[0].StateName)
}

func TestParseCSVRecordsRequiresNameColumn(t *testing.T) {
	records := [][]string{{"描述"}, {"x"}}
	_, _, _, err := parseCSVRecords(records, nil)
	require.Error(t, err)
}

func TestImportDryRunReportsWithoutWriting(t *testing.T) {
	db := importTestDB(t)
	project := importTestProject(t, db, 7)
	svc := NewIssueService(db, nil, nil, nil, nil)

	result, err := svc.importItems(project.ID, 7, 1, []request.ImportIssueItem{
		{Name: "会成功的行", StateName: "Backlog"},
		{Name: "未知状态的行", StateName: "不存在的状态", TypeName: "不存在的类型"},
		{Name: ""},
	}, request.ImportOptions{DryRun: true})
	require.NoError(t, err)

	require.True(t, result.DryRun)
	require.Equal(t, 3, result.TotalRows)
	require.Equal(t, 2, result.SuccessCount, "two rows pass validation")
	require.Equal(t, 1, result.FailCount, "empty title fails")
	require.Empty(t, result.ImportedIDs, "dry run must not create anything")
	require.Len(t, result.Warnings, 2, "unknown state and type are surfaced")

	var count int64
	require.NoError(t, db.Model(&model.Issue{}).Count(&count).Error)
	require.Zero(t, count, "dry run must not persist issues")

	fields := map[string]bool{}
	for _, w := range result.Warnings {
		fields[w.Field] = true
	}
	require.True(t, fields["state"])
	require.True(t, fields["type"])
}

func TestImportWarnsAboutAssigneeOutsideProject(t *testing.T) {
	db := importTestDB(t)
	project := importTestProject(t, db, 7)
	svc := NewIssueService(db, nil, nil, nil, nil)

	user := model.User{Email: "outsider@example.com", Username: "outsider", IsActive: true}
	require.NoError(t, db.Create(&user).Error)
	require.NoError(t, db.Create(&model.WorkspaceMember{WorkspaceID: 7, UserID: user.ID, Role: 15, IsActive: true}).Error)
	// Deliberately no ProjectMember row: Create() would reject this assignee.

	result, err := svc.importItems(project.ID, 7, 1, []request.ImportIssueItem{
		{Name: "带负责人", AssigneeEmails: []string{"outsider@example.com"}},
	}, request.ImportOptions{DryRun: true})
	require.NoError(t, err)

	require.Equal(t, 1, result.SuccessCount)
	require.Len(t, result.Warnings, 1)
	require.Equal(t, "assignee", result.Warnings[0].Field)
	require.Contains(t, result.Warnings[0].Message, "不是该项目成员")
}

func TestImportRejectsProjectOutsideWorkspace(t *testing.T) {
	db := importTestDB(t)
	project := importTestProject(t, db, 7)
	svc := NewIssueService(db, nil, nil, nil, nil)

	_, err := svc.importItems(project.ID, 9, 1, []request.ImportIssueItem{{Name: "x"}}, request.ImportOptions{DryRun: true})
	require.Error(t, err)
	require.Contains(t, strings.ToLower(err.Error()), "工作空间")
}

func TestImportRejectsTooManyRows(t *testing.T) {
	db := importTestDB(t)
	project := importTestProject(t, db, 7)
	svc := NewIssueService(db, nil, nil, nil, nil)

	items := make([]request.ImportIssueItem, maxImportRows+1)
	for i := range items {
		items[i] = request.ImportIssueItem{Name: "row"}
	}
	_, err := svc.importItems(project.ID, 7, 1, items, request.ImportOptions{DryRun: true})
	require.Error(t, err)
	require.Contains(t, err.Error(), "5000")
}

func TestImportFormalWriteRejectsMissingMandatoryCustomField(t *testing.T) {
	db := importTestDB(t)
	project, fieldName := importTestProjectWithImportedBugAndRequiredCF(t, db, 7)
	svc := NewIssueService(db, nil, nil, nil, nil)

	result, err := svc.importItems(project.ID, 7, 1, []request.ImportIssueItem{
		{Name: "import 缺必填", StateName: "Backlog", TypeName: "Bug"},
	}, request.ImportOptions{DryRun: false})
	require.NoError(t, err)
	require.False(t, result.DryRun)
	require.Equal(t, 1, result.FailCount)
	require.Zero(t, result.SuccessCount)
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0].Message, "缺少必填字段")
	require.Contains(t, result.Errors[0].Message, fieldName)
	require.Empty(t, result.ImportedIDs)

	var n int64
	require.NoError(t, db.Model(&model.Issue{}).Where("project_id = ?", project.ID).Count(&n).Error)
	require.Zero(t, n, "must not create issue when mandatory custom field is missing")
}

func TestImportDryRunReportsMissingMandatoryCustomField(t *testing.T) {
	db := importTestDB(t)
	project, fieldName := importTestProjectWithImportedBugAndRequiredCF(t, db, 7)
	svc := NewIssueService(db, nil, nil, nil, nil)

	result, err := svc.importItems(project.ID, 7, 1, []request.ImportIssueItem{
		{Name: "dry 缺必填", StateName: "Backlog", TypeName: "Bug"},
	}, request.ImportOptions{DryRun: true})
	require.NoError(t, err)
	require.True(t, result.DryRun)
	require.Equal(t, 1, result.FailCount)
	require.Zero(t, result.SuccessCount)
	require.Len(t, result.Errors, 1)
	require.Contains(t, result.Errors[0].Message, fieldName)
	require.Empty(t, result.ImportedIDs)

	var n int64
	require.NoError(t, db.Model(&model.Issue{}).Where("project_id = ?", project.ID).Count(&n).Error)
	require.Zero(t, n, "dry run must not persist issues")
}
