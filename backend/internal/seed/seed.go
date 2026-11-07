package seed

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/model"
	"github.com/reqmango/backend/internal/randutil"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Options selects which seed phases run.
//
// Demo and E2E data both create accounts with passwords that are published in
// the repository, so neither may run on an internet-reachable deployment
// without the operator asking for it.
type Options struct {
	Demo bool
	E2E  bool
}

func strPtr(s string) *string  { return &s }
func uintPtr(u uint64) *uint64 { return &u }

func SeedAll(db *gorm.DB, opts Options) {
	fmt.Println("=== Starting data initialization ===")

	// System data: needed for a working install, safe in every environment.
	SeedRBACData(db)
	SeedIssueTypesForAllWorkspaces(db)

	// Demo data ships accounts whose passwords are published in the README, so
	// it is opt-in (SEED_DEMO=true). See EnsureBootstrapAdmin for the
	// no-demo path.
	if opts.Demo {
		SeedDemoData(db)
		SeedDemoLoginUser(db)
	} else {
		fmt.Println("Demo data disabled (SEED_DEMO=false): no sample workspaces or demo accounts created")
	}

	SeedConfigData(db)
	SeedRelationTypesForAllWorkspaces(db)
	SeedReleasesForAllProjects(db)
	BackfillIssueTypeIDs(db)
	FixIssueTypeHierarchy(db)
	SeedSearchTemplates(db)

	if opts.Demo {
		SeedAutomationRulesForAllWorkspaces(db)
		SeedWebhookDemoExecutionLogs(db)
		SeedAIData(db)
		fmt.Println("[SECURITY WARNING] Demo data was seeded with published passwords " +
			"(admin@reqmango.com / demo@example.com, password demo1234). " +
			"Never expose this instance to the internet; run real deployments with SEED_DEMO=false.")
	}

	// Test fixtures create a superuser with a published password too.
	if opts.E2E {
		SeedE2EFixtures(db)
	}

	fmt.Println("=== Data initialization complete ===")
}

// EnsureBootstrapAdmin guarantees a fresh deployment has a way in when the demo
// accounts are disabled (the default).
//
// It only acts when no superuser exists, so restarts are no-ops and an existing
// operator account is never modified. An empty password generates a random one
// and prints it once; that keeps a known credential out of every install. The
// return value reports whether an account was created.
func EnsureBootstrapAdmin(db *gorm.DB, email, password string) bool {
	var superusers int64
	if err := db.Model(&model.User{}).Where("is_superuser = ?", true).Count(&superusers).Error; err != nil {
		fmt.Printf("[BOOTSTRAP] WARNING: cannot check for an existing superuser: %v\n", err)
		return false
	}
	if superusers > 0 {
		return false
	}

	email = strings.TrimSpace(email)
	if email == "" {
		email = "admin@reqmango.com"
	}

	// Never promote a pre-registered account: whoever signs up first would
	// otherwise receive admin. Tell the operator how to recover instead.
	var existing model.User
	if err := db.Where("email = ?", email).First(&existing).Error; err == nil {
		fmt.Printf("[BOOTSTRAP] A user with email %s already exists but no superuser is configured.\n", email)
		fmt.Println("[BOOTSTRAP] Refusing to promote it automatically. Set ADMIN_EMAIL to another address and restart.")
		return false
	}

	generated := password == ""
	if generated {
		password = randomPassword()
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("[BOOTSTRAP] WARNING: cannot hash the bootstrap password: %v\n", err)
		return false
	}

	username := strings.SplitN(email, "@", 2)[0]
	user := model.User{
		Email:        email,
		Username:     username,
		DisplayName:  "Administrator",
		PasswordHash: string(hash),
		IsActive:     true,
		IsSuperuser:  true,
	}
	if err := db.Create(&user).Error; err != nil {
		// The username is derived from the email local part, which may already
		// belong to someone else. Retry once with a random suffix rather than
		// leaving the deployment without an admin.
		user.Username = username + "-" + randomToken(3)
		if err := db.Create(&user).Error; err != nil {
			fmt.Printf("[BOOTSTRAP] WARNING: cannot create bootstrap admin %s: %v\n", email, err)
			return false
		}
	}

	fmt.Println("============================================================")
	fmt.Println("Bootstrap superuser created")
	fmt.Printf("  email:    %s\n", email)
	if generated {
		fmt.Printf("  password: %s\n", password)
		fmt.Println("  Shown once. Store it now and change it after signing in.")
	} else {
		fmt.Println("  password: (value of ADMIN_PASSWORD)")
	}
	fmt.Println("============================================================")
	return true
}

// randomPassword returns a 24-character URL-safe random password.
func randomPassword() string {
	// 18 bytes encode to exactly 24 base64 characters.
	return randomToken(18)
}

func randomToken(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand failing means the process cannot be trusted with secrets.
		panic(fmt.Sprintf("seed: cannot generate a random token: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// SeedReleasesForAllProjects creates default releases for every project that has none.
func SeedReleasesForAllProjects(db *gorm.DB) {
	var projects []model.Project
	if db.Find(&projects).Error != nil {
		return
	}
	created := 0
	for _, proj := range projects {
		var count int64
		db.Model(&model.Release{}).Where("project_id = ?", proj.ID).Count(&count)
		if count > 0 {
			continue
		}
		releaseDefs := []struct {
			name, version, description, status string
		}{
			{"v1.0.0", "v1.0.0", "首个正式版本发布", "released"},
			{"v1.1.0", "v1.1.0", "功能增强与Bug修复版本", "in_progress"},
			{"v2.0.0", "v2.0.0", "重大架构升级版本", "planned"},
		}
		for _, rd := range releaseDefs {
			db.Create(&model.Release{
				Name:        rd.name,
				Version:     rd.version,
				Description: rd.description,
				Status:      rd.status,
				ProjectID:   proj.ID,
			})
			created++
		}
	}
	if created > 0 {
		fmt.Printf("Seeded %d releases across %d projects\n", created, len(projects))
	}
}

// FixIssueTypeHierarchy clears ParentTypeID on Bug/Task issue types so they can
// be children of any lower-level type (not just Feature). This runs as a migration
// for existing databases that were seeded with the old strict hierarchy.
func FixIssueTypeHierarchy(db *gorm.DB) {
	if err := db.Model(&model.IssueType{}).
		Where("level = ? AND name IN ?", 2, []string{"Bug", "Task"}).
		Update("parent_type_id", nil).Error; err != nil {
		fmt.Printf("  WARN: failed to fix issue type hierarchy: %v\n", err)
	} else {
		fmt.Println("Fixed IssueType hierarchy: Bug/Task can now be children of any lower-level type")
	}
}

// BackfillIssueTypeIDs assigns issue_type_id to existing issues that lack one,
// by inferring the type from the issue title.
func BackfillIssueTypeIDs(db *gorm.DB) {
	var missingCount int64
	db.Model(&model.Issue{}).Where("issue_type_id IS NULL").Count(&missingCount)
	if missingCount == 0 {
		fmt.Println("All issues already have issue_type_id, skipping backfill")
		return
	}
	fmt.Printf("--- Backfilling issue_type_id for %d issues ---\n", missingCount)

	// Build workspaceID -> {typeName -> issueType} map
	typeMap := make(map[uint64]map[string]model.IssueType)
	var allTypes []model.IssueType
	db.Find(&allTypes)
	for _, t := range allTypes {
		if _, ok := typeMap[t.WorkspaceID]; !ok {
			typeMap[t.WorkspaceID] = make(map[string]model.IssueType)
		}
		typeMap[t.WorkspaceID][t.Name] = t
	}

	// Fetch issues lacking issue_type_id
	var issues []model.Issue
	db.Where("issue_type_id IS NULL").Find(&issues)

	updated := 0
	for _, issue := range issues {
		typeName := inferIssueTypeFromTitle(issue.Name)
		wsTypes, ok := typeMap[issue.WorkspaceID]
		if !ok {
			continue
		}
		it, ok := wsTypes[typeName]
		if !ok {
			// Fallback: use default type (Epic) or first available
			if def, ok := wsTypes["Epic"]; ok {
				it = def
			} else {
				continue
			}
		}
		issueTypeID := it.ID
		if err := db.Model(&model.Issue{}).Where("id = ?", issue.ID).Update("issue_type_id", issueTypeID).Error; err != nil {
			fmt.Printf("  WARN: failed to update issue %d: %v\n", issue.ID, err)
			continue
		}
		updated++
	}
	fmt.Printf("  Backfilled issue_type_id for %d issues\n", updated)
}

// inferIssueTypeFromTitle guesses the issue type name based on title keywords.
func inferIssueTypeFromTitle(title string) string {
	switch {
	case containsAny(title, "Epic:"):
		return "Epic"
	case containsAny(title, "技术调研:"):
		return "Spike"
	case containsAny(title, "作为"):
		return "Story"
	case containsAny(title, "修复", "Bug", "bug", "崩溃", "异常", "报错", "失效", "缺失", "错位", "无法", "未保存", "未刷新", "未更新", "未清理", "不显示", "不更新"):
		return "Bug"
	case containsAny(title, "编写", "优化", "配置", "升级", "部署", "分析", "调整", "测试", "整理", "创建", "修复", "演练", "扫描"):
		return "Task"
	default:
		return "Feature"
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if len(sub) == 0 {
			continue
		}
		if idx := indexOf(s, sub); idx >= 0 {
			return true
		}
	}
	return false
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func SeedDemoData(db *gorm.DB) {
	var count int64
	db.Model(&model.User{}).Where("username = ?", "admin").Count(&count)
	if count > 0 {
		fmt.Println("Demo data already exists, skipping seed")
		return
	}

	rng := randutil.New()

	// ============================================================
	// 1. USERS — 25 diverse users
	// ============================================================
	hash, _ := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)

	type userDef struct {
		email, username, displayName string
		isSuperuser                  bool
	}
	userDefs := []userDef{
		{"admin@reqmango.com", "admin", "系统管理员", true},
		{"zhangsan@reqmango.com", "zhangsan", "张三 (产品经理)", false},
		{"lisi@reqmango.com", "lisi", "李四 (技术负责人)", false},
		{"wangwu@reqmango.com", "wangwu", "王五 (前端开发)", false},
		{"zhaoliu@reqmango.com", "zhaoliu", "赵六 (后端开发)", false},
		{"sunqi@reqmango.com", "sunqi", "孙七 (测试工程师)", false},
		{"zhouba@reqmango.com", "zhouba", "周八 (UI设计师)", false},
		{"wujiu@reqmango.com", "wujiu", "吴九 (运维工程师)", false},
		{"zhengshi@reqmango.com", "zhengshi", "郑十 (数据工程师)", false},
		{"liuxi@reqmango.com", "liuxi", "刘喜 (安全工程师)", false},
		{"chenyu@reqmango.com", "chenyu", "陈雨 (前端开发)", false},
		{"huanglei@reqmango.com", "huanglei", "黄磊 (后端开发)", false},
		{"linfen@reqmango.com", "linfen", "林芬 (产品运营)", false},
		{"yangguang@reqmango.com", "yangguang", "杨光 (技术架构师)", false},
		{"xumin@reqmango.com", "xumin", "许敏 (QA Lead)", false},
		{"heping@reqmango.com", "heping", "何平 (项目经理)", false},
		{"gaoyuan@reqmango.com", "gaoyuan", "高原 (全栈开发)", false},
		{"bailu@reqmango.com", "bailu", "白露 (UX研究员)", false},
		{"shenwei@reqmango.com", "shenwei", "沈伟 (DevOps)", false},
		{"hanyan@reqmango.com", "hanyan", "韩燕 (数据分析师)", false},
		{"tangjie@reqmango.com", "tangjie", "唐杰 (移动端开发)", false},
		{"maxin@reqmango.com", "maxin", "马欣 (技术文档)", false},
		{"fangyi@reqmango.com", "fangyi", "方毅 (后端开发)", false},
		{"luyao@reqmango.com", "luyao", "陆瑶 (产品设计)", false},
		{"jiangwei@reqmango.com", "jiangwei", "姜伟 (SRE)", false},
	}

	users := make([]model.User, len(userDefs))
	for i, ud := range userDefs {
		users[i] = model.User{
			Email:        ud.email,
			Username:     ud.username,
			DisplayName:  ud.displayName,
			PasswordHash: string(hash),
			IsActive:     true,
			IsSuperuser:  ud.isSuperuser,
		}
		if err := db.Create(&users[i]).Error; err != nil {
			fmt.Printf("  WARN: failed to create user %s: %v\n", ud.username, err)
		}
	}
	adminUser := users[0]
	fmt.Printf("Created %d users (password: demo1234)\n", len(users))

	// ============================================================
	// 2. WORKSPACES
	// ============================================================
	type wsDef struct {
		name, slug, timezone string
		ownerIdx             int
	}
	wsDefs := []wsDef{
		{"ReqMango 产品研发", "reqmango-dev", "Asia/Shanghai", 0},
		{"客户项目交付中心", "client-delivery", "Asia/Shanghai", 1},
		{"内部基础设施", "infra", "Asia/Shanghai", 4},
	}

	workspaces := make([]model.Workspace, len(wsDefs))
	for i, wd := range wsDefs {
		workspaces[i] = model.Workspace{
			Name:     wd.name,
			Slug:     wd.slug,
			Timezone: wd.timezone,
			OwnerID:  users[wd.ownerIdx].ID,
		}
		if err := db.Create(&workspaces[i]).Error; err != nil {
			fmt.Printf("  WARN: failed to create workspace %s: %v\n", wd.slug, err)
		}
	}
	fmt.Printf("Created %d workspaces\n", len(workspaces))

	// Add members to each workspace
	for _, ws := range workspaces {
		for _, u := range users {
			role := common.RoleMember
			if u.IsSuperuser || u.ID == ws.OwnerID {
				role = common.RoleAdmin
			} else if rng.Intn(100) < 15 {
				role = common.RoleGuest
			}
			db.Create(&model.WorkspaceMember{
				WorkspaceID: ws.ID, UserID: u.ID, Role: role, IsActive: true,
			})
		}
	}
	fmt.Println("Added members to all workspaces")

	// Pre-create relation types so issue relations can reference them
	for _, ws := range workspaces {
		var relCount int64
		db.Model(&model.RelationType{}).Where("workspace_id = ?", ws.ID).Count(&relCount)
		if relCount == 0 {
			relTypes := []model.RelationType{
				{Name: "阻塞", InwardName: "被阻塞于", OutwardName: "阻塞", WorkspaceID: ws.ID},
				{Name: "关联", InwardName: "关联到", OutwardName: "被关联于", WorkspaceID: ws.ID},
				{Name: "重复", InwardName: "重复于", OutwardName: "被重复于", WorkspaceID: ws.ID},
				{Name: "父子", InwardName: "子任务", OutwardName: "父任务", WorkspaceID: ws.ID},
				{Name: "依赖", InwardName: "依赖于", OutwardName: "被依赖于", WorkspaceID: ws.ID},
			}
			for _, r := range relTypes {
				db.Create(&r)
			}
		}
	}
	fmt.Println("Created relation types for all workspaces")

	// ============================================================
	// 3. PROJECTS
	// ============================================================
	type projDef struct {
		name, identifier, desc, color string
	}
	projectDefs := []struct {
		wsIdx int
		projs []projDef
	}{
		{0, []projDef{
			{"ReqMango 核心平台", "CORE", "产品核心功能模块开发", "#6366F1"},
			{"移动端 App", "MOBILE", "iOS/Android 移动端应用", "#8B5CF6"},
			{"AI 智能助手", "AI", "AI驱动的项目管理智能助手", "#10B981"},
			{"开放平台 & API", "OPENAPI", "第三方集成与开放API平台", "#F59E0B"},
		}},
		{1, []projDef{
			{"XX银行风控系统", "BANK-RMS", "XX银行智能风控管理平台", "#EF4444"},
			{"电商平台重构", "ESHOP", "大型B2C电商平台技术重构", "#3B82F6"},
			{"智慧城市大屏", "SMARTCITY", "智慧城市数据可视化大屏项目", "#06B6D4"},
		}},
		{2, []projDef{
			{"CI/CD 平台建设", "CICD", "企业级CI/CD流水线平台", "#84CC16"},
			{"监控告警系统", "MONITOR", "全栈监控与智能告警平台", "#F97316"},
			{"安全合规平台", "SECCOMP", "安全审计与合规检查自动化", "#DC2626"},
		}},
	}

	projects := make([]model.Project, 0)
	for _, pds := range projectDefs {
		ws := workspaces[pds.wsIdx]
		for _, pd := range pds.projs {
			desc := pd.desc
			proj := model.Project{
				Name:        pd.name,
				Identifier:  pd.identifier,
				Description: &desc,
				WorkspaceID: ws.ID,
				Color:       pd.color,
			}
			if err := db.Create(&proj).Error; err != nil {
				fmt.Printf("  WARN: failed to create project %s: %v\n", pd.identifier, err)
				continue
			}
			projects = append(projects, proj)

			// Project members (subset)
			memberCount := rng.Intn(8) + 5
			perm := rng.Perm(len(users))
			for j := 0; j < memberCount && j < len(perm); j++ {
				u := users[perm[j]]
				pRole := common.RoleMember
				if u.IsSuperuser {
					pRole = common.RoleAdmin
				}
				db.Create(&model.ProjectMember{
					ProjectID: proj.ID, UserID: u.ID, Role: pRole, IsActive: true,
				})
			}
		}
	}
	fmt.Printf("Created %d projects with members\n", len(projects))

	// ============================================================
	// 4. STATES, CYCLES, MODULES, LABELS per project
	// ============================================================
	type stateDef struct {
		name, color, group string
	}
	defaultStates := []stateDef{
		{"待处理 (Backlog)", "#6B7280", "backlog"},
		{"待办 (Todo)", "#3B82F6", "unstarted"},
		{"进行中 (In Progress)", "#F59E0B", "started"},
		{"评审中 (In Review)", "#8B5CF6", "started"},
		{"已完成 (Done)", "#10B981", "completed"},
		{"已取消 (Cancelled)", "#EF4444", "cancelled"},
	}

	allStates := make(map[uint64][]model.State)
	allCycles := make(map[uint64][]model.Cycle)
	allModules := make(map[uint64][]model.Module)
	allLabels := make(map[uint64][]model.Label)
	// allIssueTypes maps workspaceID -> {typeName -> IssueType}
	allIssueTypes := make(map[uint64]map[string]model.IssueType)

	moduleTemplates := [][]string{
		{"用户管理", "权限系统", "消息通知", "数据报表", "系统配置", "日志审计", "API网关", "任务调度"},
		{"商品管理", "订单系统", "支付模块", "库存管理", "物流追踪", "营销活动", "会员中心", "评价系统", "客服工单"},
		{"数据采集", "实时计算", "离线分析", "可视化大屏", "预警规则", "数据导出", "模型训练", "特征工程"},
		{"代码仓库", "构建流水线", "制品管理", "部署编排", "环境管理", "质量扫描", "安全检测"},
		{"前端框架", "组件库", "状态管理", "路由设计", "国际化", "主题定制", "性能监控", "错误追踪"},
	}

	labelTemplates := []struct{ name, color string }{
		{"前端", "#3B82F6"}, {"后端", "#22C55E"}, {"文档", "#F59E0B"},
		{"DevOps", "#8B5CF6"}, {"安全", "#EF4444"}, {"性能优化", "#F97316"},
		{"Bug", "#DC2626"}, {"增强", "#06B6D4"}, {"测试", "#84CC16"},
		{"设计", "#A855F7"}, {"需求", "#EC4899"}, {"架构", "#0EA5E9"},
		{"紧急", "#FF0000"}, {"技术债", "#78716C"}, {"用户体验", "#14B8A6"},
	}

	for _, proj := range projects {
		wsID := proj.WorkspaceID
		var ws model.Workspace
		for _, w := range workspaces {
			if w.ID == wsID {
				ws = w
				break
			}
		}

		// Ensure issue types exist for this workspace (create once, cache by name)
		if _, ok := allIssueTypes[ws.ID]; !ok {
			typeMap := make(map[string]model.IssueType)
			var existingTypes []model.IssueType
			db.Where("workspace_id = ?", ws.ID).Find(&existingTypes)
			for _, t := range existingTypes {
				typeMap[t.Name] = t
			}
			// Create missing standard types
			typeDefs := []struct {
				name, color, icon, desc string
				level                   int
				isDefault               bool
				sequence                int
			}{
				{"Epic", "#8B5CF6", "layers", "顶层史诗级工作项", 0, true, 1},
				{"Feature", "#6366F1", "star", "功能特性", 1, false, 2},
				{"Bug", "#EF4444", "bug", "缺陷/问题", 2, false, 3},
				{"Task", "#10B981", "check-circle", "开发任务", 2, false, 4},
				{"Story", "#F59E0B", "bookmark", "用户故事", 1, false, 5},
				{"Spike", "#06B6D4", "zap", "技术调研/探索", 1, false, 6},
			}
			var epicID uint64
			for _, td := range typeDefs {
				if existing, ok := typeMap[td.name]; ok {
					if td.name == "Epic" {
						epicID = existing.ID
					}
					continue
				}
				it := model.IssueType{
					Name:        td.name,
					Color:       td.color,
					Icon:        td.icon,
					Description: td.desc,
					Level:       td.level,
					IsDefault:   td.isDefault,
					Sequence:    td.sequence,
					WorkspaceID: ws.ID,
				}
				// Only Feature/Story/Spike (Level 1) point to Epic as their parent type.
				// Bug/Task (Level 2) have no fixed ParentTypeID — they can be children
				// of any type with a lower Level (Epic, Feature, Story, Spike).
				if td.level == 1 {
					if epicID != 0 {
						it.ParentTypeID = &epicID
					}
				}
				if err := db.Create(&it).Error; err != nil {
					fmt.Printf("  WARN: failed to create issue type %s: %v\n", td.name, err)
					continue
				}
				typeMap[td.name] = it
				if td.name == "Epic" {
					epicID = it.ID
				}
			}
			allIssueTypes[ws.ID] = typeMap
		}

		// States
		states := make([]model.State, len(defaultStates))
		for i, sd := range defaultStates {
			projID := proj.ID
			states[i] = model.State{
				Name: sd.name, Color: sd.color, Group: sd.group,
				Sequence: i + 1, IsDefault: i == 0,
				ProjectID: &projID, WorkspaceID: ws.ID,
			}
			db.Create(&states[i])
		}
		allStates[proj.ID] = states

		// Cycles — 6 per project
		baseDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local)
		cycleNames := []string{"Sprint 1", "Sprint 2", "Sprint 3", "Sprint 4", "Sprint 5", "Sprint 6"}
		cycles := make([]model.Cycle, len(cycleNames))
		for i, name := range cycleNames {
			start := baseDate.AddDate(0, 0, i*14)
			end := start.AddDate(0, 0, 13)
			var completedAt *time.Time
			if i < 4 {
				completedAt = &end
			}
			cycles[i] = model.Cycle{
				Name:        name,
				StartDate:   start,
				EndDate:     &end,
				CompletedAt: completedAt,
				ProjectID:   proj.ID,
				WorkspaceID: ws.ID,
			}
			db.Create(&cycles[i])
		}
		allCycles[proj.ID] = cycles

		// Modules
		moduleSet := moduleTemplates[rng.Intn(len(moduleTemplates))]
		modules := make([]model.Module, len(moduleSet))
		for i, name := range moduleSet {
			projectIDPtr := proj.ID
			modules[i] = model.Module{
				Name: name, ProjectID: &projectIDPtr, WorkspaceID: ws.ID,
			}
			db.Create(&modules[i])
		}
		allModules[proj.ID] = modules

		// Labels
		labels := make([]model.Label, len(labelTemplates))
		for i, l := range labelTemplates {
			labels[i] = model.Label{
				Name: l.name, Color: l.color, ProjectID: proj.ID,
				WorkspaceID: ws.ID,
			}
			db.Create(&labels[i])
		}
		allLabels[proj.ID] = labels
	}
	fmt.Println("Created states, cycles, modules, labels for all projects")

	// ============================================================
	// 5. ISSUES — ~100 per project (1000 total across 10 projects)
	// ============================================================
	priorities := []string{"urgent", "high", "medium", "low", "none"}

	featureTitles := []string{
		"用户单点登录(SSO)集成", "数据导出报表功能", "批量操作工作项",
		"自定义仪表盘", "邮件通知模板配置", "第三方API集成",
		"工作流可视化编辑器", "高级搜索过滤器", "评论@提及功能",
		"文件在线预览", "甘特图拖拽调整", "时间线视图",
		"项目模板导入导出", "自动化规则引擎", "多维数据透视表",
		"Markdown图表支持", "快捷键自定义", "深色模式切换",
		"看板泳道配置", "跨项目依赖关系图", "实时协作编辑",
		"Webhook事件订阅", "API速率限制配置", "自定义字段公式计算",
		"批量导入CSV/Excel", "日历订阅(iCal)", "附件版本管理",
	}
	bugTitles := []string{
		"修复登录页面密码可见性切换失效", "分页组件在大数据量下卡顿",
		"修复文件上传时文件类型校验绕过", "甘特图时间刻度显示错位",
		"通知邮件中链接无法点击", "修复看板拖拽后状态未保存",
		"移动端表格横向滚动异常", "修复日期选择器在Safari中无法使用",
		"API返回500当参数为空数组时", "富文本编辑器图片粘贴失败",
		"修复工作流转换时权限校验缺失", "修复批量删除后列表未刷新",
		"导出Excel中自定义字段缺失", "修复并发编辑冲突未提示",
		"修复搜索时特殊字符导致报错", "日历视图周模式显示错误",
		"修复通知中心未读数不更新", "修复模块删除后关联issue未清理",
		"修复图表在数据为空时崩溃", "修复列表横向滚动条不显示",
	}
	taskTitles := []string{
		"编写API接口文档", "数据库索引优化", "前端组件单元测试覆盖",
		"代码规范ESLint配置", "CI/CD流水线优化", "依赖包安全升级",
		"日志采集与集中存储", "Redis缓存策略调整", "Nginx反向代理配置",
		"Docker镜像体积优化", "Kubernetes健康检查配置", "数据库慢查询分析",
		"前端打包体积优化", "静态资源CDN部署", "SSL证书自动续期",
		"监控指标Dashboard创建", "告警规则阈值调整", "备份恢复演练",
		"性能基准测试", "安全漏洞修复", "i18n翻译文件整理",
	}
	storyTitles := []string{
		"作为用户，我希望能通过OAuth登录系统", "作为PM，我希望能批量指派任务",
		"作为开发者，我希望能查看代码关联的issue", "作为测试，我希望能一键提交Bug报告",
		"作为管理员，我希望能自定义用户角色权限", "作为Viewer，我希望能自定义看板视图",
		"作为团队成员，我希望能收到实时通知", "作为项目经理，我希望能查看Sprint进度报告",
		"作为客户，我希望能通过链接查看项目进度", "作为运营，我希望能导出数据到BI工具",
	}

	issueCreated := 0
	for projIdx, proj := range projects {
		wsID := proj.WorkspaceID
		var ws model.Workspace
		for _, w := range workspaces {
			if w.ID == wsID {
				ws = w
				break
			}
		}

		states := allStates[proj.ID]
		cycles := allCycles[proj.ID]
		modules := allModules[proj.ID]
		labels := allLabels[proj.ID]

		var issueTypes []model.IssueType
		db.Where("workspace_id = ?", ws.ID).Find(&issueTypes)
		typeByName := make(map[string]model.IssueType)
		for _, t := range issueTypes {
			typeByName[t.Name] = t
		}

		numIssues := 85 + rng.Intn(31) // 85-115, totalling ~1000 across 10 projects
		stateWeights := []int{15, 20, 25, 15, 20, 5}

		for i := 0; i < numIssues; i++ {
			var title string
			var issueTypeTag string
			roll := rng.Intn(100)
			switch {
			case roll < 30:
				title = featureTitles[rng.Intn(len(featureTitles))]
				issueTypeTag = "Feature"
			case roll < 50:
				title = bugTitles[rng.Intn(len(bugTitles))]
				issueTypeTag = "Bug"
			case roll < 70:
				title = taskTitles[rng.Intn(len(taskTitles))]
				issueTypeTag = "Task"
			case roll < 85:
				title = storyTitles[rng.Intn(len(storyTitles))]
				issueTypeTag = "Story"
			case roll < 95:
				title = fmt.Sprintf("Epic: %s", featureTitles[rng.Intn(len(featureTitles))])
				issueTypeTag = "Epic"
			default:
				title = fmt.Sprintf("技术调研: %s", featureTitles[rng.Intn(len(featureTitles))])
				issueTypeTag = "Spike"
			}
			title = fmt.Sprintf("%s (#%d-%d)", title, projIdx+1, i+1)

			// Weighted state
			stateIdx := weightedRandom(stateWeights, rng)
			selectedState := states[stateIdx]
			priority := priorities[weightedRandom([]int{5, 15, 30, 30, 20}, rng)]

			descHTML := fmt.Sprintf("<h2>描述</h2><p>项目 %s 中的 %s 类型工作项。</p><h3>验收标准</h3><ul><li>功能正常工作</li><li>测试覆盖通过</li></ul>", proj.Name, issueTypeTag)
			descStripped := fmt.Sprintf("项目 %s 中的 %s 类型工作项。验收标准：功能正常工作，测试覆盖通过。", proj.Name, issueTypeTag)

			startDate := time.Now().AddDate(0, 0, -rng.Intn(120))
			var targetDate *time.Time
			if rng.Intn(100) < 60 {
				td := startDate.AddDate(0, 0, rng.Intn(30)+3)
				targetDate = &td
			}
			var completedAt *time.Time
			if selectedState.ID == states[4].ID || selectedState.ID == states[5].ID {
				ct := startDate.AddDate(0, 0, rng.Intn(14)+1)
				completedAt = &ct
			}

			var issueTypeID *uint64
			if it, ok := typeByName[issueTypeTag]; ok {
				issueTypeID = &it.ID
			}

			issue := model.Issue{
				Name:                title,
				DescriptionHTML:     descHTML,
				DescriptionStripped: &descStripped,
				Priority:            priority,
				SequenceID:          i + 1,
				StateID:             selectedState.ID,
				ProjectID:           proj.ID,
				WorkspaceID:         ws.ID,
				StartDate:           &startDate,
				TargetDate:          targetDate,
				CompletedAt:         completedAt,
				SortOrder:           float64(rng.Intn(10000)) / 100.0,
				IssueTypeID:         issueTypeID,
			}

			// Assign IssueTypeID based on issueTypeTag
			if typeMap, ok := allIssueTypes[ws.ID]; ok {
				if it, ok := typeMap[issueTypeTag]; ok {
					issueTypeID := it.ID
					issue.IssueTypeID = &issueTypeID
				}
			}

			if err := db.Create(&issue).Error; err != nil {
				fmt.Printf("  WARN: failed to create issue: %v\n", err)
				continue
			}
			issueCreated++

			// Cycle (60%)
			if rng.Intn(100) < 60 && len(cycles) > 0 {
				cycle := cycles[rng.Intn(len(cycles))]
				db.Create(&model.IssueCycle{
					IssueID: issue.ID,
					CycleID: cycle.ID,
				})
			}

			// Module (80%)
			if rng.Intn(100) < 80 && len(modules) > 0 {
				module := modules[rng.Intn(len(modules))]
				db.Create(&model.ModuleIssue{
					IssueID:  issue.ID,
					ModuleID: module.ID,
				})
			}

			// Labels (1-5)
			numLabels := 1 + rng.Intn(5)
			rng.Shuffle(len(labels), func(a, b int) { labels[a], labels[b] = labels[b], labels[a] })
			for j := 0; j < numLabels && j < len(labels); j++ {
				db.Create(&model.IssueLabel{
					IssueID: issue.ID,
					LabelID: labels[j].ID,
				})
			}

			// Assignees (1-3)
			numAssignees := 1 + rng.Intn(3)
			rng.Shuffle(len(users), func(a, b int) { users[a], users[b] = users[b], users[a] })
			for j := 0; j < numAssignees && j < len(users); j++ {
				db.Create(&model.IssueAssignee{
					IssueID: issue.ID,
					UserID:  users[j].ID,
				})
			}

			// ============================================================
			// Issue Activities — 3-15 per issue
			// ============================================================
			numActivities := 3 + rng.Intn(13)
			activityVerbs := []string{"created", "updated", "commented", "assigned", "state_changed", "priority_changed", "labeled", "added_to_cycle", "added_to_module", "mentioned"}
			for k := 0; k < numActivities; k++ {
				verb := activityVerbs[rng.Intn(len(activityVerbs))]
				actor := users[rng.Intn(len(users))]
				var field, oldVal, newVal *string
				switch verb {
				case "created":
					field = strPtr("issue")
					newVal = strPtr(title)
				case "state_changed":
					field = strPtr("state")
					oldSt := states[rng.Intn(len(states))]
					oldVal = strPtr(oldSt.Name)
					newVal = strPtr(selectedState.Name)
				case "priority_changed":
					field = strPtr("priority")
					oldVal = strPtr(priorities[rng.Intn(len(priorities))])
					newVal = strPtr(priority)
				case "assigned":
					field = strPtr("assignee")
					newVal = strPtr(users[rng.Intn(len(users))].DisplayName)
				case "labeled":
					field = strPtr("label")
					newVal = strPtr(labels[rng.Intn(len(labels))].Name)
				case "commented":
					field = strPtr("comment")
					newVal = strPtr("添加了评论")
				}
				db.Create(&model.IssueActivity{
					IssueID:  &issue.ID,
					Verb:     verb,
					Field:    field,
					OldValue: oldVal,
					NewValue: newVal,
					ActorID:  uintPtr(actor.ID),
				})
			}

			// ============================================================
			// Comments — 40% of issues get 1-5 comments
			// ============================================================
			if rng.Intn(100) < 40 {
				numComments := 1 + rng.Intn(5)
				commentTexts := []string{
					"这个我来处理，优先级可以调高一些。",
					"已经和相关团队确认过了，可以按这个方案推进。",
					"需要在周三之前完成，否则会影响Sprint目标。",
					"设计稿已经更新了，可以开始开发。",
					"发现一个边界情况需要处理，详见附件截图。",
					"性能测试结果已出，响应时间在预期范围内。",
					"代码审查通过，准备合并到主分支。",
					"已部署到预发布环境，请大家测试验证。",
					"文档已经同步更新到Wiki上。",
					"建议拆分成两个独立的issue来处理",
					"相关讨论见Slack频道 #project-discussion",
					"单元测试已添加，覆盖率达标",
					"需要产品经理确认需求优先级",
					"生产环境验证通过，可以关闭此工单",
				}
				for c := 0; c < numComments; c++ {
					text := commentTexts[rng.Intn(len(commentTexts))]
					commenter := users[rng.Intn(len(users))]
					db.Create(&model.Comment{
						IssueID:  issue.ID,
						AuthorID: uintPtr(commenter.ID),
						Body:     text,
					})
				}
			}

			// ============================================================
			// Time Tracks — 30% of in-progress/done issues
			// ============================================================
			inProgressOrDone := selectedState.ID == states[2].ID || selectedState.ID == states[3].ID || selectedState.ID == states[4].ID
			if rng.Intn(100) < 30 && inProgressOrDone {
				numTracks := 1 + rng.Intn(8)
				for t := 0; t < numTracks; t++ {
					trackStart := startDate.Add(time.Duration(t) * time.Hour * 24 * time.Duration(rng.Intn(3)))
					durationMin := int64(rng.Intn(480) + 15) // 15 min to 8 hours
					duration := durationMin * 60             // store seconds (runtime unit)
					trackEnd := trackStart.Add(time.Duration(duration) * time.Second)
					tracker := users[rng.Intn(len(users))]
					descTexts := []string{"需求分析", "技术方案设计", "编码实现", "Code Review修改", "自测", "修复测试问题", "文档编写", "部署验证"}
					trackDesc := descTexts[rng.Intn(len(descTexts))]
					db.Create(&model.TimeTrack{
						IssueID:     issue.ID,
						UserID:      tracker.ID,
						StartedAt:   trackStart,
						EndedAt:     &trackEnd,
						Duration:    duration,
						Description: strPtr(trackDesc),
					})
				}
			}
		}
		fmt.Printf("  Project %s: created %d issues with activities, comments, time tracks\n", proj.Identifier, numIssues)
	}
	fmt.Printf("Total issues created: %d\n", issueCreated)

	// ============================================================
	// 6. RELEASES — 3-5 per project
	// ============================================================
	for _, proj := range projects {
		releaseNames := []string{"v1.0.0", "v1.1.0", "v1.2.0", "v2.0.0", "v2.1.0"}
		releaseStatuses := []string{"planned", "in_progress", "released", "cancelled"}
		// 3..5 releases, always within bounds of releaseNames.
		numReleases := 3 + rng.Intn(len(releaseNames)-2)
		for r, releaseName := range releaseNames[:numReleases] {
			descs := []string{
				"主要功能版本发布",
				"包含多个Bug修复和改进",
				"性能优化和安全加固版本",
				"新架构迁移过渡版本",
				"长期支持版本(LTS)",
			}
			releaseDate := time.Now().AddDate(0, r-1, 0)
			var relDatePtr *time.Time
			if rng.Intn(100) < 70 {
				relDatePtr = &releaseDate
			}
			rel := model.Release{
				Name:        releaseName,
				Description: descs[rng.Intn(len(descs))],
				Version:     releaseName,
				Status:      releaseStatuses[rng.Intn(len(releaseStatuses))],
				ReleaseDate: relDatePtr,
				ProjectID:   proj.ID,
			}
			if err := db.Create(&rel).Error; err != nil {
				continue
			}

			// Link random issues
			var projIssues []model.Issue
			db.Where("project_id = ?", proj.ID).Limit(100).Find(&projIssues)
			rng.Shuffle(len(projIssues), func(a, b int) { projIssues[a], projIssues[b] = projIssues[b], projIssues[a] })
			linkCount := rng.Intn(11) + 5
			for j := 0; j < linkCount && j < len(projIssues); j++ {
				db.Create(&model.ReleaseIssue{
					ReleaseID: rel.ID, IssueID: projIssues[j].ID,
				})
			}
		}
	}
	fmt.Println("Created releases with linked issues")

	// ============================================================
	// 6.1 INITIATIVES — 2-4 per workspace
	// ============================================================
	for _, ws := range workspaces {
		numInitiatives := 2 + rng.Intn(3)
		initiativeNames := []string{"用户增长计划", "平台稳定性提升", "国际化扩展", "移动端适配", "AI智能助手集成"}
		initiativeStatuses := []string{"active", "completed", "at_risk", "paused"}
		initiativeColors := []string{"#22c55e", "#3b82f6", "#eab308", "#6b7280"}
		for i := 0; i < numInitiatives; i++ {
			startDate := time.Now().AddDate(0, -rng.Intn(6), 0)
			targetDate := startDate.AddDate(0, 3+rng.Intn(6), 0)
			ini := model.Initiative{
				WorkspaceID: ws.ID,
				Name:        initiativeNames[rng.Intn(len(initiativeNames))],
				Description: fmt.Sprintf("战略规划：%s的年度重点目标", initiativeNames[rng.Intn(len(initiativeNames))]),
				Color:       initiativeColors[rng.Intn(len(initiativeColors))],
				Status:      initiativeStatuses[rng.Intn(len(initiativeStatuses))],
				StartDate:   &startDate,
				TargetDate:  &targetDate,
				CreatedByID: users[0].ID,
			}
			if err := db.Create(&ini).Error; err != nil {
				continue
			}
			// Link 1-2 random projects to this initiative
			rng.Shuffle(len(projects), func(a, b int) { projects[a], projects[b] = projects[b], projects[a] })
			linkCount := 1 + rng.Intn(2)
			for j := 0; j < linkCount && j < len(projects); j++ {
				db.Create(&model.InitiativeProject{InitiativeID: ini.ID, ProjectID: projects[j].ID})
			}
		}
	}
	fmt.Println("Created initiatives with linked projects")

	// ============================================================
	// 7. PAGES — 5-12 per project
	// ============================================================
	for _, proj := range projects {
		numPages := 5 + rng.Intn(8)
		pageTitles := []string{
			"项目概述", "开发环境搭建指南", "API接口文档", "部署运维手册",
			"编码规范", "测试策略", "发布流程", "常见问题FAQ",
			"架构设计文档", "数据字典", "第三方依赖说明", "安全规范",
		}
		var parentID *uint64
		for p := 0; p < numPages; p++ {
			title := pageTitles[p%len(pageTitles)]
			if p >= len(pageTitles) {
				title = fmt.Sprintf("%s (续)", title)
			}
			content := fmt.Sprintf("# %s\n\n## 概述\n本文档描述 %s 项目的相关信息。\n\n## 详细内容\n...\n\n> 最后更新：%s", title, proj.Name, time.Now().Format("2006-01-02"))
			authorID := users[rng.Intn(len(users))].ID
			page := model.Page{
				Title:       title,
				Content:     content,
				ProjectID:   proj.ID,
				WorkspaceID: proj.WorkspaceID,
				ParentID:    parentID,
				BaseModel:   model.BaseModel{CreatedByID: &authorID},
			}
			if err := db.Create(&page).Error; err != nil {
				continue
			}
			if p == 0 {
				parentID = &page.ID
			}
			if p > 4 {
				parentID = nil
			}
		}
	}
	fmt.Println("Created documentation pages")

	// ============================================================
	// 8. SAVED VIEWS — 3-5 per project
	// ============================================================
	for _, proj := range projects {
		numViews := 3 + rng.Intn(3)
		viewDefs := []struct {
			name, viewType, filters string
		}{
			{"我的待办", "list", `[{"field":"assignee","operator":"is_me","value":"true"}]`},
			{"高优先级任务", "list", `[{"field":"priority","operator":"in","value":"urgent,high"}]`},
			{"进行中", "kanban", `[{"field":"state_group","operator":"equals","value":"started"}]`},
			{"Bug列表", "list", `[{"field":"issue_type","operator":"equals","value":"Bug"}]`},
			{"未规划工作", "list", `[{"field":"cycle","operator":"is_null","value":"true"}]`},
		}
		for v := 0; v < numViews; v++ {
			vd := viewDefs[v%len(viewDefs)]
			owner := users[rng.Intn(len(users))]
			db.Create(&model.SavedView{
				Name:      vd.name,
				ViewType:  vd.viewType,
				Filters:   []byte(vd.filters),
				ProjectID: proj.ID,
				OwnerID:   owner.ID,
			})
		}
	}
	fmt.Println("Created saved views")

	// ============================================================
	// 9. PROJECT UPDATES — 5-10 per project
	// ============================================================
	for _, proj := range projects {
		numUpdates := 5 + rng.Intn(6)
		statuses := []string{"on_track", "at_risk", "off_track"}
		updateContents := []string{
			"本周完成了核心模块的开发和自测，代码审查通过，准备进入测试阶段。",
			"由于第三方API变更，部分功能需要调整实现方案，预计延期2天。",
			"Sprint回顾会议已召开，团队整体进度正常，质量指标达标。",
			"新成员完成入职培训，已开始承担开发任务。",
			"生产环境监控指标一切正常，本周无重大事件。",
			"需求变更已确认，对应的开发任务已更新并重新排期。",
			"性能压测通过，TPS达到预期目标的120%。",
			"安全扫描发现的2个中危漏洞已修复，待发布到生产。",
			"客户验收测试通过，准备下周部署到生产环境。",
			"技术债务清理工作已完成30%，剩余项已排入下个Sprint。",
		}
		for u := 0; u < numUpdates; u++ {
			author := users[rng.Intn(len(users))]
			db.Create(&model.ProjectUpdate{
				ProjectID: proj.ID,
				AuthorID:  author.ID,
				Status:    statuses[rng.Intn(len(statuses))],
				Content:   updateContents[u%len(updateContents)],
			})
		}
	}
	fmt.Println("Created project updates (status reports)")

	// ============================================================
	// 10. NOTIFICATIONS
	// ============================================================
	adminID := adminUser.ID
	notifications := []model.Notification{
		{Title: "欢迎使用 ReqMango", Message: "您的项目管理平台已就绪。开始创建第一个工作项吧！", Type: "info", Priority: "medium", RecipientID: adminID},
		{Title: "数据初始化完成", Message: "测试数据已生成：3个工作空间、10个项目、500+工作项", Type: "success", Priority: "high", RecipientID: adminID},
		{Title: "安全提醒", Message: "请及时修改默认密码，确保系统安全。", Type: "warning", Priority: "high", RecipientID: adminID},
		{Title: "系统更新通知", Message: "ReqMango v2.0 已发布，包含自定义报表和RBAC权限系统。", Type: "info", Priority: "medium", RecipientID: adminID},
		{Title: "Sprint Review 提醒", Message: "当前Sprint将在3天后结束，请及时更新工作项状态。", Type: "reminder", Priority: "high", RecipientID: adminID, ProjectID: &projects[0].ID},
	}
	for _, n := range notifications {
		db.Create(&n)
	}
	fmt.Println("Created sample notifications")

	// ============================================================
	// 11. ISSUE RELATIONS — cross-issue blocking relationships
	// ============================================================
	wsProjects := make(map[uint64][]model.Project)
	for _, proj := range projects {
		wsProjects[proj.WorkspaceID] = append(wsProjects[proj.WorkspaceID], proj)
	}
	for wsID := range wsProjects {
		var wsIssues []model.Issue
		db.Where("workspace_id = ?", wsID).Limit(200).Find(&wsIssues)
		if len(wsIssues) < 10 {
			continue
		}
		var blocksRel model.RelationType
		if db.Where("workspace_id = ? AND name = ?", wsID, "阻塞").First(&blocksRel).Error != nil {
			continue
		}
		if blocksRel.ID == 0 {
			continue
		}
		for j := 0; j < 30 && j*2 < len(wsIssues); j++ {
			db.Create(&model.IssueRelation{
				IssueID:        wsIssues[j*2].ID,
				RelatedIssueID: wsIssues[j*2+1].ID,
				RelationTypeID: blocksRel.ID,
			})
		}
	}
	fmt.Println("Created cross-issue relations")

	// ============================================================
	// 12. WORK ITEM TEMPLATES — 3 per project
	// ============================================================
	for _, proj := range projects {
		templates := []struct {
			name     string
			defaults string
		}{
			{"标准Bug报告", `{"priority":"high","description_html":"<h2>问题描述</h2><p>...</p><h2>复现步骤</h2><ol><li>步骤1</li></ol>"}`},
			{"功能需求模板", `{"priority":"medium","description_html":"<h2>需求背景</h2><p>...</p><h2>验收标准</h2><ul><li>标准1</li></ul>"}`},
			{"技术任务模板", `{"priority":"medium","description_html":"<h2>任务目标</h2><p>...</p><h2>技术方案</h2><p>...</p>"}`},
		}
		for _, tmpl := range templates {
			db.Create(&model.WorkItemTemplate{
				Name:        tmpl.name,
				Defaults:    []byte(tmpl.defaults),
				ProjectID:   proj.ID,
				WorkspaceID: proj.WorkspaceID,
			})
		}
	}
	fmt.Println("Created work item templates")

	fmt.Println("Demo data seeded successfully!")
}

func SeedConfigData(db *gorm.DB) {
	fmt.Println("--- Seeding configuration data ---")

	var ws model.Workspace
	if db.Where("slug = ?", "reqmango-dev").First(&ws).Error != nil {
		// Fall back to first workspace
		if db.First(&ws).Error != nil {
			fmt.Println("No workspace found, skipping config seed")
			return
		}
	}
	var proj model.Project
	if db.Where("workspace_id = ?", ws.ID).First(&proj).Error != nil {
		fmt.Println("No project found for workspace, skipping config seed")
		return
	}

	// Issue Types
	var typeCount int64
	db.Model(&model.IssueType{}).Where("workspace_id = ?", ws.ID).Count(&typeCount)
	if typeCount == 0 {
		epic := model.IssueType{Name: "Epic", Color: "#8B5CF6", Icon: "layers", Description: "顶层史诗级工作项", Level: 0, IsDefault: true, Sequence: 1, WorkspaceID: ws.ID}
		db.Create(&epic)
		feature := model.IssueType{Name: "Feature", Color: "#6366F1", Icon: "star", Description: "功能特性", Level: 1, ParentTypeID: &epic.ID, Sequence: 2, WorkspaceID: ws.ID}
		db.Create(&feature)
		// Bug/Task (Level 2) have no fixed ParentTypeID — can be children of any lower-level type
		bug := model.IssueType{Name: "Bug", Color: "#EF4444", Icon: "bug", Description: "缺陷/问题", Level: 2, Sequence: 3, WorkspaceID: ws.ID}
		db.Create(&bug)
		task := model.IssueType{Name: "Task", Color: "#10B981", Icon: "check-circle", Description: "开发任务", Level: 2, Sequence: 4, WorkspaceID: ws.ID}
		db.Create(&task)
		story := model.IssueType{Name: "Story", Color: "#F59E0B", Icon: "bookmark", Description: "用户故事", Level: 1, ParentTypeID: &epic.ID, Sequence: 5, WorkspaceID: ws.ID}
		db.Create(&story)
		spike := model.IssueType{Name: "Spike", Color: "#06B6D4", Icon: "zap", Description: "技术调研/探索", Level: 1, ParentTypeID: &epic.ID, Sequence: 6, WorkspaceID: ws.ID}
		db.Create(&spike)
		fmt.Printf("  Created 6 issue types (Epic/L0 → Feature/Story/Spike/L1 → Bug/Task/L2)\n")
	}

	// Custom Fields
	var cfCount int64
	db.Model(&model.CustomField{}).Where("workspace_id = ?", ws.ID).Count(&cfCount)
	if cfCount == 0 {
		priority := model.CustomField{Name: "优先级", FieldType: "dropdown", IsRequired: true, WorkspaceID: ws.ID}
		db.Create(&priority)
		db.Create(&model.CustomFieldOption{FieldID: priority.ID, Value: "P0-紧急", Color: "#EF4444", Sequence: 1})
		db.Create(&model.CustomFieldOption{FieldID: priority.ID, Value: "P1-高", Color: "#F97316", Sequence: 2})
		db.Create(&model.CustomFieldOption{FieldID: priority.ID, Value: "P2-中", Color: "#F59E0B", Sequence: 3})
		db.Create(&model.CustomFieldOption{FieldID: priority.ID, Value: "P3-低", Color: "#10B981", Sequence: 4})

		deadline := model.CustomField{Name: "截止日期", FieldType: "date", WorkspaceID: ws.ID}
		db.Create(&deadline)
		version := model.CustomField{Name: "版本号", FieldType: "text", WorkspaceID: ws.ID}
		db.Create(&version)
		storyPoints := model.CustomField{Name: "故事点", FieldType: "number", WorkspaceID: ws.ID}
		db.Create(&storyPoints)
		reporter := model.CustomField{Name: "报告人", FieldType: "member", WorkspaceID: ws.ID}
		db.Create(&reporter)
		refURL := model.CustomField{Name: "参考链接", FieldType: "url", WorkspaceID: ws.ID}
		db.Create(&refURL)
		fmt.Printf("  Created 6 custom fields (dropdown/date/text/number/member/url)\n")
	}

	// Bind custom fields to issue types
	var bindCount int64
	db.Model(&model.IssueTypeField{}).Count(&bindCount)
	if bindCount == 0 {
		var types []model.IssueType
		db.Where("workspace_id = ?", ws.ID).Find(&types)
		var fields []model.CustomField
		db.Where("workspace_id = ?", ws.ID).Find(&fields)
		for _, t := range types {
			for _, f := range fields {
				isReq := f.Name == "优先级"
				db.Create(&model.IssueTypeField{TypeID: t.ID, FieldID: f.ID, IsRequired: isReq, Sequence: 1})
			}
		}
		fmt.Printf("  Bound %d fields × %d types\n", len(fields), len(types))
	}

	// Labels (if not already created)
	var lblCount int64
	db.Model(&model.Label{}).Where("project_id = ?", proj.ID).Count(&lblCount)
	if lblCount == 0 {
		labelDefs := []struct{ name, color string }{
			{"前端", "#3B82F6"}, {"后端", "#22C55E"}, {"文档", "#F59E0B"},
			{"DevOps", "#8B5CF6"}, {"安全", "#EF4444"}, {"性能优化", "#F97316"},
			{"Bug", "#DC2626"}, {"增强", "#06B6D4"}, {"测试", "#84CC16"},
			{"设计", "#A855F7"}, {"需求", "#EC4899"}, {"架构", "#0EA5E9"},
		}
		for _, l := range labelDefs {
			db.Create(&model.Label{Name: l.name, Color: l.color, ProjectID: proj.ID, WorkspaceID: ws.ID})
		}
		fmt.Printf("  Created %d labels\n", len(labelDefs))
	}

	// Releases (if not already created)
	var releaseCount int64
	db.Model(&model.Release{}).Where("project_id = ?", proj.ID).Count(&releaseCount)
	if releaseCount == 0 {
		releaseDefs := []struct {
			name, version, description, status string
		}{
			{"v1.0.0", "v1.0.0", "首个正式版本发布", "released"},
			{"v1.1.0", "v1.1.0", "功能增强与Bug修复版本", "in_progress"},
			{"v2.0.0", "v2.0.0", "重大架构升级版本", "planned"},
		}
		for _, rd := range releaseDefs {
			db.Create(&model.Release{
				Name:        rd.name,
				Version:     rd.version,
				Description: rd.description,
				Status:      rd.status,
				ProjectID:   proj.ID,
			})
		}
		fmt.Printf("  Created %d releases\n", len(releaseDefs))
	}

	// Workflow
	var wfCount int64
	db.Model(&model.Workflow{}).Where("project_id = ?", proj.ID).Count(&wfCount)
	if wfCount == 0 {
		var stList []model.State
		db.Where("project_id = ? AND is_active = true", proj.ID).Order("sequence").Find(&stList)
		if len(stList) >= 5 {
			bid, tid, ipid, rid, dnid := stList[0].ID, stList[1].ID, stList[2].ID, stList[3].ID, stList[4].ID
			pid := proj.ID
			wf := model.Workflow{Name: "Default Workflow", Description: "标准状态流转规则", ProjectID: &pid, IsActive: true}
			db.Create(&wf)
			trs := []model.StateTransition{
				{Name: "Backlog→Todo", WorkflowID: wf.ID, SourceStateID: bid, TargetStateID: tid, RuleType: "allow", ProjectID: &pid, WorkspaceID: ws.ID},
				{Name: "Todo→InProgress", WorkflowID: wf.ID, SourceStateID: tid, TargetStateID: ipid, RuleType: "allow", ProjectID: &pid, WorkspaceID: ws.ID},
				{Name: "InProgress→Review", WorkflowID: wf.ID, SourceStateID: ipid, TargetStateID: rid, RuleType: "allow", ProjectID: &pid, WorkspaceID: ws.ID},
				{Name: "Review→Done", WorkflowID: wf.ID, SourceStateID: rid, TargetStateID: dnid, RuleType: "approval", ProjectID: &pid, WorkspaceID: ws.ID},
				{Name: "Review→InProgress", WorkflowID: wf.ID, SourceStateID: rid, TargetStateID: ipid, RuleType: "allow", ProjectID: &pid, WorkspaceID: ws.ID},
			}
			if len(stList) >= 6 {
				cid := stList[5].ID
				trs = append(trs,
					model.StateTransition{Name: "Backlog→Cancelled", WorkflowID: wf.ID, SourceStateID: bid, TargetStateID: cid, RuleType: "allow", ProjectID: &pid, WorkspaceID: ws.ID},
					model.StateTransition{Name: "Todo→Cancelled", WorkflowID: wf.ID, SourceStateID: tid, TargetStateID: cid, RuleType: "allow", ProjectID: &pid, WorkspaceID: ws.ID},
					model.StateTransition{Name: "InProgress→Cancelled", WorkflowID: wf.ID, SourceStateID: ipid, TargetStateID: cid, RuleType: "allow", ProjectID: &pid, WorkspaceID: ws.ID},
					model.StateTransition{Name: "Review→Cancelled", WorkflowID: wf.ID, SourceStateID: rid, TargetStateID: cid, RuleType: "allow", ProjectID: &pid, WorkspaceID: ws.ID},
					model.StateTransition{Name: "Done→Review", WorkflowID: wf.ID, SourceStateID: dnid, TargetStateID: rid, RuleType: "allow", ProjectID: &pid, WorkspaceID: ws.ID},
				)
			}
			for _, tr := range trs {
				db.Create(&tr)
			}
			fmt.Printf("  Created workflow with %d transitions\n", len(trs))
		}
	}

	// Automations
	var autoCount int64
	db.Model(&model.AutomationRule{}).Where("project_id = ?", proj.ID).Count(&autoCount)
	if autoCount == 0 {
		db.Create(&model.AutomationRule{
			Name: "紧急任务自动分配", Description: "urgent优先级的任务创建时自动分配给管理员",
			ProjectID: proj.ID, TriggerType: "issue_created", IsEnabled: true, Sequence: 1,
			Conditions: `[{"field":"priority","operator":"equals","value":"urgent"}]`,
			Actions:    `[{"type":"assign","field":"assignee","value":"1"}]`,
		})
		db.Create(&model.AutomationRule{
			Name: "完成任务时记录时间", Description: "状态变为Done时自动记录完成时间",
			ProjectID: proj.ID, TriggerType: "state_changed", IsEnabled: true, Sequence: 2,
			Conditions: `[{"field":"state_group","operator":"equals","value":"completed"}]`,
			Actions:    `[{"type":"set_timestamp","field":"completed_at"}]`,
		})
		db.Create(&model.AutomationRule{
			Name: "自动设置截止日期", Description: "新建任务时自动设置截止日期为3天后",
			ProjectID: proj.ID, TriggerType: "issue_created", IsEnabled: true, Sequence: 3,
			Conditions: `[]`,
			Actions:    `[{"type":"set_date_offset","field":"target_date","offset_days":3}]`,
		})
		fmt.Println("  Created 3 automation rules")
	}

	// Relation Types
	var relCount int64
	db.Model(&model.RelationType{}).Where("workspace_id = ?", ws.ID).Count(&relCount)
	if relCount == 0 {
		relTypes := []model.RelationType{
			{Name: "阻塞", InwardName: "被阻塞于", OutwardName: "阻塞", WorkspaceID: ws.ID},
			{Name: "关联", InwardName: "关联到", OutwardName: "被关联于", WorkspaceID: ws.ID},
			{Name: "重复", InwardName: "重复于", OutwardName: "被重复于", WorkspaceID: ws.ID},
			{Name: "父子", InwardName: "子任务", OutwardName: "父任务", WorkspaceID: ws.ID},
			{Name: "依赖", InwardName: "依赖于", OutwardName: "被依赖于", WorkspaceID: ws.ID},
		}
		for _, r := range relTypes {
			db.Create(&r)
		}
		fmt.Printf("  Created %d relation types\n", len(relTypes))
	}

	// AI Agents
	var agentCount int64
	db.Model(&model.Agent{}).Where("workspace_id = ?", ws.ID).Count(&agentCount)
	if agentCount == 0 {
		adminID := ws.OwnerID
		db.Create(&model.Agent{
			Name: "Triage Agent", Avatar: "🏥", AgentType: "builtin",
			Capabilities: []byte(`["analyze","search","comment","list"]`),
			Status:       "active",
			SystemPrompt: strPtr("You are a triage specialist. Analyze incoming issues and suggest the correct type, priority, labels, and assignee. Be concise and data-driven."),
			WorkspaceID:  ws.ID,
			BaseModel:    model.BaseModel{CreatedByID: &adminID},
		})
		db.Create(&model.Agent{
			Name: "Summary Agent", Avatar: "📋", AgentType: "builtin",
			Capabilities: []byte(`["summarize","analyze","list"]`),
			Status:       "active",
			SystemPrompt: strPtr("You are a project analyst. Summarize sprint progress, project health, and team performance. Provide actionable insights."),
			WorkspaceID:  ws.ID,
			BaseModel:    model.BaseModel{CreatedByID: &adminID},
		})
		db.Create(&model.Agent{
			Name: "Assistant Agent", Avatar: "🤖", AgentType: "builtin",
			Capabilities: []byte(`["all"]`),
			Status:       "active",
			SystemPrompt: strPtr("You are a helpful project management assistant. Help users with any task."),
			WorkspaceID:  ws.ID,
			BaseModel:    model.BaseModel{CreatedByID: &adminID},
		})
		fmt.Println("  Created 3 built-in AI agents")
	}

	fmt.Println("--- Config seed complete ---")
}

// weightedRandom picks an index based on weights
func weightedRandom(weights []int, rng *randutil.RNG) int {
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rng.Intn(total)
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if r < cumulative {
			return i
		}
	}
	return len(weights) - 1
}

func SeedSearchTemplates(db *gorm.DB) {
	fmt.Println("--- Seeding search templates ---")

	var projects []model.Project
	db.Find(&projects)
	if len(projects) == 0 {
		fmt.Println("  No projects found, skipping search templates")
		return
	}

	builtInTemplates := []struct {
		Name        string
		Description string
		Icon        string
		RQLTemplate string
		ViewType    string
		SortConfig  string
		GroupBy     *string
	}{
		{
			Name:        "我的待办",
			Description: "分配给我的所有待处理任务",
			Icon:        "📝",
			RQLTemplate: "state_group IN ('backlog', 'unstarted', 'started') AND assignee_id IN ($CURRENT_USER)",
			ViewType:    "list",
			SortConfig:  `[{"field":"priority","dir":"desc"},{"field":"target_date","dir":"asc"}]`,
		},
		{
			Name:        "本周到期",
			Description: "本周即将到期的任务",
			Icon:        "⏰",
			RQLTemplate: "target_date <= $END_OF_WEEK AND state_group != 'completed'",
			ViewType:    "list",
			SortConfig:  `[{"field":"target_date","dir":"asc"}]`,
		},
		{
			Name:        "高优先级",
			Description: "所有高优先级和紧急任务",
			Icon:        "🚨",
			RQLTemplate: "priority IN ('high', 'urgent') AND state_group != 'completed'",
			ViewType:    "list",
			SortConfig:  `[{"field":"priority","dir":"desc"},{"field":"created_at","dir":"desc"}]`,
		},
		{
			Name:        "我的已完成",
			Description: "我完成的任务",
			Icon:        "✅",
			RQLTemplate: "state_group = 'completed' AND assignee_id IN ($CURRENT_USER)",
			ViewType:    "list",
			SortConfig:  `[{"field":"completed_at","dir":"desc"}]`,
		},
		{
			Name:        "未分配任务",
			Description: "还没有分配人的任务",
			Icon:        "👤",
			RQLTemplate: "assignee_id IS NULL AND state_group != 'completed'",
			ViewType:    "list",
			SortConfig:  `[{"field":"created_at","dir":"asc"}]`,
		},
		{
			Name:        "需要关注",
			Description: "超过一周未更新的任务",
			Icon:        "🔔",
			RQLTemplate: "updated_at <= $ONE_WEEK_AGO AND state_group != 'completed'",
			ViewType:    "list",
			SortConfig:  `[{"field":"updated_at","dir":"asc"}]`,
		},
		{
			Name:        "待审核",
			Description: "所有进行中和评审中的任务",
			Icon:        "🔍",
			RQLTemplate: "state_group = 'started'",
			ViewType:    "list",
			SortConfig:  `[{"field":"updated_at","dir":"desc"}]`,
		},
		{
			Name:        "看板视图",
			Description: "按状态分组的看板视图",
			Icon:        "📋",
			RQLTemplate: "",
			ViewType:    "kanban",
			SortConfig:  `[{"field":"priority","dir":"desc"}]`,
		},
	}

	for _, proj := range projects {
		for _, bt := range builtInTemplates {
			var exists model.SearchTemplate
			if db.Where("project_id = ? AND name = ? AND is_built_in = ?", proj.ID, bt.Name, true).First(&exists).Error == nil {
				continue
			}

			t := &model.SearchTemplate{
				Name:        bt.Name,
				Icon:        bt.Icon,
				RQLTemplate: bt.RQLTemplate,
				ViewType:    bt.ViewType,
				SortConfig:  []byte(bt.SortConfig),
				GroupBy:     bt.GroupBy,
				IsBuiltIn:   true,
				IsPublic:    true,
				OwnerID:     nil,
				ProjectID:   proj.ID,
			}
			desc := bt.Description
			t.Description = &desc

			if err := db.Create(t).Error; err != nil {
				fmt.Printf("  WARN: failed to create search template %s for project %d: %v\n", bt.Name, proj.ID, err)
			}
		}
	}
	fmt.Printf("  Created search templates for %d projects\n", len(projects))
}

// SeedIssueTypesForAllWorkspaces ensures every workspace has issue types.
func SeedIssueTypesForAllWorkspaces(db *gorm.DB) {
	var workspaces []model.Workspace
	db.Find(&workspaces)
	for _, ws := range workspaces {
		var count int64
		db.Model(&model.IssueType{}).Where("workspace_id = ?", ws.ID).Count(&count)
		if count > 0 {
			continue
		}
		epic := model.IssueType{Name: "Epic", Color: "#8B5CF6", Icon: "layers", Description: "顶层史诗级工作项", Level: 0, IsDefault: true, Sequence: 1, WorkspaceID: ws.ID}
		db.Create(&epic)
		feature := model.IssueType{Name: "Feature", Color: "#6366F1", Icon: "star", Description: "功能特性", Level: 1, ParentTypeID: &epic.ID, Sequence: 2, WorkspaceID: ws.ID}
		db.Create(&feature)
		// Bug/Task (Level 2) have no fixed ParentTypeID — can be children of any lower-level type
		bug := model.IssueType{Name: "Bug", Color: "#EF4444", Icon: "bug", Description: "缺陷/问题", Level: 2, Sequence: 3, WorkspaceID: ws.ID}
		db.Create(&bug)
		task := model.IssueType{Name: "Task", Color: "#10B981", Icon: "check-circle", Description: "开发任务", Level: 2, Sequence: 4, WorkspaceID: ws.ID}
		db.Create(&task)
		story := model.IssueType{Name: "Story", Color: "#F59E0B", Icon: "bookmark", Description: "用户故事", Level: 1, ParentTypeID: &epic.ID, Sequence: 5, WorkspaceID: ws.ID}
		db.Create(&story)
		spike := model.IssueType{Name: "Spike", Color: "#06B6D4", Icon: "zap", Description: "技术调研/探索", Level: 1, ParentTypeID: &epic.ID, Sequence: 6, WorkspaceID: ws.ID}
		db.Create(&spike)
	}
	fmt.Println("Seeded issue types for all workspaces")
}

// SeedRelationTypesForAllWorkspaces ensures every workspace has default relation
// types so that issue relations can reference them out of the box.
func SeedRelationTypesForAllWorkspaces(db *gorm.DB) {
	var workspaces []model.Workspace
	db.Find(&workspaces)
	for _, ws := range workspaces {
		var count int64
		db.Model(&model.RelationType{}).Where("workspace_id = ?", ws.ID).Count(&count)
		if count > 0 {
			continue
		}
		relTypes := []model.RelationType{
			{Name: "阻塞", InwardName: "被阻塞于", OutwardName: "阻塞", WorkspaceID: ws.ID},
			{Name: "关联", InwardName: "关联到", OutwardName: "被关联于", WorkspaceID: ws.ID},
			{Name: "重复", InwardName: "重复于", OutwardName: "被重复于", WorkspaceID: ws.ID},
			{Name: "父子", InwardName: "子任务", OutwardName: "父任务", WorkspaceID: ws.ID},
			{Name: "依赖", InwardName: "依赖于", OutwardName: "被依赖于", WorkspaceID: ws.ID},
		}
		for _, r := range relTypes {
			db.Create(&r)
		}
	}
	fmt.Println("Seeded relation types for all workspaces")
}

// SeedAutomationRulesForAllWorkspaces creates built-in automation rules for
// every workspace that has no automation rules yet. These are sensible defaults
// inspired by built-in automations.
func SeedAutomationRulesForAllWorkspaces(db *gorm.DB) {
	var workspaces []model.Workspace
	db.Find(&workspaces)
	for _, ws := range workspaces {
		var count int64
		db.Model(&model.AutomationRule{}).Where("workspace_id = ?", ws.ID).Count(&count)
		if count > 0 {
			continue // Workspace already has automation rules, skip
		}

		builtInRules := []model.AutomationRule{
			{
				Name:        "高优先级任务自动分配",
				Description: "当创建 urgent 或 high 优先级任务时，自动分配给工作区管理员",
				WorkspaceID: ws.ID,
				TriggerType: "issue.created",
				Conditions:  `[{"field":"priority","operator":"in","value":["urgent","high"]}]`,
				Actions:     `[{"type":"assign_to","value":` + fmt.Sprintf("%d", ws.OwnerID) + `}]`,
				IsEnabled:   true,
				Sequence:    1,
				Scope:       "all",
			},
			{
				Name:           "自动归档已完成任务",
				Description:    "状态变更为已完成超过7天的任务自动归档",
				WorkspaceID:    ws.ID,
				TriggerType:    "scheduled",
				Conditions:     `[]`,
				Actions:        `[{"type":"comment","value":"Scheduled: 可在此配置自动归档逻辑"}]`,
				IsEnabled:      false, // disabled by default — needs schedule_config
				Sequence:       2,
				Scope:          "all",
				ScheduleConfig: `{"frequency":"daily","time":"02:00"}`,
			},
			{
				Name:        "Bug自动评论",
				Description: "当创建 Bug 类型工作项时自动添加评论提醒",
				WorkspaceID: ws.ID,
				TriggerType: "issue.created",
				Conditions:  `[{"field":"issue_type","operator":"equals","value":"Bug"}]`,
				Actions:     `[{"type":"add_comment","value":"[自动化] Bug 已创建，请及时处理"}]`,
				IsEnabled:   true,
				Sequence:    3,
				Scope:       "all",
			},
			{
				Name:           "长期未更新提醒",
				Description:    "每周一检查超过14天未更新的进行中任务，发送提醒",
				WorkspaceID:    ws.ID,
				TriggerType:    "scheduled",
				Conditions:     `[]`,
				Actions:        `[{"type":"comment","value":"Scheduled: 可在此配置过期任务提醒逻辑"}]`,
				IsEnabled:      false,
				Sequence:       4,
				Scope:          "all",
				ScheduleConfig: `{"frequency":"weekly","time":"09:00","days":["mon"]}`,
			},
			{
				Name:        "Webhook通知示例",
				Description: "当创建新工作项时，通过Webhook通知外部系统",
				WorkspaceID: ws.ID,
				TriggerType: "issue.created",
				Conditions:  `[]`,
				Actions:     `[{"type":"call_webhook","field":"https://httpbin.org/post","value":{"method":"POST","headers":{"Content-Type":"application/json"},"body":"{\"event\":\"issue_created\",\"issue_id\":{{issue_id}},\"workspace_id\":{{workspace_id}},\"trigger\":\"{{trigger_type}}\"}"}}]`,
				IsEnabled:   true,
				Sequence:    5,
				Scope:       "all",
			},
		}

		for _, rule := range builtInRules {
			if err := db.Create(&rule).Error; err != nil {
				fmt.Printf("  WARN: failed to seed automation rule '%s' for workspace %d: %v\n",
					rule.Name, ws.ID, err)
			}
		}
	}
	fmt.Println("Seeded built-in automation rules for all workspaces")
}

// SeedWebhookDemoExecutionLogs inserts demo execution logs for webhook
// automation rules so users can see what the execution log looks like.
func SeedWebhookDemoExecutionLogs(db *gorm.DB) {
	var rules []model.AutomationRule
	db.Where("actions LIKE ?", "%call_webhook%").Find(&rules)
	if len(rules) == 0 {
		fmt.Println("No webhook automation rules found, skipping execution log seed")
		return
	}

	for _, rule := range rules {
		// Find an issue in the same workspace
		var issue model.Issue
		if err := db.Where("workspace_id = ?", rule.WorkspaceID).First(&issue).Error; err != nil {
			fmt.Printf("  WARN: no issue found for workspace %d, skipping execution log seed for rule '%s'\n",
				rule.WorkspaceID, rule.Name)
			continue
		}

		// Check if execution log already exists for this rule
		var count int64
		db.Model(&model.AutomationExecution{}).Where("rule_id = ?", rule.ID).Count(&count)
		if count > 0 {
			continue
		}

		contextJSON := fmt.Sprintf(`{"event_type":"issue.created","trigger_type":"issue.created","project_id":%d,"workspace_id":%d}`,
			issue.ProjectID, rule.WorkspaceID)
		actionsTaken := `[{"type":"call_webhook","field":"https://httpbin.org/post","status":"success","response_code":200}]`

		exec := model.AutomationExecution{
			RuleID:       rule.ID,
			IssueID:      issue.ID,
			TriggerType:  "issue.created",
			ContextJSON:  contextJSON,
			ActionsTaken: actionsTaken,
			Status:       "success",
			Duration:     320, // ms
			ExecutedAt:   time.Now().Add(-1 * time.Hour),
		}
		if err := db.Create(&exec).Error; err != nil {
			fmt.Printf("  WARN: failed to seed execution log for rule '%s': %v\n", rule.Name, err)
		} else {
			fmt.Printf("  Seeded demo execution log for rule '%s' (rule_id=%d, issue_id=%d)\n",
				rule.Name, rule.ID, issue.ID)
		}

		// Also create a second execution log (a "failed" one for demo variety)
		failExec := model.AutomationExecution{
			RuleID:       rule.ID,
			IssueID:      issue.ID,
			TriggerType:  "issue.created",
			ContextJSON:  contextJSON,
			ActionsTaken: `[{"type":"call_webhook","field":"https://httpbin.org/post","status":"failed"}]`,
			Status:       "failed",
			Error:        "context deadline exceeded (Client.Timeout exceeded while awaiting headers)",
			Duration:     15000, // ms - timed out
			ExecutedAt:   time.Now().Add(-30 * time.Minute),
		}
		if err := db.Create(&failExec).Error; err != nil {
			fmt.Printf("  WARN: failed to seed fail execution log for rule '%s': %v\n", rule.Name, err)
		}
	}
}

// SeedAIData creates rich seed data for all AI modules.
func SeedAIData(db *gorm.DB) {
	fmt.Println("--- Seeding AI data ---")

	var workspaces []model.Workspace
	db.Find(&workspaces)
	if len(workspaces) == 0 {
		fmt.Println("  No workspaces found, skipping AI seed")
		return
	}

	// Use the first workspace for AI seed data
	ws := workspaces[0]
	adminID := ws.OwnerID

	// 1. Agent Templates (preset)
	seedAgentTemplates(db, ws.ID, adminID)

	// 2. Agent Configs (LLM model configs)
	seedAgentConfigs(db, ws.ID)

	// 3. Skills (preset)
	seedSkills(db, ws.ID)

	// 4. Runtimes
	seedRuntimes(db, ws.ID)

	// 5. Squads with members
	seedSquads(db, ws.ID, adminID)

	// 6. Memory entries
	seedMemories(db, ws.ID)

	// 7. Agent tasks
	seedAgentTasks(db, ws.ID)

	// 8. Agent activities
	seedAgentActivities(db, ws.ID)

	// 9. Agent sessions
	seedAgentSessions(db, ws.ID)

	fmt.Println("--- AI data seed complete ---")
}

func seedAgentTemplates(db *gorm.DB, wsID, adminID uint64) {
	var count int64
	db.Model(&model.AgentTemplate{}).Where("workspace_id = ? OR workspace_id IS NULL", wsID).Count(&count)
	if count > 0 {
		fmt.Println("  Agent templates already exist, skipping")
		return
	}

	templates := []model.AgentTemplate{
		{
			Name:        "代码审查专家",
			Description: strPtr("专注于代码质量审查，识别潜在问题并提供改进建议"),
			IsPreset:    true,
			Icon:        "🔍",
			SystemPrompt: `你是一位资深的代码审查专家。你的职责是：
1. 检查代码风格和一致性
2. 识别潜在的bug和安全漏洞
3. 评估代码复杂度和可维护性
4. 提供具体的改进建议
请用简洁专业的语言给出审查意见。`,
			AvailableTools: []byte(`["analyze_code","search_code"]`),
			DefaultConfig:  []byte(`{"model":"deepseek-chat","temperature":0.3}`),
			Version:        "1.0",
			Status:         "active",
		},
		{
			Name:        "需求分析师",
			Description: strPtr("分析用户需求，提取关键信息，生成结构化需求文档"),
			IsPreset:    true,
			Icon:        "📋",
			SystemPrompt: `你是一位专业的需求分析师。你的职责是：
1. 理解用户的原始需求描述
2. 提取关键功能点和约束条件
3. 识别需求中的歧义和遗漏
4. 生成结构化的需求文档
请确保输出清晰、完整、可执行。`,
			AvailableTools: []byte(`["extract_requirements","analyze_content"]`),
			DefaultConfig:  []byte(`{"model":"deepseek-chat","temperature":0.5}`),
			Version:        "1.0",
			Status:         "active",
		},
		{
			Name:        "测试工程师",
			Description: strPtr("根据需求和代码生成测试用例，执行测试分析"),
			IsPreset:    true,
			Icon:        "🧪",
			SystemPrompt: `你是一位经验丰富的测试工程师。你的职责是：
1. 分析需求和代码逻辑
2. 设计全面的测试用例
3. 覆盖正常流程和边界情况
4. 识别潜在的测试盲区
请输出结构化的测试用例，包含输入、预期输出和测试步骤。`,
			AvailableTools: []byte(`["analyze_code","generate_test"]`),
			DefaultConfig:  []byte(`{"model":"deepseek-chat","temperature":0.4}`),
			Version:        "1.0",
			Status:         "active",
		},
		{
			Name:        "技术文档撰写者",
			Description: strPtr("根据代码和需求自动生成技术文档"),
			IsPreset:    true,
			Icon:        "📝",
			SystemPrompt: `你是一位技术文档撰写专家。你的职责是：
1. 理解代码结构和功能
2. 生成清晰的技术文档
3. 包含API说明、使用示例和注意事项
4. 保持文档格式规范统一
请输出Markdown格式的文档。`,
			AvailableTools: []byte(`["analyze_code","analyze_content"]`),
			DefaultConfig:  []byte(`{"model":"deepseek-chat","temperature":0.6}`),
			Version:        "1.0",
			Status:         "active",
		},
		{
			Name:        "项目管理助手",
			Description: strPtr("协助项目经理进行任务分配、进度跟踪和风险评估"),
			IsPreset:    true,
			Icon:        "📊",
			SystemPrompt: `你是一位专业的项目管理助手。你的职责是：
1. 分析项目进度和团队工作量
2. 提供任务优先级建议
3. 识别项目风险和瓶颈
4. 生成项目状态报告
请基于数据给出客观、可操作的建议。`,
			AvailableTools: []byte(`["list_issues","analyze_sprint","generate_report"]`),
			DefaultConfig:  []byte(`{"model":"deepseek-chat","temperature":0.5}`),
			Version:        "1.0",
			Status:         "active",
		},
	}

	for i := range templates {
		templates[i].WorkspaceID = &wsID
		templates[i].BaseModel = model.BaseModel{CreatedByID: &adminID}
		if err := db.Create(&templates[i]).Error; err != nil {
			fmt.Printf("  WARN: failed to create agent template '%s': %v\n", templates[i].Name, err)
		}
	}
	fmt.Printf("  Created %d agent templates\n", len(templates))
}

func seedAgentConfigs(db *gorm.DB, wsID uint64) {
	var count int64
	db.Model(&model.AgentConfig{}).Where("workspace_id = ?", wsID).Count(&count)
	if count > 0 {
		fmt.Println("  Agent configs already exist, skipping")
		return
	}

	configs := []model.AgentConfig{
		{
			Name:           "DeepSeek Chat",
			Description:    strPtr("DeepSeek 通用对话模型"),
			Provider:       "deepseek",
			Model:          "deepseek-chat",
			MaxTokens:      4096,
			Temperature:    0.7,
			TopP:           1.0,
			IsDefault:      true,
			IsActive:       true,
			InferenceLevel: "normal",
			ServiceLevel:   "standard",
			WorkspaceID:    wsID,
		},
		{
			Name:           "DeepSeek Coder",
			Description:    strPtr("DeepSeek 代码专用模型"),
			Provider:       "deepseek",
			Model:          "deepseek-coder",
			MaxTokens:      8192,
			Temperature:    0.3,
			TopP:           0.95,
			IsDefault:      false,
			IsActive:       true,
			InferenceLevel: "advanced",
			ServiceLevel:   "premium",
			WorkspaceID:    wsID,
		},
		{
			Name:           "DeepSeek Reasoner",
			Description:    strPtr("DeepSeek 推理增强模型"),
			Provider:       "deepseek",
			Model:          "deepseek-reasoner",
			MaxTokens:      16384,
			Temperature:    0.2,
			TopP:           0.9,
			IsDefault:      false,
			IsActive:       true,
			InferenceLevel: "thinking-2",
			ServiceLevel:   "premium",
			WorkspaceID:    wsID,
		},
	}

	for i := range configs {
		if err := db.Create(&configs[i]).Error; err != nil {
			fmt.Printf("  WARN: failed to create agent config '%s': %v\n", configs[i].Name, err)
		}
	}
	fmt.Printf("  Created %d agent configs\n", len(configs))
}

func seedSkills(db *gorm.DB, wsID uint64) {
	var count int64
	db.Model(&model.Skill{}).Where("workspace_id = ?", wsID).Count(&count)
	if count > 0 {
		fmt.Println("  Skills already exist, skipping")
		return
	}

	skills := []model.Skill{
		{
			Name:        "代码审查",
			Description: strPtr("分析代码质量，识别潜在问题和改进建议"),
			SkillType:   "builtin",
			Version:     "1.0",
			Status:      "active",
			SkillMD: `## Step 1: 分析代码结构
**Tool:** analyze_code
**Input:** {"code": "{{code}}", "language": "{{language}}"}

## Step 2: 识别问题
分析代码中的潜在问题：
- 代码复杂度
- 潜在bug
- 安全漏洞
- 性能问题

## Step 3: 生成改进建议
提供具体的改进建议和优化方案。`,
			Parameters:  []byte(`[{"name":"code","type":"string","required":true,"description":"要审查的代码"},{"name":"language","type":"string","required":true,"description":"代码语言"}]`),
			Tags:        []byte(`["code","review","quality"]`),
			UsageCount:  12,
			IsShared:    true,
			IsPreset:    true,
			WorkspaceID: wsID,
		},
		{
			Name:        "需求分析",
			Description: strPtr("分析用户需求，提取关键信息，生成需求文档"),
			SkillType:   "builtin",
			Version:     "1.0",
			Status:      "active",
			SkillMD: `## Step 1: 提取关键信息
**Tool:** extract_requirements
**Input:** {"input": "{{input}}"}

## Step 2: 分类整理
将需求分类为：功能需求、非功能需求、约束条件

## Step 3: 生成需求文档
输出结构化的需求文档。`,
			Parameters:  []byte(`[{"name":"input","type":"string","required":true,"description":"原始需求文本"}]`),
			Tags:        []byte(`["requirement","analysis","document"]`),
			UsageCount:  8,
			IsShared:    true,
			IsPreset:    true,
			WorkspaceID: wsID,
		},
		{
			Name:        "文档生成",
			Description: strPtr("根据内容自动生成技术文档"),
			SkillType:   "builtin",
			Version:     "1.0",
			Status:      "active",
			SkillMD: `## Step 1: 分析内容
分析输入内容的结构和关键信息。

## Step 2: 生成大纲
根据分析结果生成文档大纲。

## Step 3: 生成文档
按大纲生成完整的Markdown文档。`,
			Parameters:  []byte(`[{"name":"content","type":"string","required":true,"description":"原始内容"},{"name":"format","type":"string","required":false,"description":"输出格式","default":"markdown"}]`),
			Tags:        []byte(`["documentation","generation"]`),
			UsageCount:  5,
			IsShared:    true,
			IsPreset:    true,
			WorkspaceID: wsID,
		},
		{
			Name:        "问题分类",
			Description: strPtr("自动分类问题类型和优先级"),
			SkillType:   "builtin",
			Version:     "1.0",
			Status:      "active",
			SkillMD: `## Step 1: 分析问题
**Tool:** analyze_issue
分析问题描述和标题。

## Step 2: 识别类型
分类为：Bug、功能请求、改进建议、文档问题

## Step 3: 建议优先级
根据影响范围和紧急程度建议优先级。`,
			Parameters:  []byte(`[{"name":"title","type":"string","required":true,"description":"问题标题"},{"name":"description","type":"string","required":true,"description":"问题描述"}]`),
			Tags:        []byte(`["issue","classification","triage"]`),
			UsageCount:  15,
			IsShared:    true,
			IsPreset:    true,
			WorkspaceID: wsID,
		},
		{
			Name:        "代码优化",
			Description: strPtr("分析代码性能，提供优化建议"),
			SkillType:   "builtin",
			Version:     "1.0",
			Status:      "active",
			SkillMD: `## Step 1: 性能分析
**Tool:** analyze_performance
分析代码性能瓶颈。

## Step 2: 识别优化点
找出可优化的代码段。

## Step 3: 生成优化方案
提供具体的优化代码和建议。`,
			Parameters:  []byte(`[{"name":"code","type":"string","required":true,"description":"要优化的代码"},{"name":"language","type":"string","required":true,"description":"代码语言"}]`),
			Tags:        []byte(`["code","optimization","performance"]`),
			UsageCount:  3,
			IsShared:    true,
			IsPreset:    true,
			WorkspaceID: wsID,
		},
		{
			Name:        "会议纪要",
			Description: strPtr("根据会议内容生成结构化会议纪要"),
			SkillType:   "builtin",
			Version:     "1.0",
			Status:      "active",
			SkillMD: `## Step 1: 提取信息
提取会议主题、参与人、时间等信息。

## Step 2: 整理要点
整理讨论要点、决策事项、行动项。

## Step 3: 生成纪要
生成结构化的会议纪要文档。`,
			Parameters:  []byte(`[{"name":"transcript","type":"string","required":true,"description":"会议记录文本"}]`),
			Tags:        []byte(`["meeting","minutes","summary"]`),
			UsageCount:  2,
			IsShared:    true,
			IsPreset:    true,
			WorkspaceID: wsID,
		},
	}

	for i := range skills {
		if err := db.Create(&skills[i]).Error; err != nil {
			fmt.Printf("  WARN: failed to create skill '%s': %v\n", skills[i].Name, err)
		}
	}
	fmt.Printf("  Created %d skills\n", len(skills))
}

func seedRuntimes(db *gorm.DB, wsID uint64) {
	var count int64
	db.Model(&model.Runtime{}).Where("workspace_id = ?", wsID).Count(&count)
	if count > 0 {
		fmt.Println("  Runtimes already exist, skipping")
		return
	}

	now := time.Now()
	runtimes := []model.Runtime{
		{
			Name:          "本地开发环境",
			RuntimeType:   "local_daemon",
			RuntimeMode:   "pull",
			Status:        "online",
			Health:        "online",
			Capacity:      4,
			CurrentLoad:   1,
			Version:       "1.2.0",
			HostInfo:      []byte(`{"os":"windows","cpu":"Intel i7-12700","memory":"32GB","disk":"512GB SSD"}`),
			LastHeartbeat: &now,
			WorkspaceID:   wsID,
		},
		{
			Name:          "云端推理集群",
			RuntimeType:   "cloud",
			RuntimeMode:   "push",
			Status:        "online",
			Health:        "online",
			Capacity:      16,
			CurrentLoad:   5,
			Version:       "2.0.1",
			HostInfo:      []byte(`{"provider":"aliyun","region":"cn-shanghai","gpu":"A100","memory":"64GB"}`),
			LastHeartbeat: &now,
			Endpoint:      strPtr("wss://ai-cluster.example.com/ws"),
			WorkspaceID:   wsID,
		},
	}

	for i := range runtimes {
		if err := db.Create(&runtimes[i]).Error; err != nil {
			fmt.Printf("  WARN: failed to create runtime '%s': %v\n", runtimes[i].Name, err)
		}
	}
	fmt.Printf("  Created %d runtimes\n", len(runtimes))
}

func seedSquads(db *gorm.DB, wsID, adminID uint64) {
	var count int64
	db.Model(&model.Squad{}).Where("workspace_id = ?", wsID).Count(&count)
	if count > 0 {
		fmt.Println("  Squads already exist, skipping")
		return
	}

	// Get agents in this workspace
	var agents []model.Agent
	db.Where("workspace_id = ?", wsID).Find(&agents)
	if len(agents) < 2 {
		fmt.Println("  Not enough agents for squads, skipping")
		return
	}

	// Squad 1: Sprint质量保障团队
	squad1 := model.Squad{
		WorkspaceID:   wsID,
		Name:          "Sprint质量保障团队",
		Description:   "负责Sprint期间的代码质量保障，包括代码审查、测试和Bug修复",
		LeaderAgentID: &agents[0].ID,
		Status:        "active",
		Goal:          "确保当前Sprint的所有交付物通过质量门禁",
		Config:        []byte(`{"max_parallel_tasks":3,"timeout_minutes":60,"auto_assign":true}`),
		BaseModel:     model.BaseModel{CreatedByID: &adminID},
	}
	if err := db.Create(&squad1).Error; err != nil {
		fmt.Printf("  WARN: failed to create squad '%s': %v\n", squad1.Name, err)
		return
	}

	// Add members to squad 1
	for i, agent := range agents {
		role := "member"
		if i == 0 {
			role = "leader"
		}
		db.Create(&model.SquadMember{
			SquadID: squad1.ID,
			AgentID: agent.ID,
			Role:    role,
			Status:  "active",
		})
	}

	// Squad 2: 需求分析团队
	if len(agents) >= 3 {
		squad2 := model.Squad{
			WorkspaceID:   wsID,
			Name:          "需求分析团队",
			Description:   "负责需求分析、拆解和任务分配",
			LeaderAgentID: &agents[1].ID,
			Status:        "active",
			Goal:          "将产品需求拆解为可执行的开发任务",
			Config:        []byte(`{"max_parallel_tasks":2,"timeout_minutes":30}`),
			BaseModel:     model.BaseModel{CreatedByID: &adminID},
		}
		if err := db.Create(&squad2).Error; err != nil {
			fmt.Printf("  WARN: failed to create squad '%s': %v\n", squad2.Name, err)
			return
		}

		for i := 0; i < 2 && i < len(agents); i++ {
			role := "member"
			if i == 0 {
				role = "leader"
			}
			db.Create(&model.SquadMember{
				SquadID: squad2.ID,
				AgentID: agents[i].ID,
				Role:    role,
				Status:  "active",
			})
		}
	}

	fmt.Println("  Created 2 squads with members")
}

func seedMemories(db *gorm.DB, wsID uint64) {
	var count int64
	db.Model(&model.MemoryEntry{}).Where("workspace_id = ?", wsID).Count(&count)
	if count > 0 {
		fmt.Println("  Memories already exist, skipping")
		return
	}

	memories := []model.MemoryEntry{
		{
			WorkspaceID: wsID,
			MemoryType:  "long_term",
			Scope:       "workspace",
			Content:     "项目使用Go语言后端 + Vue3前端技术栈，数据库为PostgreSQL，ORM使用GORM",
			Tags:        []byte(`["tech-stack","architecture"]`),
			ContextKey:  "project_architecture",
			ContextName: "项目架构",
		},
		{
			WorkspaceID: wsID,
			MemoryType:  "long_term",
			Scope:       "workspace",
			Content:     "API设计遵循RESTful规范，统一使用/api/v1前缀，工作空间路由为/api/v1/workspaces/:wsParam/",
			Tags:        []byte(`["api","conventions"]`),
			ContextKey:  "api_conventions",
			ContextName: "API规范",
		},
		{
			WorkspaceID: wsID,
			MemoryType:  "medium_term",
			Scope:       "workspace",
			Content:     "当前Sprint目标：完成AI Agent平台Phase 2开发，包括技能执行引擎、监控仪表盘和Squad协作功能",
			Tags:        []byte(`["sprint","goals"]`),
			ContextKey:  "current_sprint",
			ContextName: "当前Sprint",
		},
		{
			WorkspaceID: wsID,
			MemoryType:  "short_term",
			Scope:       "workspace",
			Content:     "最近一次代码审查发现：Squad执行时需要从context中获取userID，否则权限检查会失败",
			Tags:        []byte(`["code-review","bug-fix"]`),
			ContextKey:  "recent_review",
			ContextName: "最近审查",
			ExpiresAt:   timePtr(time.Now().Add(7 * 24 * time.Hour)),
		},
		{
			WorkspaceID: wsID,
			MemoryType:  "long_term",
			Scope:       "workspace",
			Content:     "用户角色体系：超级管理员(admin)、工作空间所有者(owner)、成员(member)、访客(guest)",
			Tags:        []byte(`["rbac","roles"]`),
			ContextKey:  "user_roles",
			ContextName: "用户角色",
		},
		{
			WorkspaceID: wsID,
			MemoryType:  "medium_term",
			Scope:       "workspace",
			Content:     "Agent权限模式：private(仅创建者)、public(所有人)、restricted(指定成员)，同工作空间成员可调用private模式Agent",
			Tags:        []byte(`["agent","permissions"]`),
			ContextKey:  "agent_permissions",
			ContextName: "Agent权限",
		},
		{
			WorkspaceID: wsID,
			MemoryType:  "short_term",
			Scope:       "workspace",
			Content:     "安全隔离测试已通过：使用sqlmock模拟数据库，验证不同用户在不同工作空间的数据隔离性",
			Tags:        []byte(`["testing","security"]`),
			ContextKey:  "test_status",
			ContextName: "测试状态",
			ExpiresAt:   timePtr(time.Now().Add(3 * 24 * time.Hour)),
		},
		{
			WorkspaceID: wsID,
			MemoryType:  "long_term",
			Scope:       "workspace",
			Content:     "预设技能系统包含：代码审查、需求分析、文档生成、问题分类、代码优化、会议纪要共6个内置技能",
			Tags:        []byte(`["skills","presets"]`),
			ContextKey:  "preset_skills",
			ContextName: "预设技能",
		},
	}

	for i := range memories {
		if err := db.Create(&memories[i]).Error; err != nil {
			fmt.Printf("  WARN: failed to create memory: %v\n", err)
		}
	}
	fmt.Printf("  Created %d memory entries\n", len(memories))
}

func seedAgentTasks(db *gorm.DB, wsID uint64) {
	var count int64
	db.Model(&model.AgentTask{}).Where("workspace_id = ?", wsID).Count(&count)
	if count > 0 {
		fmt.Println("  Agent tasks already exist, skipping")
		return
	}

	tasks := []model.AgentTask{
		{
			Title:       "审查用户认证模块代码",
			Description: strPtr("对auth_service.go进行代码审查，检查JWT token生成和验证逻辑"),
			Status:      "completed",
			Priority:    "high",
			Progress:    100,
			TaskType:    "code_review",
			InputData:   []byte(`{"file":"internal/service/auth_service.go","language":"go"}`),
			OutputData:  []byte(`{"issues_found":3,"suggestions":["增加token刷新机制","添加rate limiting","优化错误处理"]}`),
			WorkspaceID: wsID,
		},
		{
			Title:       "分析Sprint 3进度报告",
			Description: strPtr("汇总Sprint 3的任务完成情况和风险项"),
			Status:      "completed",
			Priority:    "normal",
			Progress:    100,
			TaskType:    "analyze_requirement",
			InputData:   []byte(`{"sprint_id":3}`),
			OutputData:  []byte(`{"completion_rate":78,"risk_items":["API文档未更新"],"velocity":42}`),
			WorkspaceID: wsID,
		},
		{
			Title:       "生成单元测试用例",
			Description: strPtr("为squad_service.go的StartExecution方法生成测试用例"),
			Status:      "running",
			Priority:    "normal",
			Progress:    65,
			TaskType:    "write_test",
			InputData:   []byte(`{"file":"internal/service/squad_service.go","method":"StartExecution"}`),
			WorkspaceID: wsID,
		},
		{
			Title:       "重构权限检查模块",
			Description: strPtr("优化CanInvoke方法，减少数据库查询次数"),
			Status:      "enqueue",
			Priority:    "high",
			Progress:    0,
			TaskType:    "refactor",
			InputData:   []byte(`{"file":"internal/ai/service/agent_service.go","method":"CanInvoke"}`),
			WorkspaceID: wsID,
		},
		{
			Title:       "分析性能瓶颈",
			Description: strPtr("分析Agent监控仪表盘接口的性能瓶颈"),
			Status:      "failed",
			Priority:    "low",
			Progress:    30,
			TaskType:    "analyze_requirement",
			ErrorInfo:   strPtr("timeout: LLM API response exceeded 30s limit"),
			WorkspaceID: wsID,
		},
		{
			Title:       "编写API文档",
			Description: strPtr("为Squad相关API编写OpenAPI规范文档"),
			Status:      "completed",
			Priority:    "normal",
			Progress:    100,
			TaskType:    "generate_doc",
			InputData:   []byte(`{"module":"squad","format":"openapi"}`),
			OutputData:  []byte(`{"endpoints":10,"schemas":5}`),
			WorkspaceID: wsID,
		},
	}

	now := time.Now()
	for i := range tasks {
		tasks[i].EnqueuedAt = now.Add(-time.Duration(len(tasks)-i) * time.Hour)
		if tasks[i].Status == "completed" {
			completed := now.Add(-time.Duration(len(tasks)-i) * 30 * time.Minute)
			tasks[i].StartedAt = &completed
			tasks[i].CompletedAt = &completed
		}
		if err := db.Create(&tasks[i]).Error; err != nil {
			fmt.Printf("  WARN: failed to create agent task '%s': %v\n", tasks[i].Title, err)
		}
	}
	fmt.Printf("  Created %d agent tasks\n", len(tasks))
}

func seedAgentActivities(db *gorm.DB, wsID uint64) {
	// AgentActivity has no workspace_id column — scope through the workspace's agents.
	var agent model.Agent
	if err := db.Where("workspace_id = ?", wsID).First(&agent).Error; err != nil {
		fmt.Println("  No agent found, skipping activities")
		return
	}

	var count int64
	db.Model(&model.AgentActivity{}).Where("agent_id = ?", agent.ID).Count(&count)
	if count > 0 {
		fmt.Println("  Agent activities already exist, skipping")
		return
	}

	activities := []model.AgentActivity{
		{
			AgentID:       agent.ID,
			Action:        "auto_triage",
			ResultSummary: "已将Issue #101分类为Bug，优先级设为High，分配给后端团队",
			ExecutedAt:    time.Now().Add(-2 * time.Hour),
			AgentName:     agent.Name,
			TaskContext:   strPtr("分析Issue #101: 用户登录时出现500错误"),
		},
		{
			AgentID:       agent.ID,
			Action:        "dispatch",
			ResultSummary: "已派发代码审查任务，审查auth模块的JWT实现",
			ExecutedAt:    time.Now().Add(-5 * time.Hour),
			AgentName:     agent.Name,
			TaskContext:   strPtr("审查internal/service/auth_service.go的JWT实现"),
		},
		{
			AgentID:       agent.ID,
			Action:        "summarize",
			ResultSummary: "Sprint 3进度：完成78%，剩余5个任务，预计本周五完成",
			ExecutedAt:    time.Now().Add(-24 * time.Hour),
			AgentName:     agent.Name,
		},
		{
			AgentID:       agent.ID,
			Action:        "auto_assign",
			ResultSummary: "已将Issue #105自动分配给张三（前端开发）",
			ExecutedAt:    time.Now().Add(-3 * time.Hour),
			AgentName:     agent.Name,
			Rating:        intPtr(1),
		},
	}

	for i := range activities {
		if err := db.Create(&activities[i]).Error; err != nil {
			fmt.Printf("  WARN: failed to create agent activity: %v\n", err)
		}
	}
	fmt.Printf("  Created %d agent activities\n", len(activities))
}

func seedAgentSessions(db *gorm.DB, wsID uint64) {
	var count int64
	db.Model(&model.AgentSession{}).Where("workspace_id = ?", wsID).Count(&count)
	if count > 0 {
		fmt.Println("  Agent sessions already exist, skipping")
		return
	}

	sessions := []model.AgentSession{
		{
			ID:            "sess-001",
			WorkspaceID:   wsID,
			AgentType:     "builtin",
			AgentRef:      strPtr("Triage Agent"),
			Status:        "completed",
			ModelUsed:     strPtr("deepseek-chat"),
			InputSummary:  strPtr("分析Issue #101的类型和优先级"),
			OutputSummary: strPtr("Bug, High Priority, Backend Team"),
			TokensInput:   450,
			TokensOutput:  120,
			CostUSD:       0.002,
			ToolsCalled:   []byte(`["analyze_issue","search_assignee"]`),
			StartedAt:     time.Now().Add(-2 * time.Hour),
			CompletedAt:   timePtr(time.Now().Add(-2*time.Hour + 15*time.Second)),
		},
		{
			ID:            "sess-002",
			WorkspaceID:   wsID,
			AgentType:     "builtin",
			AgentRef:      strPtr("Summary Agent"),
			Status:        "completed",
			ModelUsed:     strPtr("deepseek-chat"),
			InputSummary:  strPtr("汇总Sprint 3进度"),
			OutputSummary: strPtr("完成率78%，剩余5个任务"),
			TokensInput:   800,
			TokensOutput:  350,
			CostUSD:       0.005,
			ToolsCalled:   []byte(`["list_issues","analyze_sprint"]`),
			StartedAt:     time.Now().Add(-24 * time.Hour),
			CompletedAt:   timePtr(time.Now().Add(-24*time.Hour + 45*time.Second)),
		},
		{
			ID:           "sess-003",
			WorkspaceID:  wsID,
			AgentType:    "builtin",
			AgentRef:     strPtr("Assistant Agent"),
			Status:       "running",
			ModelUsed:    strPtr("deepseek-chat"),
			InputSummary: strPtr("生成单元测试用例"),
			TokensInput:  600,
			TokensOutput: 200,
			ToolsCalled:  []byte(`["analyze_code"]`),
			StartedAt:    time.Now().Add(-5 * time.Minute),
		},
	}

	for i := range sessions {
		if err := db.Create(&sessions[i]).Error; err != nil {
			fmt.Printf("  WARN: failed to create agent session: %v\n", err)
		}
	}
	fmt.Printf("  Created %d agent sessions\n", len(sessions))
}

func timePtr(t time.Time) *time.Time { return &t }
func intPtr(i int) *int              { return &i }

// Ensure json is used
var _ = json.Marshal

// SeedE2EFixtures ensures the workspace, project, and user required by the
// tests/ Playwright suite exist on a fresh database. It is idempotent and pins
// the project id to 2347 so that every spec's hardcoded URL stays valid.
//
// The fixtures are: workspace slug "qa-test", user qa_tester@reqmango.com
// (password Test@12345), project identifier "QAT" (id 2347), and the default
// state set that most specs expect.
// SeedDemoLoginUser ensures the account advertised in README and promo posts
// (demo@example.com / demo1234) exists and administers the demo workspaces.
// Runs independently of SeedDemoData so already-seeded databases get it too.
func SeedDemoLoginUser(db *gorm.DB) {
	var user model.User
	if db.Where("email = ?", "demo@example.com").First(&user).Error != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte("demo1234"), bcrypt.DefaultCost)
		user = model.User{
			Email:        "demo@example.com",
			Username:     "demo",
			DisplayName:  "Demo",
			PasswordHash: string(hash),
			IsActive:     true,
			IsSuperuser:  true,
		}
		if err := db.Create(&user).Error; err != nil {
			fmt.Printf("  WARN: failed to create demo user: %v\n", err)
			return
		}
	}

	var workspaces []model.Workspace
	db.Where("slug IN ?", []string{"reqmango-dev", "client-delivery", "infra"}).Find(&workspaces)
	for _, ws := range workspaces {
		var wsMember model.WorkspaceMember
		if db.Where("workspace_id = ? AND user_id = ?", ws.ID, user.ID).First(&wsMember).Error != nil {
			db.Create(&model.WorkspaceMember{
				WorkspaceID: ws.ID, UserID: user.ID, Role: common.RoleAdmin, IsActive: true,
			})
		}
		var projects []model.Project
		db.Where("workspace_id = ?", ws.ID).Find(&projects)
		for _, proj := range projects {
			var projMember model.ProjectMember
			if db.Where("project_id = ? AND user_id = ?", proj.ID, user.ID).First(&projMember).Error != nil {
				db.Create(&model.ProjectMember{
					ProjectID: proj.ID, UserID: user.ID, Role: common.RoleAdmin, IsActive: true,
				})
			}
		}
	}
}

func SeedE2EFixtures(db *gorm.DB) {
	// 1. User qa_tester (password Test@12345)
	var user model.User
	if db.Where("email = ?", "qa_tester@reqmango.com").First(&user).Error != nil {
		hash, _ := bcrypt.GenerateFromPassword([]byte("Test@12345"), bcrypt.DefaultCost)
		user = model.User{
			Email:        "qa_tester@reqmango.com",
			Username:     "qa_tester",
			DisplayName:  "QA Tester",
			PasswordHash: string(hash),
			IsActive:     true,
			IsSuperuser:  true,
		}
		if err := db.Create(&user).Error; err != nil {
			fmt.Printf("  WARN: failed to create qa_tester: %v\n", err)
			return
		}
	}

	// 2. Workspace qa-test
	var ws model.Workspace
	if db.Where("slug = ?", "qa-test").First(&ws).Error != nil {
		ws = model.Workspace{
			Name:     "QA Test",
			Slug:     "qa-test",
			OwnerID:  user.ID,
			Timezone: "UTC",
		}
		if err := db.Create(&ws).Error; err != nil {
			fmt.Printf("  WARN: failed to create qa-test workspace: %v\n", err)
			return
		}
	}

	// 3. Project QAT with fixed id 2347
	var project model.Project
	if db.Where("workspace_id = ? AND identifier = ?", ws.ID, "QAT").First(&project).Error != nil {
		// Advance the sequence past the desired id to avoid future collisions.
		db.Exec(`SELECT setval('projects_id_seq', GREATEST((SELECT COALESCE(MAX(id),0) FROM projects), 2347))`)
		project = model.Project{
			Name:        "QAT",
			Identifier:  "QAT",
			WorkspaceID: ws.ID,
			Description: strPtr("E2E test project (seeded for tests/ suite)"),
			Color:       "#6366F1",
		}
		// Force the id to 2347 so every spec's hardcoded URL stays valid.
		project.ID = 2347
		if err := db.Create(&project).Error; err != nil {
			fmt.Printf("  WARN: failed to create QAT project (id 2347): %v\n", err)
			return
		}
	}

	// 4. Ensure qa_tester is admin of the workspace and project
	var wsMember model.WorkspaceMember
	if db.Where("workspace_id = ? AND user_id = ?", ws.ID, user.ID).First(&wsMember).Error != nil {
		db.Create(&model.WorkspaceMember{
			WorkspaceID: ws.ID, UserID: user.ID, Role: common.RoleAdmin, IsActive: true,
		})
	}
	var projMember model.ProjectMember
	if db.Where("project_id = ? AND user_id = ?", project.ID, user.ID).First(&projMember).Error != nil {
		db.Create(&model.ProjectMember{
			ProjectID: project.ID, UserID: user.ID, Role: common.RoleAdmin, IsActive: true,
		})
	}

	// 5. Create default states for the project (required by issue specs)
	var stateCount int64
	db.Model(&model.State{}).Where("project_id = ?", project.ID).Count(&stateCount)
	if stateCount == 0 {
		type stateDef struct{ name, group string }
		defs := []stateDef{
			{"待处理", "backlog"},
			{"待办", "unstarted"},
			{"进行中", "started"},
			{"评审中", "started"},
			{"已完成", "completed"},
			{"已取消", "cancelled"},
		}
		for i, s := range defs {
			db.Create(&model.State{
				Name:        s.name,
				Group:       s.group,
				ProjectID:   &project.ID,
				WorkspaceID: ws.ID,
				Sequence:    i + 1,
				IsDefault:   i == 0,
			})
		}
	}

	// 6. Create default issue types for the project
	var typeCount int64
	db.Model(&model.IssueType{}).Where("project_id = ?", project.ID).Count(&typeCount)
	if typeCount == 0 {
		typeDefs := []struct {
			name, icon string
			seq        int
		}{
			{"Epic", "🎯", 1},
			{"Feature", "✨", 2},
			{"Story", "📖", 3},
			{"Bug", "🐛", 4},
			{"Task", "📋", 5},
			{"Spike", "🔬", 6},
		}
		for _, td := range typeDefs {
			db.Create(&model.IssueType{
				Name:        td.name,
				Icon:        td.icon,
				ProjectID:   &project.ID,
				WorkspaceID: ws.ID,
				Sequence:    td.seq,
			})
		}
	}

	fmt.Println("E2E fixtures ensured (qa-test workspace, project 2347, qa_tester user)")
}
