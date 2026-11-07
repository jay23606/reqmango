package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/reqmango/backend/internal/middleware"
	"github.com/reqmango/backend/internal/model"
	"github.com/reqmango/backend/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// attachmentTestRouter builds routes matching production for attachments:
// list/get/download open; post/delete gated by RequirePermission(issue:edit).
func attachmentTestRouter(t *testing.T) (*gin.Engine, *gorm.DB, model.Issue, model.User, model.User) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.Workspace{}, &model.WorkspaceMember{}, &model.Project{}, &model.ProjectMember{},
		&model.Issue{}, &model.State{}, &model.User{}, &model.Permission{}, &model.Role{},
		&model.RolePermission{}, &model.Attachment{},
	))

	editPerm := model.Permission{Code: "issue:edit", Name: "issue:edit", Resource: "issue", Action: "edit"}
	require.NoError(t, db.Create(&editPerm).Error)

	memberRole := model.Role{Name: "member", Level: 15, IsSystem: true, Permissions: []model.Permission{editPerm}}
	require.NoError(t, db.Create(&memberRole).Error)

	guestRole := model.Role{Name: "guest", Level: 5, IsSystem: true}
	require.NoError(t, db.Create(&guestRole).Error)

	ws := model.Workspace{Name: "acme", Slug: "acme"}
	require.NoError(t, db.Create(&ws).Error)
	project := model.Project{Name: "core", Identifier: "CORE", WorkspaceID: ws.ID}
	require.NoError(t, db.Create(&project).Error)
	st := model.State{Name: "Backlog", Group: "backlog", IsDefault: true, ProjectID: &project.ID, WorkspaceID: ws.ID}
	require.NoError(t, db.Create(&st).Error)

	editor := model.User{Email: "editor@example.com", Username: "editor", IsActive: true}
	guest := model.User{Email: "guest@example.com", Username: "guest", IsActive: true}
	require.NoError(t, db.Create(&editor).Error)
	require.NoError(t, db.Create(&guest).Error)

	require.NoError(t, db.Create(&model.WorkspaceMember{WorkspaceID: ws.ID, UserID: editor.ID, Role: 15, IsActive: true}).Error)
	require.NoError(t, db.Create(&model.WorkspaceMember{WorkspaceID: ws.ID, UserID: guest.ID, Role: 5, IsActive: true}).Error)

	require.NoError(t, db.Create(&model.ProjectMember{ProjectID: project.ID, UserID: editor.ID, Role: 15, IsActive: true}).Error)
	require.NoError(t, db.Create(&model.ProjectMember{ProjectID: project.ID, UserID: guest.ID, Role: 5, IsActive: true}).Error)

	issue := model.Issue{Name: "Task", ProjectID: project.ID, WorkspaceID: ws.ID, StateID: st.ID}
	require.NoError(t, db.Create(&issue).Error)

	h := NewAttachmentHandler(service.NewAttachmentService(db))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		switch c.GetHeader("X-Test-User") {
		case "editor":
			c.Set("currentUser", &editor)
		case "guest":
			c.Set("currentUser", &guest)
		}
		c.Next()
	})

	r.GET("/issues/:issueId/attachments", h.ListByIssue)
	r.POST("/issues/:issueId/attachments", middleware.RequirePermission(db, "issue:edit", "project"), h.Create)
	r.GET("/issues/:issueId/attachments/:attachmentId/download", h.Download)
	r.DELETE("/issues/:issueId/attachments/:attachmentId", middleware.RequirePermission(db, "issue:edit", "project"), h.Delete)

	return r, db, issue, editor, guest
}

func attachmentIssuePath(issueID uint64, parts ...string) string {
	s := "/issues/" + strconv.FormatUint(issueID, 10)
	for _, p := range parts {
		s += "/" + p
	}
	return s
}

func TestAttachmentHandler_UploadOver10MBRejectedByValidator(t *testing.T) {
	r, _, issue, _, _ := attachmentTestRouter(t)

	large := bytes.Repeat([]byte("x"), 10*1024*1024+1)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", "note.txt")
	require.NoError(t, err)
	_, err = part.Write(large)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, attachmentIssuePath(issue.ID, "attachments"), &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Test-User", "editor")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	assert.Contains(t, payload["error"], "10MB")
}

func TestAttachmentHandler_DeleteThenDownload404(t *testing.T) {
	r, db, issue, editor, _ := attachmentTestRouter(t)

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "f.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello"), 0o600))
	uid := editor.ID
	att := model.Attachment{
		Name:       "note.txt",
		FilePath:   path,
		FileSize:   5,
		MimeType:   "text/plain",
		IssueID:    issue.ID,
		UploaderID: &uid,
	}
	require.NoError(t, db.Create(&att).Error)

	dlURL := attachmentIssuePath(issue.ID, "attachments", strconv.FormatUint(att.ID, 10), "download")
	req := httptest.NewRequest(http.MethodGet, dlURL, nil)
	req.Header.Set("X-Test-User", "editor")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())

	delURL := attachmentIssuePath(issue.ID, "attachments", strconv.FormatUint(att.ID, 10))
	reqDel := httptest.NewRequest(http.MethodDelete, delURL, nil)
	reqDel.Header.Set("X-Test-User", "editor")
	recDel := httptest.NewRecorder()
	r.ServeHTTP(recDel, reqDel)
	assert.Equal(t, http.StatusOK, recDel.Code)

	req404 := httptest.NewRequest(http.MethodGet, dlURL, nil)
	req404.Header.Set("X-Test-User", "editor")
	rec404 := httptest.NewRecorder()
	r.ServeHTTP(rec404, req404)
	assert.Equal(t, http.StatusNotFound, rec404.Code)
}

func TestAttachmentHandler_PostAndDeleteRequireIssueEdit(t *testing.T) {
	r, db, issue, _, guest := attachmentTestRouter(t)

	small := bytes.Repeat([]byte("z"), 100)
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("file", "a.txt")
	require.NoError(t, err)
	_, err = part.Write(small)
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, attachmentIssuePath(issue.ID, "attachments"), &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("X-Test-User", "guest")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ro.txt")
	require.NoError(t, os.WriteFile(path, []byte("x"), 0o600))
	gid := guest.ID
	att := model.Attachment{
		Name:       "ro.txt",
		FilePath:   path,
		FileSize:   1,
		MimeType:   "text/plain",
		IssueID:    issue.ID,
		UploaderID: &gid,
	}
	require.NoError(t, db.Create(&att).Error)

	delURL := attachmentIssuePath(issue.ID, "attachments", strconv.FormatUint(att.ID, 10))
	reqDel := httptest.NewRequest(http.MethodDelete, delURL, nil)
	reqDel.Header.Set("X-Test-User", "guest")
	recDel := httptest.NewRecorder()
	r.ServeHTTP(recDel, reqDel)
	assert.Equal(t, http.StatusForbidden, recDel.Code)
}

func TestAttachmentHandler_ListByIssue_AllowsGuest(t *testing.T) {
	r, db, issue, _, _ := attachmentTestRouter(t)
	require.NoError(t, db.Create(&model.Attachment{
		Name:     "a.txt",
		FilePath: filepath.Join(t.TempDir(), "a.txt"),
		FileSize: 1, MimeType: "text/plain",
		IssueID: issue.ID,
	}).Error)

	req := httptest.NewRequest(http.MethodGet, attachmentIssuePath(issue.ID, "attachments"), nil)
	req.Header.Set("X-Test-User", "guest")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	var list []map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	assert.Len(t, list, 1)
}
