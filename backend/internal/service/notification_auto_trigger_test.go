package service

import (
	"testing"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupNotificationTestDB uses page test seed data plus tables IssueService.buildResponse
// preloads (omit a table and First() fails with "Issue not found").
func setupNotificationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&model.IssueLabel{},
		&model.Label{},
		&model.IssueCycle{},
		&model.Cycle{},
		&model.ModuleIssue{},
		&model.Module{},
		&model.ReleaseIssue{},
		&model.Release{},
		&model.Attachment{},
		&model.Notification{},
		&model.IssueAssignee{},
		&model.IssueWatcher{},
		&model.Comment{},
		&model.IssueActivity{},
		&model.Webhook{},
		&model.ProjectSubscriber{},
		&model.IssueTypeImport{},
	))
	return db
}

func linkWorkspaceIssueTypeToProject(t *testing.T, db *gorm.DB, projectID, workspaceID uint64) {
	t.Helper()
	var wt model.IssueType
	require.NoError(t, db.Where("workspace_id = ? AND project_id IS NULL", workspaceID).First(&wt).Error)
	require.NoError(t, db.Create(&model.IssueTypeImport{
		ProjectID:       projectID,
		WorkspaceTypeID: wt.ID,
		WorkspaceID:     workspaceID,
	}).Error)
}

func seedProjectMember(t *testing.T, db *gorm.DB, projectID uint64, username, email string) uint64 {
	t.Helper()
	u := &model.User{DisplayName: username, Username: username, Email: email}
	require.NoError(t, db.Create(u).Error)
	require.NoError(t, db.Create(&model.ProjectMember{
		ProjectID: projectID,
		UserID:    u.ID,
		Role:      common.RoleMember,
		IsActive:  true,
	}).Error)
	return u.ID
}

func countUnreadByType(t *testing.T, db *gorm.DB, recipientID uint64, nType string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&model.Notification{}).
		Where("recipient_id = ? AND type = ? AND is_read = ?", recipientID, nType, false).
		Count(&n).Error)
	return n
}

func TestNotification_TriggerSkipsSelfRecipient(t *testing.T) {
	db := setupNotificationTestDB(t)
	svc := NewNotificationService(db)
	uid := uint64(42)
	require.NoError(t, svc.TriggerNotification(db, "issue_assigned", "t", "m", uid, &uid, nil, nil))
	var count int64
	require.NoError(t, db.Model(&model.Notification{}).Where("recipient_id = ?", uid).Count(&count).Error)
	require.Zero(t, count)
}

func TestCreate_SendsAssignmentToAssigneesNotActor(t *testing.T) {
	db := setupNotificationTestDB(t)
	wsID, projID, actorID := seedTestData(t, db)
	linkWorkspaceIssueTypeToProject(t, db, projID, wsID)
	ensureDefaultIssueState(t, db, projID)
	assigneeID := seedProjectMember(t, db, projID, "assignee_one", "a1@t.com")

	notifSvc := NewNotificationService(db)
	svc := NewIssueService(db, notifSvc, NewWebhookService(db), nil, nil)
	_, err := svc.Create(&request.IssueCreateRequest{
		Name:            "assigned at create",
		AssigneeIDs:     []uint64{assigneeID, actorID},
		DescriptionHTML: "<p></p>",
	}, projID, wsID, actorID)
	require.NoError(t, err)
	require.Equal(t, int64(1), countUnreadByType(t, db, assigneeID, "assignment"))
	require.Zero(t, countUnreadByType(t, db, actorID, "assignment"))
}

func TestUpdate_AssigneeDiffSendsAssignmentNotifications(t *testing.T) {
	db := setupNotificationTestDB(t)
	wsID, projID, actorID := seedTestData(t, db)
	linkWorkspaceIssueTypeToProject(t, db, projID, wsID)
	ensureDefaultIssueState(t, db, projID)
	userB := seedProjectMember(t, db, projID, "peer_b", "b@t.com")
	userC := seedProjectMember(t, db, projID, "peer_c", "c@t.com")

	notifSvc := NewNotificationService(db)
	svc := NewIssueService(db, notifSvc, NewWebhookService(db), nil, nil)
	created, err := svc.Create(&request.IssueCreateRequest{
		Name:            "reassign flow",
		AssigneeIDs:     []uint64{userB},
		DescriptionHTML: "<p></p>",
	}, projID, wsID, actorID)
	require.NoError(t, err)
	_, err = svc.Update(created.ID, &request.IssueUpdateRequest{AssigneeIDs: []uint64{userC}}, actorID)
	require.NoError(t, err)
	require.Equal(t, int64(1), countUnreadByType(t, db, userC, "assignment"))
}

func TestUpdate_StateChangeNotifiesAssigneeAfterAssigneeRefresh(t *testing.T) {
	db := setupNotificationTestDB(t)
	wsID, projID, actorID := seedTestData(t, db)
	linkWorkspaceIssueTypeToProject(t, db, projID, wsID)
	ensureDefaultIssueState(t, db, projID)
	assigneeID := seedProjectMember(t, db, projID, "state_peer", "s@t.com")
	doing := &model.State{Name: "Doing", Color: "#00f", ProjectID: &projID, WorkspaceID: wsID, IsActive: true}
	require.NoError(t, db.Create(doing).Error)

	notifSvc := NewNotificationService(db)
	svc := NewIssueService(db, notifSvc, NewWebhookService(db), nil, nil)
	created, err := svc.Create(&request.IssueCreateRequest{
		Name:            "state notify",
		AssigneeIDs:     []uint64{assigneeID},
		DescriptionHTML: "<p></p>",
	}, projID, wsID, actorID)
	require.NoError(t, err)
	_, err = svc.Update(created.ID, &request.IssueUpdateRequest{StateID: &doing.ID}, actorID)
	require.NoError(t, err)
	require.Equal(t, int64(1), countUnreadByType(t, db, assigneeID, "status"))
}

func TestComment_NotifiesAssigneesAndWatchers(t *testing.T) {
	db := setupNotificationTestDB(t)
	wsID, projID, authorID := seedTestData(t, db)
	linkWorkspaceIssueTypeToProject(t, db, projID, wsID)
	ensureDefaultIssueState(t, db, projID)
	assigneeID := seedProjectMember(t, db, projID, "cmt_asg", "ca@t.com")
	watcherID := seedProjectMember(t, db, projID, "cmt_watch", "cw@t.com")

	notifSvc := NewNotificationService(db)
	issueSvc := NewIssueService(db, notifSvc, NewWebhookService(db), nil, nil)
	commentSvc := NewCommentService(db, notifSvc)
	created, err := issueSvc.Create(&request.IssueCreateRequest{
		Name:            "comment recipients",
		AssigneeIDs:     []uint64{assigneeID},
		DescriptionHTML: "<p></p>",
	}, projID, wsID, authorID)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.IssueWatcher{IssueID: created.ID, UserID: watcherID}).Error)
	_, err = commentSvc.Create(created.ID, authorID, "hello thread", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), countUnreadByType(t, db, assigneeID, "comment"))
	require.Equal(t, int64(1), countUnreadByType(t, db, watcherID, "comment"))
}

func TestComment_MentionNotifiesUser(t *testing.T) {
	db := setupNotificationTestDB(t)
	wsID, projID, authorID := seedTestData(t, db)
	linkWorkspaceIssueTypeToProject(t, db, projID, wsID)
	ensureDefaultIssueState(t, db, projID)
	mentioned := seedProjectMember(t, db, projID, "mentioned_u", "m@t.com")

	notifSvc := NewNotificationService(db)
	issueSvc := NewIssueService(db, notifSvc, NewWebhookService(db), nil, nil)
	commentSvc := NewCommentService(db, notifSvc)
	created, err := issueSvc.Create(&request.IssueCreateRequest{
		Name:            "mention test",
		DescriptionHTML: "<p></p>",
	}, projID, wsID, authorID)
	require.NoError(t, err)
	_, err = commentSvc.Create(created.ID, authorID, "ping @mentioned_u", nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), countUnreadByType(t, db, mentioned, "assignment"))
}

func TestNotification_GetSummaryUnread(t *testing.T) {
	db := setupNotificationTestDB(t)
	_, _, uid := seedTestData(t, db)
	notifSvc := NewNotificationService(db)
	require.NoError(t, notifSvc.TriggerNotification(db, "issue_commented", "t", "m", uid, nil, nil, nil))
	sum, err := notifSvc.GetSummary(uid)
	require.NoError(t, err)
	require.GreaterOrEqual(t, sum.Unread, 1)
	list, err := notifSvc.List(uid, true, 50, 0)
	require.NoError(t, err)
	require.NotEmpty(t, list)
}
