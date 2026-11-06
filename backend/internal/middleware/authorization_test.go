package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestRequirePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.WorkspaceMember{}, &model.Project{}, &model.ProjectMember{},
		&model.Issue{}, &model.Permission{}, &model.Role{}))

	perms := map[string]*model.Permission{}
	for _, code := range []string{"issue:create", "issue:edit", "issue:delete", "workflow:manage"} {
		p := &model.Permission{Code: code, Name: code, Resource: "x", Action: "x"}
		require.NoError(t, db.Create(p).Error)
		perms[code] = p
	}
	role := func(level int, system bool, codes ...string) {
		r := model.Role{Name: "r", Level: level, IsSystem: system}
		for _, code := range codes {
			r.Permissions = append(r.Permissions, *perms[code])
		}
		require.NoError(t, db.Create(&r).Error)
	}
	role(20, true, "issue:create", "issue:edit", "issue:delete", "workflow:manage")
	role(15, true, "issue:edit")
	role(5, true)
	role(15, false, "issue:delete") // a custom role at the member level must not leak its permissions

	own := model.Project{Name: "own", Identifier: "OWN", WorkspaceID: 10}
	foreign := model.Project{Name: "foreign", Identifier: "FOR", WorkspaceID: 20}
	require.NoError(t, db.Create(&own).Error)
	require.NoError(t, db.Create(&foreign).Error)

	const admin, member, guestWithProjectRole, guest, outsider, root = 1, 2, 3, 4, 5, 6
	for _, m := range []model.WorkspaceMember{
		{WorkspaceID: 10, UserID: admin, Role: 20, IsActive: true},
		{WorkspaceID: 10, UserID: member, Role: 15, IsActive: true},
		{WorkspaceID: 10, UserID: guestWithProjectRole, Role: 5, IsActive: true},
		{WorkspaceID: 10, UserID: guest, Role: 5, IsActive: true},
	} {
		require.NoError(t, db.Create(&m).Error)
	}
	require.NoError(t, db.Create(&model.ProjectMember{ProjectID: own.ID, UserID: guestWithProjectRole, Role: 15, IsActive: true}).Error)

	ownIssue := model.Issue{Name: "i", ProjectID: own.ID, WorkspaceID: 10, StateID: 1}
	foreignIssue := model.Issue{Name: "i", ProjectID: foreign.ID, WorkspaceID: 20, StateID: 1}
	deletedIssue := model.Issue{Name: "d", ProjectID: own.ID, WorkspaceID: 10, StateID: 1}
	for _, i := range []*model.Issue{&ownIssue, &foreignIssue, &deletedIssue} {
		require.NoError(t, db.Create(i).Error)
	}
	require.NoError(t, db.Delete(&deletedIssue).Error)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		id := uint64(0)
		for name, v := range map[string]uint64{"admin": admin, "member": member, "gpr": guestWithProjectRole,
			"guest": guest, "outsider": outsider, "root": root} {
			if c.GetHeader("X-User") == name {
				id = v
			}
		}
		c.Set("currentUser", &model.User{BaseModel: model.BaseModel{ID: id}, IsSuperuser: id == root})
	})
	echoBody := func(c *gin.Context) {
		b, _ := io.ReadAll(c.Request.Body)
		c.String(http.StatusOK, string(b))
	}
	r.PUT("/issues/:issueId", RequirePermission(db, "issue:edit", "project"), echoBody)
	r.DELETE("/issues/:issueId", RequirePermission(db, "issue:delete", "project"), echoBody)
	r.POST("/issues/:issueId/restore", RequirePermission(db, "issue:edit", "project"), echoBody)
	r.POST("/issues/bulk/update", RequirePermission(db, "issue:edit", "project"), echoBody)
	r.POST("/issues/bulk/delete", RequirePermission(db, "issue:delete", "project"), echoBody)
	r.POST("/issues/bulk/move", RequirePermission(db, "issue:edit", "project"), echoBody)
	r.POST("/issues", RequirePermission(db, "issue:create", "project"), echoBody)
	r.PUT("/projects/:projectId/sla", RequirePermission(db, "workflow:manage", "project"), echoBody)

	do := func(user, method, path, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("X-User", user)
		r.ServeHTTP(w, req)
		return w
	}
	status := func(user, method, path, body string) int { return do(user, method, path, body).Code }
	issue := "/issues/" + uitoa(ownIssue.ID)

	assert.Equal(t, 200, status("member", "PUT", issue, ""))
	assert.Equal(t, 200, status("gpr", "PUT", issue, ""), "the project role lifts a workspace guest to member")
	assert.Equal(t, 403, status("guest", "PUT", issue, ""))
	assert.Equal(t, 403, status("outsider", "PUT", issue, ""))
	assert.Equal(t, 200, status("root", "PUT", issue, ""))

	assert.Equal(t, 403, status("member", "DELETE", issue, ""), "only the system role's permissions count")
	assert.Equal(t, 200, status("admin", "DELETE", issue, ""))
	assert.Equal(t, 403, status("admin", "DELETE", "/issues/"+uitoa(foreignIssue.ID), ""))
	assert.Equal(t, 200, status("guest", "DELETE", "/issues/99999", ""), "unknown resources fall through to the handler")

	restore := "/issues/" + uitoa(deletedIssue.ID) + "/restore"
	assert.Equal(t, 403, status("guest", "POST", restore, ""), "deleted work items still resolve their project")
	assert.Equal(t, 200, status("member", "POST", restore, ""))

	assert.Equal(t, 200, status("member", "POST", "/issues/bulk/update?project_id="+uitoa(own.ID), `{"issue_ids":[]}`))
	assert.Equal(t, 403, status("guest", "POST", "/issues/bulk/update?project_id="+uitoa(own.ID), `{"issue_ids":[]}`))
	body := `{"issue_ids":[` + uitoa(ownIssue.ID) + `]}`
	w := do("admin", "POST", "/issues/bulk/delete", body)
	assert.Equal(t, 200, w.Code)
	assert.Equal(t, body, w.Body.String(), "the handler still sees the body")
	assert.Equal(t, 403, status("member", "POST", "/issues/bulk/delete", body))
	assert.Equal(t, 403, status("admin", "POST", "/issues/bulk/delete",
		`{"issue_ids":[`+uitoa(ownIssue.ID)+`,`+uitoa(foreignIssue.ID)+`]}`), "every touched project is checked")
	assert.Equal(t, 403, status("member", "POST", "/issues/bulk/move",
		`{"issue_ids":[`+uitoa(ownIssue.ID)+`],"target_project_id":`+uitoa(foreign.ID)+`}`), "the target project is checked")

	createIssue := "/issues?project_id=" + uitoa(own.ID)
	assert.Equal(t, 200, status("admin", "POST", createIssue, ""))
	assert.Equal(t, 403, status("member", "POST", createIssue, ""))
	assert.Equal(t, 403, status("guest", "POST", createIssue, ""))

	assert.Equal(t, 200, status("admin", "PUT", "/projects/"+uitoa(own.ID)+"/sla", ""))
	assert.Equal(t, 403, status("member", "PUT", "/projects/"+uitoa(own.ID)+"/sla", ""))

	for _, tc := range []struct {
		user  uint64
		codes []string
		level int
	}{
		{admin, []string{"issue:create", "issue:delete", "issue:edit", "workflow:manage"}, 20},
		{member, []string{"issue:edit"}, 15},
		{guestWithProjectRole, []string{"issue:edit"}, 15},
		{guest, []string{}, 5},
		{outsider, []string{}, 0},
	} {
		codes, level := ProjectPermissions(db, &model.User{BaseModel: model.BaseModel{ID: tc.user}}, own.ID)
		assert.Equal(t, tc.codes, codes, "user %d", tc.user)
		assert.Equal(t, tc.level, level, "user %d", tc.user)
	}
	codes, _ := ProjectPermissions(db, &model.User{BaseModel: model.BaseModel{ID: root}, IsSuperuser: true}, own.ID)
	assert.Len(t, codes, 4)
}
