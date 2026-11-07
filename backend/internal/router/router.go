package router

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	aihandler "github.com/reqmango/backend/internal/ai/handler"
	"github.com/reqmango/backend/internal/ai/harness"
	"github.com/reqmango/backend/internal/ai/llm"
	"github.com/reqmango/backend/internal/ai/registry"
	aiservice "github.com/reqmango/backend/internal/ai/service"
	"github.com/reqmango/backend/internal/client"
	"github.com/reqmango/backend/internal/config"
	"github.com/reqmango/backend/internal/handler"
	"github.com/reqmango/backend/internal/middleware"
	"github.com/reqmango/backend/internal/model"
	"github.com/reqmango/backend/internal/rql"
	"github.com/reqmango/backend/internal/scheduler"
	"github.com/reqmango/backend/internal/service"
	"github.com/reqmango/backend/internal/version"
	"gorm.io/gorm"
)

var (
	memScheduler *scheduler.MemoryScheduler
	schedulerMu  sync.Mutex
)

// SetupRoutes initializes all services, handlers, and routes.
func SetupRoutes(r *gin.Engine, db *gorm.DB, cfg *config.Config) {
	// Prevent Gin from sending 301 redirects for trailing slashes,
	// which causes issues when the frontend sends requests with
	// inconsistent slash patterns.
	r.RedirectTrailingSlash = false

	// Initialize services
	authSvc := service.NewAuthService(db, cfg)
	workspaceSvc := service.NewWorkspaceService(db)
	projectSvc := service.NewProjectService(db)
	settingsSvc := service.NewProjectSettingsService(db)
	notificationSvc := service.NewNotificationService(db)
	webhookSvc := service.NewWebhookService(db)
	automationSvc := service.NewAutomationService(db)
	slackSvc := service.NewSlackService(db)
	issueSvc := service.NewIssueService(db, notificationSvc, webhookSvc, automationSvc, slackSvc)
	cycleSvc := service.NewCycleService(db, notificationSvc)
	moduleSvc := service.NewModuleService(db)
	issueTypeSvc := service.NewIssueTypeService(db)
	customFieldSvc := service.NewCustomFieldService(db)
	templateSvc := service.NewProjectTemplateService(db)
	typeTemplateSvc := service.NewTypeTemplateService(db)
	relationSvc := service.NewRelationService(db)
	approvalSvc := service.NewApprovalService(db, notificationSvc)
	approvalH := handler.NewApprovalHandler(approvalSvc)
	commentSvc := service.NewCommentService(db, notificationSvc)
	savedViewSvc := service.NewSavedViewService(db)
	searchTemplateSvc := service.NewSearchTemplateService(db)
	gitSvc := service.NewGitService(db)
	projectIssueTypeH := handler.NewProjectIssueTypeHandler(issueTypeSvc)
	pageSvc := service.NewPageService(db)
	pageVersionSvc := service.NewPageVersionService(db)
	pageTemplateSvc := service.NewPageTemplateService(db)
	witSvc := service.NewWorkItemTemplateService(db)
	conditionalFieldSvc := service.NewConditionalFieldService(db)
	releaseSvc := service.NewReleaseService(db)
	estimateSvc := service.NewEstimateService(db)
	attachmentSvc := service.NewAttachmentService(db)
	timeTrackSvc := service.NewTimeTrackService(db)
	recurrenceSvc := service.NewRecurrenceService(db)
	reportSvc := service.NewReportService(db)
	savedReportSvc := service.NewSavedReportService(db)
	metricSvc := service.NewMetricService(db)
	pageTabSvc := service.NewProjectPageTabService(db)
	intakeH := handler.NewIntakeHandler(db, issueSvc)
	reportH := handler.NewReportHandler(reportSvc)
	savedReportH := handler.NewSavedReportHandler(savedReportSvc)
	metricH := handler.NewMetricHandler(metricSvc)
	dashboardSvc := service.NewDashboardService(db)
	dashboardH := handler.NewDashboardHandler(dashboardSvc)
	// --- AI Module (merged from agent-service) ---
	// Initialize LLM client
	llmClient := llm.NewLLMClient(cfg.AIAPIKey, cfg.AIModel, cfg.AIBaseURL, cfg.AIProvider)

	// Initialize AI services
	aiSvc := aiservice.NewAIService(db, llmClient)
	dashboardSvc.SetAISummaryProvider(&service.AIServiceSummaryProvider{AI: aiSvc, DB: db})
	agentSvc := aiservice.NewAgentService(db, llmClient, aiSvc)
	loopSvc := aiservice.NewLoopService(db, agentSvc)

	// Initialize Memory service and inject via setter (to avoid import cycle)
	memSvc := service.NewMemoryService(db, llmClient)
	aiSvc.SetMemoryService(memSvc)
	intakeH.SetSpecAI(aiSvc)
	agentSvc.SetMemoryService(memSvc)

	// Start memory cleanup scheduler
	schedulerMu.Lock()
	memScheduler = scheduler.NewMemoryScheduler(memSvc)
	memScheduler.Start(context.Background())
	schedulerMu.Unlock()

	// Seed agent registry
	reg := registry.NewRegistry(db)
	_ = reg.SeedDefaults(nil)

	// AgentClient adapter (now calls local AgentService instead of HTTP proxy)
	agentClient := client.NewAgentClient(agentSvc)
	automationSvc.SetAgentService(agentClient)     // break circular dependency: automation -> agent -> issue -> automation
	commentSvc.SetAgentService(agentClient)        // enable @agent-name mention handling in comments
	commentSvc.SetAutomationService(automationSvc) // enable comment_added automation trigger

	// Chat & Messages: construct chatSvc, inject agent client + memory service,
	// and wire the state-change hook into issueSvc (setter injection, mirrors commentSvc).
	chatSvc := service.NewChatService(db, memSvc)
	chatSvc.SetAgentClient(agentClient)
	chatSvc.StartDebouncerCleanup(context.Background())
	issueSvc.SetChatService(chatSvc)

	// Start the scheduled automation trigger background scheduler
	automationSvc.StartScheduler(context.Background())
	mcpSvc := service.NewMCPService(db)
	githubSvc := service.NewGitHubService(db)
	roleSvc := service.NewRoleService(db)
	fieldPermSvc := service.NewFieldPermissionService(db)
	pluginSvc := service.NewPluginService(db)

	// --- Agent-Project Integration services ---
	agentMemberSvc := service.NewAgentMemberService(db)
	agentBudgetSvc := service.NewAgentCostBudgetService(db)
	agentSLASvc := service.NewAgentSLAService(db)
	agentDecisionSvc := service.NewAgentDecisionService(db)
	agentTaskSvc := service.NewAgentTaskService(db)
	agentTaskSvc.SetPresenceService(agentSvc)
	contextPayloadSvc := service.NewContextPayloadService(db)
	stateWorkflowSvc := service.NewStateWorkflowService(db)
	workflowSvc := service.NewWorkflowService(db, contextPayloadSvc, agentDecisionSvc, agentBudgetSvc)
	workflowExecutor := service.NewWorkflowExecutor(db, workflowSvc, contextPayloadSvc, agentDecisionSvc, agentBudgetSvc, agentSLASvc)
	workflowExecutor.SetAgentExecutor(&aiAgentExecutorAdapter{agentSvc: agentSvc})
	issueAgentSvc := service.NewIssueAgentService(db, agentTaskSvc, agentBudgetSvc, agentSLASvc, agentDecisionSvc)

	// Initialize handlers
	authH := handler.NewAuthHandler(authSvc)
	workspaceH := handler.NewWorkspaceHandler(workspaceSvc)
	workspaceH.SetPMAgentEnsurer(agentSvc)
	projectH := handler.NewProjectHandler(projectSvc, templateSvc)
	settingsH := handler.NewProjectSettingsHandler(settingsSvc)
	issueH := handler.NewIssueHandler(issueSvc)
	issueH.SetAutoTriager(agentSvc)
	agentSvc.SetActivityNotifier(func(userID uint64, event string, data interface{}) {
		b, _ := json.Marshal(data)
		service.SSE.SendToUser(userID, event, string(b))
	})
	cycleH := handler.NewCycleHandler(cycleSvc)
	moduleH := handler.NewModuleHandler(moduleSvc)
	issueTypeH := handler.NewIssueTypeHandler(issueTypeSvc)
	customFieldH := handler.NewCustomFieldHandler(customFieldSvc)
	templateH := handler.NewProjectTemplateHandler(templateSvc)
	typeTemplateH := handler.NewTypeTemplateHandler(typeTemplateSvc)
	relationH := handler.NewRelationHandler(relationSvc)
	workflowH := handler.NewWorkflowHandler(stateWorkflowSvc, workflowSvc, workflowExecutor)
	agentMemberH := handler.NewAgentMemberHandler(agentMemberSvc)
	issueAgentH := handler.NewIssueAgentHandler(issueAgentSvc)
	budgetH := handler.NewBudgetHandler(agentBudgetSvc)
	slaH := handler.NewSLAHandler(agentSLASvc)
	decisionH := handler.NewDecisionHandler(agentDecisionSvc)
	commentH := handler.NewCommentHandler(commentSvc)
	chatH := handler.NewChatHandler(chatSvc, db, cfg.SecretKey)
	notificationH := handler.NewNotificationHandler(notificationSvc)
	savedViewH := handler.NewSavedViewHandler(savedViewSvc)
	searchTemplateH := handler.NewSearchTemplateHandler(searchTemplateSvc)
	pageH := handler.NewPageHandler(pageSvc)
	pageVersionH := handler.NewPageVersionHandler(pageVersionSvc)
	pageTemplateH := handler.NewPageTemplateHandler(pageTemplateSvc)
	conditionalFieldH := handler.NewConditionalFieldHandler(conditionalFieldSvc)
	witH := handler.NewWorkItemTemplateHandler(witSvc)
	releaseH := handler.NewReleaseHandler(releaseSvc)
	initiativeH := handler.NewInitiativeHandler(db)
	wsAnalyticsH := handler.NewWorkspaceAnalyticsHandler(db, service.NewWorkspaceAnalyticsService(db))
	estimateH := handler.NewEstimateHandler(estimateSvc)
	attachmentH := handler.NewAttachmentHandler(attachmentSvc)
	webhookH := handler.NewWebhookHandler(webhookSvc)
	timeTrackH := handler.NewTimeTrackHandler(timeTrackSvc)
	recurrenceH := handler.NewRecurrenceHandler(recurrenceSvc)
	pageTabH := handler.NewProjectPageTabHandler(pageTabSvc)
	mcpH := handler.NewMCPHandler(mcpSvc)
	// RequirePermission resolves no workspace on these routes, so AI and agent
	// endpoints check membership explicitly.
	wsMember := middleware.RequireWorkspaceMember(db)
	wsAdmin := middleware.RequireWorkspaceAdmin(db)
	// Workspace settings mutations need settings:manage (admin); members only
	// have settings:view.
	settingsManage := middleware.RequireWorkspacePermission(db, "settings:manage")
	projectAccess := middleware.RequireProjectAccess(db)
	issueAccess := middleware.RequireIssueAccess(db)
	pageAccess := middleware.RequirePageAccess(db)
	wsWorkflow := middleware.RequireWorkspaceWorkflow(db)
	githubH := handler.NewGitHubHandler(githubSvc)
	slackH := handler.NewSlackHandler(slackSvc)
	roleH := handler.NewRoleHandler(roleSvc, db)
	fieldPermH := handler.NewFieldPermissionHandler(fieldPermSvc)
	pluginH := handler.NewPluginHandler(pluginSvc)
	automationH := handler.NewAutomationHandler(automationSvc, db)
	gitIntegrationH := handler.NewGitIntegrationHandler(gitSvc)
	gitWebhookH := handler.NewGitWebhookHandler(gitSvc)

	// AI handlers
	aiH := aihandler.NewAIHandler(aiSvc, db, cfg.AIAPIKey, cfg.AIModel, cfg.AIBaseURL, cfg.AIProvider)
	agentH := aihandler.NewAgentHandler(agentSvc)
	loopH := aihandler.NewAgentLoopHandler(loopSvc)
	pipelineH := aihandler.NewAgentPipelineHandler(db, reg, agentSvc)
	sessionH := aihandler.NewAgentSessionHandler(db)

	// JWT middleware
	authMiddleware := middleware.ScopedAuthMiddleware(db, cfg.SecretKey)

	// Probes are registered before the language middleware and the rate limiter
	// so they stay cheap and cannot be throttled by an orchestrator's polling.
	healthH := handler.NewHealthHandler(db)
	r.GET("/health", healthH.Live)
	r.GET("/readyz", healthH.Ready)

	// Build identity. Registered next to the probes (and before the rate
	// limiter) so an operator can answer "which version is deployed?" without
	// a token and without being throttled.
	r.GET("/version", func(c *gin.Context) {
		c.JSON(200, version.Current())
	})

	// Language detection middleware
	r.Use(middleware.LanguageMiddleware())

	// Request metrics and slow-request logging. Grouped by route template, so
	// cardinality is bounded by the routes registered below.
	metricsCollector := middleware.NewMetricsCollector(time.Duration(cfg.SlowRequestMS) * time.Millisecond)
	r.Use(metricsCollector.Middleware())

	// Rate limiter: global middleware (applied before routes)
	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRequests, cfg.RateLimitWindowSec)
	r.Use(rateLimiter.Middleware())

	// SSE endpoint for real-time notifications
	sseH := handler.NewSSEHandler(db, cfg)
	v1SSE := r.Group("/api/v1")
	v1SSE.GET("/sse", sseH.Connect)
	// ==================== API v1 ====================
	v1 := r.Group("/api/v1")
	{
		// ---- Operational metrics (superuser only) ----
		v1.GET("/metrics", authMiddleware, middleware.MetricsHandler(metricsCollector))

		// ---- Intake (public) ----
		v1.POST("/intake/:projectId", intakeH.Submit)
		v1.GET("/intake/:projectId/info", intakeH.Info)
		v1.POST("/intake-channels/:token/webhook", intakeH.Webhook)
		v1.POST("/intake-channels/:token/email", intakeH.Email)

		// ---- GitHub Webhook (public) ----
		v1.POST("/webhook/github/:id", githubH.Webhook)

		// ---- Git Webhook (public) ----
		v1.POST("/webhook/git/:projectId", gitWebhookH.GitHubWebhook)

		// ---- Auth (public + protected) ----
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authH.Register)
			auth.POST("/login", authH.Login)
			auth.GET("/me", authMiddleware, authH.GetCurrentUser)
		}

		// ---- Personal Access Tokens ----
		patSvc := service.NewPATService(db)
		patH := handler.NewPATHandler(patSvc)
		tokens := v1.Group("/auth/tokens", authMiddleware)
		{
			tokens.GET("", patH.List)
			tokens.POST("", patH.Create)
			tokens.DELETE("/:id", patH.Revoke)
		}

		// ---- Workspaces (protected) ----
		workspaces := v1.Group("/workspaces", authMiddleware)
		{
			workspaces.GET("", workspaceH.List)
			workspaces.POST("", workspaceH.Create)
			workspaces.GET("/:wsParam", workspaceH.Get)       // slug or numeric ID
			workspaces.PATCH("/:wsParam", workspaceH.Update)  // numeric ID or slug
			workspaces.DELETE("/:wsParam", workspaceH.Delete) // numeric ID or slug
			workspaces.GET("/:wsParam/members", workspaceH.ListMembers)
			workspaces.GET("/:wsParam/member-candidates", middleware.RequireWorkspacePermission(db, "member:manage"), workspaceH.ListMemberCandidates)
			workspaces.GET("/:wsParam/my-permissions", handler.MyWorkspacePermissions(db))
			workspaces.GET("/:wsParam/analytics", wsAnalyticsH.Get)
			workspaces.GET("/:wsParam/analytics/loop", wsAnalyticsH.Loop)
			workspaces.POST("/:wsParam/members", workspaceH.AddMember)
			workspaces.PATCH("/:wsParam/members/:userId", workspaceH.UpdateMember)
			workspaces.DELETE("/:wsParam/members/:userId", workspaceH.RemoveMember)
			// AI Config
			workspaces.GET("/:wsParam/ai-config", wsMember, aiH.GetAIConfig)
			workspaces.PUT("/:wsParam/ai-config", wsAdmin, aiH.UpdateAIConfig)
			workspaces.POST("/:wsParam/ai-config/test", wsAdmin, aiH.TestAIConfig)

			// Initiatives
			workspaces.POST("/:wsParam/initiatives", initiativeH.Create)
			workspaces.GET("/:wsParam/initiatives", initiativeH.List)
			workspaces.GET("/:wsParam/initiatives/search", initiativeH.Search)

			// Agent Templates (new)
			agentTemplateSvc := service.NewAgentTemplateService(db)
			agentTemplateH := handler.NewAgentTemplateHandler(agentTemplateSvc)
			workspaces.GET("/:wsParam/agent-templates", agentTemplateH.ListAgentTemplates)
			workspaces.POST("/:wsParam/agent-templates", agentTemplateH.CreateAgentTemplate)
			workspaces.GET("/:wsParam/agent-templates/:templateId", agentTemplateH.GetAgentTemplate)
			workspaces.PUT("/:wsParam/agent-templates/:templateId", agentTemplateH.UpdateAgentTemplate)
			workspaces.DELETE("/:wsParam/agent-templates/:templateId", agentTemplateH.DeleteAgentTemplate)
			// Preset templates
			workspaces.POST("/:wsParam/agent-templates/presets/initialize", agentTemplateH.InitializePresetTemplates)

			// Agent Configs (new)
			agentConfigSvc := service.NewAgentConfigService(db)
			agentConfigH := handler.NewAgentConfigHandler(agentConfigSvc)
			workspaces.GET("/:wsParam/agent-configs", agentConfigH.ListAgentConfigs)
			workspaces.POST("/:wsParam/agent-configs", agentConfigH.CreateAgentConfig)
			workspaces.GET("/:wsParam/agent-configs/:configId", agentConfigH.GetAgentConfig)
			workspaces.PUT("/:wsParam/agent-configs/:configId", agentConfigH.UpdateAgentConfig)
			workspaces.DELETE("/:wsParam/agent-configs/:configId", agentConfigH.DeleteAgentConfig)
			workspaces.GET("/:wsParam/agent-configs/default", agentConfigH.GetDefaultAgentConfig)

			// Runtimes (new)
			runtimeSvc := service.NewRuntimeService(db)
			runtimeH := handler.NewRuntimeHandler(runtimeSvc)
			workspaces.GET("/:wsParam/runtimes", runtimeH.ListRuntimes)
			workspaces.POST("/:wsParam/runtimes", runtimeH.CreateRuntime)
			workspaces.GET("/:wsParam/runtimes/:runtimeId", runtimeH.GetRuntime)
			workspaces.PUT("/:wsParam/runtimes/:runtimeId", runtimeH.UpdateRuntime)
			workspaces.DELETE("/:wsParam/runtimes/:runtimeId", runtimeH.DeleteRuntime)
			workspaces.POST("/:wsParam/runtimes/register", runtimeH.RegisterRuntime)
			workspaces.POST("/:wsParam/runtimes/:runtimeId/heartbeat", runtimeH.Heartbeat)
			workspaces.GET("/:wsParam/runtimes/available", runtimeH.FindAvailableRuntime)
			// Runtime health check & scheduling
			workspaces.POST("/:wsParam/runtimes/health-check", runtimeH.HealthCheck)
			workspaces.POST("/:wsParam/runtimes/health-check/global", runtimeH.GlobalHealthCheck)
			workspaces.POST("/:wsParam/runtimes/schedule", runtimeH.ScheduleTask)
			workspaces.POST("/:wsParam/runtimes/:runtimeId/release", runtimeH.ReleaseTask)
			workspaces.GET("/:wsParam/runtimes/stats", runtimeH.GetRuntimeStats)

			// Skills (new)
			skillSvc := service.NewSkillService(db)
			// Create tool executor and skill executor
			toolExecutor := harness.NewDatabaseToolExecutor(db)
			skillExecutor := harness.NewSkillExecutor(db, toolExecutor)
			// Wrap harness executor to implement service.SkillExecutor interface
			skillSvc.SetExecutor(&harnessSkillExecutorAdapter{executor: skillExecutor})
			// Inject SkillService into AgentService for skill integration
			agentSvc.SetSkillExecutor(skillSvc)
			skillH := handler.NewSkillHandler(skillSvc)
			workspaces.GET("/:wsParam/skills", skillH.ListSkills)
			workspaces.POST("/:wsParam/skills", skillH.CreateSkill)
			// Preset skills (must be registered before :skillId routes to avoid conflict)
			workspaces.GET("/:wsParam/skills/presets", skillH.ListPresetSkills)
			workspaces.POST("/:wsParam/skills/presets/initialize", skillH.InitializePresetSkills)
			// Skill execution logs
			workspaces.GET("/:wsParam/skills/execution-logs", skillH.ListSkillExecutionLogs)
			// Dynamic skill ID routes (must come last)
			workspaces.GET("/:wsParam/skills/:skillId", skillH.GetSkill)
			workspaces.PUT("/:wsParam/skills/:skillId", skillH.UpdateSkill)
			workspaces.DELETE("/:wsParam/skills/:skillId", skillH.DeleteSkill)
			workspaces.POST("/:wsParam/skills/:skillId/execute", skillH.ExecuteSkill)
			workspaces.GET("/:wsParam/skills/:skillId/execution-logs", skillH.ListSkillExecutionLogs)

			// Agent Tasks (new)
			agentTaskH := handler.NewAgentTaskHandler(agentTaskSvc)
			workspaces.GET("/:wsParam/agent-tasks", agentTaskH.ListAgentTasks)
			workspaces.POST("/:wsParam/agent-tasks", agentTaskH.CreateAgentTask)
			workspaces.GET("/:wsParam/agent-tasks/:taskId", agentTaskH.GetAgentTask)
			workspaces.PUT("/:wsParam/agent-tasks/:taskId", agentTaskH.UpdateAgentTask)
			workspaces.DELETE("/:wsParam/agent-tasks/:taskId", agentTaskH.DeleteAgentTask)
			workspaces.POST("/:wsParam/agent-tasks/:taskId/claim", agentTaskH.ClaimAgentTask)
			workspaces.POST("/:wsParam/agent-tasks/:taskId/start", agentTaskH.StartAgentTask)
			workspaces.POST("/:wsParam/agent-tasks/:taskId/complete", agentTaskH.CompleteAgentTask)
			workspaces.POST("/:wsParam/agent-tasks/:taskId/fail", agentTaskH.FailAgentTask)
			workspaces.POST("/:wsParam/agent-tasks/:taskId/cancel", agentTaskH.CancelAgentTask)
			workspaces.GET("/:wsParam/agent-tasks/:taskId/logs", agentTaskH.GetTaskLogs)
			workspaces.POST("/:wsParam/agent-tasks/:taskId/retry", agentTaskH.RetryAgentTask)
			workspaces.POST("/:wsParam/agent-tasks/:taskId/rerun", agentTaskH.RerunAgentTask)

			// Agent Performance Analytics (PRD P4-010)
			agentPerfSvc := service.NewAgentPerformanceService(db)
			agentPerfH := handler.NewAgentPerformanceHandler(agentPerfSvc)
			workspaces.GET("/:wsParam/agent-performance/overview", agentPerfH.Overview)
			workspaces.GET("/:wsParam/agent-performance/by-template", agentPerfH.ByTemplate)
			workspaces.GET("/:wsParam/agent-performance/timeline", agentPerfH.Timeline)
			workspaces.GET("/:wsParam/agent-performance/failures", agentPerfH.FailureBreakdown)
			workspaces.GET("/:wsParam/agent-performance/suggestions", agentPerfH.SuggestionAcceptance)

			// Squads (new)
			squadSvc := service.NewSquadService(db)
			// Create adapter to connect AgentService to SquadService
			squadSvc.SetAgentExecutor(&aiAgentExecutorAdapter{agentSvc: agentSvc})
			squadH := handler.NewSquadHandler(squadSvc)
			workspaces.GET("/:wsParam/squads", squadH.ListSquads)
			workspaces.POST("/:wsParam/squads", squadH.CreateSquad)
			workspaces.GET("/:wsParam/squads/:squadId", squadH.GetSquad)
			workspaces.PUT("/:wsParam/squads/:squadId", squadH.UpdateSquad)
			workspaces.DELETE("/:wsParam/squads/:squadId", squadH.DeleteSquad)
			workspaces.POST("/:wsParam/squads/:squadId/members", squadH.AddMember)
			workspaces.DELETE("/:wsParam/squads/:squadId/members/:memberId", squadH.RemoveMember)
			workspaces.POST("/:wsParam/squads/:squadId/executions", squadH.StartExecution)
			workspaces.GET("/:wsParam/squads/:squadId/executions", squadH.ListExecutions)
			workspaces.GET("/:wsParam/squads/:squadId/executions/:executionId", squadH.GetExecution)
			workspaces.DELETE("/:wsParam/squads/:squadId/executions/:executionId", squadH.CancelExecution)

			// Autopilot (new)
			autopilotSvc := service.NewAutopilotService(db)
			autopilotH := handler.NewAutopilotHandler(autopilotSvc)
			// The handlers read the resource by id only, so bind :taskId and
			// :executionId to :wsParam here; otherwise any workspace admin who
			// guessed an id could read another tenant's task and executions.
			autopilotScope := middleware.RequireWorkspaceAutopilotTask(db)
			workspaces.GET("/:wsParam/autopilot-tasks", autopilotH.ListTasks)
			workspaces.POST("/:wsParam/autopilot-tasks", autopilotH.CreateTask)
			workspaces.GET("/:wsParam/autopilot-tasks/:taskId", autopilotScope, autopilotH.GetTask)
			workspaces.PUT("/:wsParam/autopilot-tasks/:taskId", autopilotScope, autopilotH.UpdateTask)
			workspaces.DELETE("/:wsParam/autopilot-tasks/:taskId", autopilotScope, autopilotH.DeleteTask)
			workspaces.POST("/:wsParam/autopilot-tasks/:taskId/toggle", autopilotScope, autopilotH.ToggleTask)
			workspaces.POST("/:wsParam/autopilot-tasks/:taskId/execute", autopilotScope, autopilotH.ExecuteTask)
			workspaces.GET("/:wsParam/autopilot-tasks/:taskId/executions", autopilotScope, autopilotH.ListExecutions)
			workspaces.GET("/:wsParam/autopilot-tasks/:taskId/executions/:executionId", autopilotScope, autopilotH.GetExecution)
			// Public webhook trigger (no auth) so external systems can fire it.
			v1.POST("/autopilot/webhook/:token", autopilotH.TriggerWebhook)

			// Tools (new)
			toolSvc := service.NewToolService(db)
			toolSvc.SetMCPService(mcpSvc)
			toolH := handler.NewToolHandlerWithService(toolSvc)
			workspaces.GET("/:wsParam/tools", toolH.ListTools)
			workspaces.POST("/:wsParam/tools", toolH.CreateTool)
			workspaces.GET("/:wsParam/tools/:toolId", toolH.GetTool)
			workspaces.PUT("/:wsParam/tools/:toolId", toolH.UpdateTool)
			workspaces.DELETE("/:wsParam/tools/:toolId", toolH.DeleteTool)
			workspaces.POST("/:wsParam/tools/call", toolH.CallTool)
			workspaces.GET("/:wsParam/tools/call-logs", toolH.GetToolCallLogs)

			// Agent Sessions
			workspaces.GET("/:wsParam/agent-sessions", sessionH.List)
			workspaces.GET("/:wsParam/agent-sessions/:sessionId", sessionH.Get)

			// Agent Loops
			workspaces.GET("/:wsParam/loops", loopH.List)
			workspaces.POST("/:wsParam/loops", loopH.Create)
			workspaces.GET("/:wsParam/loops/:id", loopH.Get)
			workspaces.PUT("/:wsParam/loops/:id", loopH.Update)
			workspaces.DELETE("/:wsParam/loops/:id", loopH.Delete)
			workspaces.POST("/:wsParam/loops/:id/start", loopH.Start)
			workspaces.POST("/:wsParam/loops/runs/:runId/stop", loopH.Stop)
			workspaces.GET("/:wsParam/loops/:id/runs", loopH.GetRuns)
			workspaces.GET("/:wsParam/loops/runs/:runId", loopH.GetRun)

			// Agent Pipelines
			workspaces.GET("/:wsParam/pipelines", pipelineH.List)
			workspaces.POST("/:wsParam/pipelines", pipelineH.Create)
			workspaces.GET("/:wsParam/pipelines/:id", pipelineH.Get)
			workspaces.PUT("/:wsParam/pipelines/:id", pipelineH.Update)
			workspaces.DELETE("/:wsParam/pipelines/:id", pipelineH.Delete)
			workspaces.POST("/:wsParam/pipelines/:id/run", pipelineH.Run)
			workspaces.GET("/:wsParam/pipelines/:id/runs", pipelineH.GetRuns)
			workspaces.GET("/:wsParam/pipelines/runs/:runId", pipelineH.GetRun)

			// Agent routes (must come after other agent-related routes to avoid parameter conflicts)
			workspaces.GET("/:wsParam/agents", wsMember, agentH.List)
			workspaces.POST("/:wsParam/agents", wsMember, agentH.Create)
			workspaces.POST("/:wsParam/agents/ensure-pm", wsMember, agentH.EnsurePM)
			workspaces.GET("/:wsParam/agents/activity", wsMember, agentH.ListWorkspaceActivity)
			workspaces.PATCH("/:wsParam/agents/activity/:id/feedback", wsMember, agentH.UpdateActivityFeedback)
			workspaces.GET("/:wsParam/agents/:id", wsMember, agentH.GetByID)
			workspaces.PUT("/:wsParam/agents/:id", wsMember, agentH.Update)
			workspaces.DELETE("/:wsParam/agents/:id", wsMember, agentH.Delete)
			workspaces.POST("/:wsParam/agents/:id/dispatch", wsMember, agentH.Dispatch)
			workspaces.GET("/:wsParam/agents/:id/activity", wsMember, agentH.GetActivity)
			workspaces.POST("/:wsParam/agents/:id/auto-triage", wsMember, agentH.AutoTriage)
			workspaces.POST("/:wsParam/agents/:id/auto-assign", wsMember, agentH.AutoAssign)
			// Agent Presence routes
			workspaces.GET("/:wsParam/agents/presence", wsMember, agentH.ListPresence)
			workspaces.GET("/:wsParam/agents/:id/presence", wsMember, agentH.GetPresence)
			workspaces.POST("/:wsParam/agents/:id/heartbeat", wsMember, agentH.Heartbeat)
			workspaces.PUT("/:wsParam/agents/:id/availability", wsMember, agentH.UpdateAvailability)
			workspaces.PUT("/:wsParam/agents/:id/workload", wsMember, agentH.UpdateWorkload)
			workspaces.GET("/:wsParam/agents/:id/snapshots", wsMember, agentH.GetSnapshots)
			workspaces.POST("/:wsParam/agents/:id/snapshots", wsMember, agentH.CreateSnapshot)

			// Agent Monitoring Dashboard
			workspaces.GET("/:wsParam/agents/monitoring", wsMember, agentH.GetMonitoringStats)

			// MCP Server
			workspaces.GET("/:wsParam/mcp", wsMember, mcpH.List)
			workspaces.POST("/:wsParam/mcp", wsMember, mcpH.Create)
			workspaces.GET("/:wsParam/mcp/:id", wsMember, mcpH.Get)
			workspaces.PUT("/:wsParam/mcp/:id", wsMember, mcpH.Update)
			workspaces.DELETE("/:wsParam/mcp/:id", wsMember, mcpH.Delete)
			workspaces.POST("/:wsParam/mcp/:id/discover", wsMember, mcpH.DiscoverTools)
			workspaces.GET("/:wsParam/mcp/:id/tools", wsMember, mcpH.GetTools)
			workspaces.POST("/:wsParam/mcp/:id/execute", wsMember, mcpH.ExecuteTool)
			workspaces.POST("/:wsParam/mcp/:id/sync", wsMember, mcpH.SyncTools)

			// GitHub integration
			workspaces.GET("/:wsParam/github", githubH.List)
			workspaces.POST("/:wsParam/github", githubH.Create)
			workspaces.GET("/:wsParam/github/:id", githubH.Get)
			workspaces.PUT("/:wsParam/github/:id", githubH.Update)
			workspaces.DELETE("/:wsParam/github/:id", githubH.Delete)
			workspaces.POST("/:wsParam/github/:id/sync", githubH.SyncIssues)

			// Developer Agent (PRD P4-001)
			developerAgentSvc := service.NewDeveloperAgentService(db, githubSvc, llmClient)
			developerAgentH := handler.NewDeveloperAgentHandler(developerAgentSvc)
			workspaces.GET("/:wsParam/developer-agent/jobs", developerAgentH.List)
			workspaces.POST("/:wsParam/developer-agent/jobs", developerAgentH.Create)
			workspaces.GET("/:wsParam/developer-agent/jobs/:jobId", developerAgentH.Get)
			workspaces.POST("/:wsParam/developer-agent/jobs/:jobId/cancel", developerAgentH.Cancel)
			workspaces.DELETE("/:wsParam/developer-agent/jobs/:jobId", developerAgentH.Delete)

			// Tester Agent (PRD P4-002)
			testerAgentSvc := service.NewTesterAgentService(db, llmClient)
			testerAgentH := handler.NewTesterAgentHandler(testerAgentSvc)
			workspaces.GET("/:wsParam/tester-agent/jobs", testerAgentH.List)
			workspaces.POST("/:wsParam/tester-agent/jobs", testerAgentH.Create)
			workspaces.GET("/:wsParam/tester-agent/jobs/:jobId", testerAgentH.Get)
			workspaces.POST("/:wsParam/tester-agent/jobs/:jobId/cancel", testerAgentH.Cancel)
			workspaces.DELETE("/:wsParam/tester-agent/jobs/:jobId", testerAgentH.Delete)

			// CI/CD integration (PRD P4-003/004)
			cicdSvc := service.NewCICDService(db)
			cicdH := handler.NewCICDHandler(cicdSvc)
			workspaces.GET("/:wsParam/cicd/configs", cicdH.ListConfigs)
			workspaces.POST("/:wsParam/cicd/configs", cicdH.CreateConfig)
			workspaces.GET("/:wsParam/cicd/configs/:configId", cicdH.GetConfig)
			workspaces.PATCH("/:wsParam/cicd/configs/:configId", cicdH.UpdateConfig)
			workspaces.DELETE("/:wsParam/cicd/configs/:configId", cicdH.DeleteConfig)
			workspaces.GET("/:wsParam/cicd/builds", cicdH.ListBuilds)
			workspaces.POST("/:wsParam/cicd/builds", cicdH.TriggerBuild)
			workspaces.GET("/:wsParam/cicd/builds/:buildId", cicdH.GetBuild)
			workspaces.POST("/:wsParam/cicd/builds/:buildId/cancel", cicdH.CancelBuild)
			workspaces.DELETE("/:wsParam/cicd/builds/:buildId", cicdH.DeleteBuild)

			// SDLC orchestration (PRD P4-006)
			sdlcSvc := service.NewSDLCService(db)
			sdlcH := handler.NewSDLCHandler(sdlcSvc)
			workspaces.GET("/:wsParam/sdlc/workflows", sdlcH.ListWorkflows)
			workspaces.POST("/:wsParam/sdlc/workflows", sdlcH.CreateWorkflow)
			workspaces.GET("/:wsParam/sdlc/workflows/:workflowId", sdlcH.GetWorkflow)
			workspaces.POST("/:wsParam/sdlc/workflows/:workflowId/cancel", sdlcH.CancelWorkflow)
			workspaces.DELETE("/:wsParam/sdlc/workflows/:workflowId", sdlcH.DeleteWorkflow)
			workspaces.POST("/:wsParam/sdlc/workflows/:workflowId/retry", sdlcH.RetryWorkflow)
			workspaces.GET("/:wsParam/sdlc/workflows/:workflowId/stages", sdlcH.ListStages)
			workspaces.GET("/:wsParam/sdlc/workflows/:workflowId/stages/:stageId", sdlcH.GetStage)

			// Git integration
			workspaces.POST("/:wsParam/git-integration", gitIntegrationH.CreateIntegration)
			workspaces.GET("/:wsParam/git-integration", gitIntegrationH.GetIntegration)
			workspaces.PUT("/:wsParam/git-integration", gitIntegrationH.UpdateIntegration)
			workspaces.DELETE("/:wsParam/git-integration", gitIntegrationH.DeleteIntegration)
			workspaces.GET("/:wsParam/issues/:issueId/git-links", gitIntegrationH.GetIssueGitLinks)
			workspaces.POST("/:wsParam/issues/:issueId/git-links", gitIntegrationH.LinkPullRequest)

			// Slack integration
			workspaces.GET("/:wsParam/slack", slackH.List)
			workspaces.POST("/:wsParam/slack", slackH.Create)
			workspaces.GET("/:wsParam/slack/:id", slackH.Get)
			workspaces.PUT("/:wsParam/slack/:id", slackH.Update)
			workspaces.DELETE("/:wsParam/slack/:id", slackH.Delete)
			workspaces.POST("/:wsParam/slack/:id/notify", slackH.SendNotification)
			workspaces.POST("/:wsParam/slack/:id/test", slackH.TestNotification)

			// RBAC Roles & Permissions
			workspaces.GET("/:wsParam/roles", roleH.ListRoles)
			workspaces.POST("/:wsParam/roles", roleH.CreateRole)
			workspaces.PUT("/:wsParam/roles/:id", roleH.UpdateRole)
			workspaces.DELETE("/:wsParam/roles/:id", roleH.DeleteRole)

			// Field Permissions
			workspaces.GET("/:wsParam/field-permissions", fieldPermH.List)
			workspaces.POST("/:wsParam/field-permissions", fieldPermH.Create)
			workspaces.PUT("/:wsParam/field-permissions/:id", fieldPermH.Update)
			workspaces.DELETE("/:wsParam/field-permissions/:id", fieldPermH.Delete)
			workspaces.GET("/:wsParam/field-permissions/check", fieldPermH.CheckAccess)

			// Plugin management
			workspaces.GET("/:wsParam/plugins/catalog", pluginH.ListCatalog)
			workspaces.GET("/:wsParam/plugins", pluginH.ListInstalled)
			workspaces.POST("/:wsParam/plugins", pluginH.Install)
			workspaces.GET("/:wsParam/plugins/:id", pluginH.Get)
			workspaces.PUT("/:wsParam/plugins/:id", pluginH.Update)
			workspaces.DELETE("/:wsParam/plugins/:id", pluginH.Uninstall)
			workspaces.POST("/:wsParam/plugins/:id/enable", pluginH.Enable)
			workspaces.POST("/:wsParam/plugins/:id/disable", pluginH.Disable)
			workspaces.GET("/:wsParam/plugins/:id/logs", pluginH.GetEventLogs)
			workspaces.POST("/:wsParam/plugins/:id/test", pluginH.TestExecute)

			// Workspace-level Automation Rules
			workspaces.GET("/:wsParam/automations", automationH.ListWorkspace)
			workspaces.POST("/:wsParam/automations", automationH.CreateWorkspace)
			workspaces.GET("/:wsParam/automations/:id", automationH.Get)
			workspaces.PUT("/:wsParam/automations/:id", automationH.UpdateWorkspace)
			workspaces.DELETE("/:wsParam/automations/:id", automationH.DeleteWorkspace)

			// Workspace-level Workflows (state-machine)
			workspaces.GET("/:wsParam/workflows", wsWorkflow, workflowH.ListWorkspaceWorkflows)
			workspaces.POST("/:wsParam/workflows", wsWorkflow, workflowH.CreateWorkspaceWorkflow)
			workspaces.GET("/:wsParam/workflows/:workflowId", wsWorkflow, workflowH.GetStateWorkflow)
			workspaces.PUT("/:wsParam/workflows/:workflowId", wsWorkflow, workflowH.UpdateWorkspaceWorkflow)
			workspaces.DELETE("/:wsParam/workflows/:workflowId", wsWorkflow, workflowH.DeleteWorkspaceWorkflow)
			workspaces.GET("/:wsParam/workflows/:workflowId/transitions", wsWorkflow, workflowH.ListWorkspaceTransitions)
			workspaces.POST("/:wsParam/workflows/:workflowId/transitions", wsWorkflow, workflowH.AddWorkspaceTransition)
			workspaces.PUT("/:wsParam/workflows/:workflowId/transitions/:transitionId", wsWorkflow, workflowH.UpdateWorkspaceTransition)
			workspaces.DELETE("/:wsParam/workflows/:workflowId/transitions/:transitionId", wsWorkflow, workflowH.DeleteWorkspaceTransition)
			// Alias /edges → transitions for StateWorkflowManager compatibility
			workspaces.POST("/:wsParam/workflows/:workflowId/edges", wsWorkflow, workflowH.AddWorkspaceTransition)
			workspaces.DELETE("/:wsParam/workflows/:workflowId/edges/:transitionId", wsWorkflow, workflowH.DeleteWorkspaceTransition)

			// Workspace-level Approvals
			workspaces.GET("/:wsParam/approvals", approvalH.ListByWorkspace)
			workspaces.GET("/:wsParam/approvals/count", approvalH.CountPending)

			// Workspace-level Settings: States
			workspaces.GET("/:wsParam/settings/states", settingsH.ListWorkspaceStates)
			workspaces.POST("/:wsParam/settings/states", settingsManage, settingsH.CreateWorkspaceState)
			workspaces.GET("/:wsParam/settings/states/:stateId", settingsH.GetWorkspaceState)
			workspaces.PUT("/:wsParam/settings/states/:stateId", settingsManage, settingsH.UpdateWorkspaceState)
			workspaces.DELETE("/:wsParam/settings/states/:stateId", settingsManage, settingsH.DeleteWorkspaceState)

			// Workspace-level Modules
			workspaces.GET("/:wsParam/modules", moduleH.ListWorkspaceModules)
			workspaces.POST("/:wsParam/modules", moduleH.CreateWorkspaceModule)
			workspaces.GET("/:wsParam/modules/:moduleId", moduleH.GetWorkspaceModule)
			workspaces.PUT("/:wsParam/modules/:moduleId", moduleH.UpdateWorkspaceModule)
			workspaces.DELETE("/:wsParam/modules/:moduleId", moduleH.DeleteWorkspaceModule)

			// Memory (new)
			memoryH := handler.NewMemoryHandler(db, llmClient)
			workspaces.GET("/:wsParam/memories", memoryH.ListMemories)
			workspaces.POST("/:wsParam/memories", memoryH.CreateMemory)
			workspaces.GET("/:wsParam/memories/:memoryId", memoryH.GetMemory)
			workspaces.PUT("/:wsParam/memories/:memoryId", memoryH.UpdateMemory)
			workspaces.DELETE("/:wsParam/memories/:memoryId", memoryH.DeleteMemory)
			workspaces.POST("/:wsParam/memories/search", memoryH.SearchMemories)
			workspaces.POST("/:wsParam/memories/semantic-search", memoryH.SemanticSearch)
			workspaces.GET("/:wsParam/memories/context/:contextKey", memoryH.GetContextMemories)
			workspaces.POST("/:wsParam/memories/prune", memoryH.PruneMemories)
			workspaces.GET("/:wsParam/memories/stats", memoryH.GetMemoryStats)
			workspaces.POST("/:wsParam/memories/find-similar", memoryH.FindSimilarMemories)
			workspaces.POST("/:wsParam/memories/merge", memoryH.MergeMemories)

			// Memory Sessions
			workspaces.POST("/:wsParam/memory-sessions", memoryH.CreateMemorySession)
			workspaces.GET("/:wsParam/memory-sessions/:sessionId", memoryH.GetMemorySession)
			workspaces.PUT("/:wsParam/memory-sessions/:sessionId/close", memoryH.CloseMemorySession)
		}

		// Permissions (global, read-only)
		v1.GET("/permissions", authMiddleware, roleH.ListPermissions)

		// Initiatives (top-level routes)
		initiatives := v1.Group("/initiatives", authMiddleware)
		{
			initiatives.GET("/:initiativeId", initiativeH.Get)
			initiatives.PUT("/:initiativeId", initiativeH.Update)
			initiatives.DELETE("/:initiativeId", initiativeH.Delete)
			initiatives.GET("/:initiativeId/progress", initiativeH.GetProgress)
		}

		// ---- Projects (protected) ----
		projects := v1.Group("/projects", authMiddleware)
		{
			projects.POST("", projectH.Create) // ?workspace_id=
			projects.GET("", projectH.List)    // ?workspace_id=
			projects.GET("/:projectId", projectH.Get)
			projects.PATCH("/:projectId", projectH.Update)
			projects.DELETE("/:projectId", projectH.Delete)
			projects.POST("/:projectId/archive", projectH.Archive)
			projects.POST("/:projectId/restore", projectH.Restore)
			projects.GET("/:projectId/my-permissions", projectAccess, handler.MyProjectPermissions(db))
			projects.GET("/:projectId/approvals", approvalH.ListByProject)
			projects.GET("/:projectId/members", projectH.ListMembers) // ?only_active=
			projects.GET("/:projectId/member-candidates", middleware.RequirePermission(db, "member:manage_project", "project"), projectH.ListMemberCandidates)
			projects.POST("/:projectId/members", projectH.AddMember)             // ?user_id=&role=
			projects.PATCH("/:projectId/members/:userId", projectH.UpdateMember) // ?role=
			projects.DELETE("/:projectId/members/:userId", projectH.RemoveMember)
			projects.GET("/:projectId/statistics", projectH.GetStatistics)
			projects.POST("/:projectId/reports", reportH.Generate)
			projects.POST("/:projectId/reports/v2", reportH.GenerateV2)
			projects.GET("/:projectId/saved-reports", savedReportH.List)
			projects.POST("/:projectId/saved-reports", savedReportH.Create)
			projects.PATCH("/:projectId/saved-reports/:id", savedReportH.Update)
			projects.DELETE("/:projectId/saved-reports/:id", savedReportH.Delete)
			// Metrics
			projects.GET("/:projectId/metrics/templates", metricH.ListTemplates)
			projects.GET("/:projectId/metrics/charts", metricH.ListCharts)
			projects.GET("/:projectId/metrics/charts/:chartId", metricH.GetChart)
			projects.POST("/:projectId/metrics/charts", metricH.CreateChart)
			projects.PUT("/:projectId/metrics/charts/:chartId", metricH.UpdateChart)
			projects.DELETE("/:projectId/metrics/charts/:chartId", metricH.DeleteChart)
			projects.POST("/:projectId/metrics/charts/:chartId/render", metricH.RenderChart)
			projects.POST("/:projectId/metrics/preview", metricH.PreviewChart)
			projects.POST("/:projectId/metrics/charts/reorder", metricH.ReorderCharts)
			projects.GET("/:projectId/metrics/filter-values", metricH.GetFilterValues)
			projects.GET("/:projectId/metrics/custom-fields", metricH.GetCustomFields)
			projects.POST("/:projectId/custom-fields/:fieldId/enroll", middleware.RequirePermission(db, "settings:manage", "project"), customFieldH.EnrollField)
			projects.POST("/:projectId/custom-fields/:fieldId/unenroll", middleware.RequirePermission(db, "settings:manage", "project"), customFieldH.UnenrollField)
			projects.GET("/:projectId/intake", intakeH.List)
			projects.GET("/:projectId/intake/metrics", intakeH.Metrics)
			projects.GET("/:projectId/intake/settings", intakeH.GetSettings)
			projects.PUT("/:projectId/intake/settings", intakeH.UpdateSettings)
			projects.GET("/:projectId/webhooks", webhookH.List)
			projects.POST("/:projectId/webhooks", webhookH.Create)
			projects.PUT("/:projectId/webhooks/:id", webhookH.Update)
			projects.DELETE("/:projectId/webhooks/:id", webhookH.Delete)
			projects.POST("/:projectId/intake/:issueId/triage", intakeH.Triage)
			projects.GET("/:projectId/intake/:issueId/spec", intakeH.GetSpec)
			projects.POST("/:projectId/intake/:issueId/spec", intakeH.DraftSpec)
			projects.POST("/:projectId/intake/:issueId/ai-analyze", projectAccess, aiH.TriageAnalyze)
			projects.GET("/:projectId/issues-summary", projectH.GetIssuesSummary)
			projects.PATCH("/:projectId/lead", projectH.UpdateProjectLead)
			projects.GET("/:projectId/subscribers", projectH.ListSubscribers)
			projects.POST("/:projectId/subscribers", projectH.AddSubscriber)
			projects.DELETE("/:projectId/subscribers/:userId", projectH.RemoveSubscriber)
			// Project Updates
			projectUpdateH := handler.NewProjectUpdateHandler(db)
			projects.GET("/:projectId/updates", projectUpdateH.List)
			projects.POST("/:projectId/updates", projectUpdateH.Create)
			projects.GET("/:projectId/cycles", cycleH.List)
			projects.GET("/:projectId/cycles/search", cycleH.Search)
			projects.POST("/:projectId/cycles", cycleH.Create) // ?workspace_id=

			// ---- Pages ----
			pages := projects.Group("/:projectId/pages", authMiddleware)
			{
				pages.GET("", pageH.List)
				pages.GET("/search", pageH.Search)          // ?q=
				pages.GET("/by-source", pageH.ListBySource) // ?source_type=&source_id=
				pages.POST("", pageH.Create)
				pages.GET("/tree", pageH.GetTree)
				pages.GET("/:pageId", pageH.Get)
				pages.PUT("/:pageId", pageH.Update)
				pages.DELETE("/:pageId", pageH.Delete)
				pages.POST("/:pageId/archive", pageH.Archive)
				pages.POST("/:pageId/restore", pageH.Restore)
				pages.POST("/:pageId/move", pageH.Move)
				pages.GET("/:pageId/children", pageH.ListChildren)
				pages.POST("/:pageId/lock", pageH.Lock)
				pages.POST("/:pageId/unlock", pageH.Unlock)
				pages.GET("/:pageId/export", pageH.Export)
				pages.POST("/:pageId/convert-to-issue", pageH.ConvertToIssue)
				pages.GET("/:pageId/versions", pageVersionH.List)
				pages.GET("/:pageId/versions/:versionId", pageVersionH.Get)
				pages.POST("/:pageId/versions/:versionId/restore", pageVersionH.Restore)
			}

			// ---- Page Templates ----
			pageTemplates := projects.Group("/:projectId/page-templates", authMiddleware)
			{
				pageTemplates.GET("", pageTemplateH.List)
				pageTemplates.POST("", pageTemplateH.Create)
				pageTemplates.GET("/:templateId", pageTemplateH.Get)
				pageTemplates.PUT("/:templateId", pageTemplateH.Update)
				pageTemplates.DELETE("/:templateId", pageTemplateH.Delete)
			}

			// ---- Project Time Tracking Summary ----
			projects.GET("/:projectId/time-tracks/summary", timeTrackH.ProjectSummary)

			// ---- Project Issue Types ----
			projTypes := projects.Group("/:projectId/issue-types", authMiddleware)
			{
				projTypes.GET("", projectIssueTypeH.ListProjectTypes)
				projTypes.POST("", middleware.RequirePermission(db, "settings:manage", "project"), projectIssueTypeH.CreateProjectType)
				projTypes.PATCH("/reorder", middleware.RequirePermission(db, "settings:manage", "project"), projectIssueTypeH.Reorder)

				// workspace-type Import model (project references workspace type by link)
				projTypes.GET("/importable", projectIssueTypeH.ListImportable)
				projTypes.POST("/:typeId/import", middleware.RequirePermission(db, "settings:manage", "project"), projectIssueTypeH.ImportType)
				projTypes.DELETE("/:typeId/import", middleware.RequirePermission(db, "settings:manage", "project"), projectIssueTypeH.UnimportType)
			}

			// ---- Saved Views ----
			views := projects.Group("/:projectId/views", authMiddleware)
			{
				views.GET("", savedViewH.List)
				views.POST("", savedViewH.Create)
				views.GET("/:viewId", savedViewH.Get)
				views.PUT("/:viewId", savedViewH.Update)
				views.DELETE("/:viewId", savedViewH.Delete)
				views.POST("/:viewId/set-default", savedViewH.SetDefault)
				views.POST("/:viewId/duplicate", savedViewH.Duplicate)
			}

			// ---- Dashboards ----
			dashboards := projects.Group("/:projectId/dashboards", authMiddleware)
			{
				dashboards.GET("", dashboardH.List)
				dashboards.POST("", dashboardH.Create)
				dashboards.GET("/:id", dashboardH.Get)
				dashboards.PUT("/:id", dashboardH.Update)
				dashboards.DELETE("/:id", dashboardH.Delete)
				dashboards.POST("/:id/set-default", dashboardH.SetDefault)
				dashboards.POST("/:id/duplicate", dashboardH.Duplicate)
				dashboards.GET("/:id/full", dashboardH.GetFull)

				// Widget CRUD
				dashboards.POST("/:id/widgets", dashboardH.AddWidget)
				dashboards.PUT("/:id/widgets/:widgetId", dashboardH.UpdateWidget)
				dashboards.DELETE("/:id/widgets/:widgetId", dashboardH.DeleteWidget)
				dashboards.PUT("/:id/widgets/reorder", dashboardH.ReorderWidgets)
			}

			// ---- Search Templates ----
			searchTemplates := projects.Group("/:projectId/search-templates", authMiddleware)
			{
				searchTemplates.GET("", searchTemplateH.List)
				searchTemplates.POST("", searchTemplateH.Create)
				searchTemplates.GET("/:templateId", searchTemplateH.Get)
				searchTemplates.DELETE("/:templateId", searchTemplateH.Delete)
				searchTemplates.POST("/:templateId/apply", searchTemplateH.Apply)
			}

			// ---- Work Item Templates ----
			templates := projects.Group("/:projectId/work-item-templates", authMiddleware)
			{
				templates.GET("", witH.List)
				templates.POST("", witH.Create)
				templates.GET("/:id", witH.Get)
				templates.PUT("/:id", witH.Update)
				templates.DELETE("/:id", witH.Delete)
			}

			// ---- Releases ----
			releases := projects.Group("/:projectId/releases", authMiddleware)
			{
				releases.GET("", releaseH.List)
				releases.GET("/search", releaseH.Search)
				releases.POST("", releaseH.Create)
				releases.GET("/:releaseId", releaseH.Get)
				releases.PUT("/:releaseId", releaseH.Update)
				releases.DELETE("/:releaseId", releaseH.Delete)
				releases.POST("/:releaseId/issues", releaseH.AddIssues)
				releases.DELETE("/:releaseId/issues", releaseH.RemoveIssues)
				releases.GET("/:releaseId/progress", releaseH.GetProgress)
			}

			// ---- Milestones (alias for releases) ----
			projects.GET("/:projectId/milestones", authMiddleware, releaseH.ListMilestones)

			// ---- Estimates ----
			estimates := projects.Group("/:projectId/estimate-points", authMiddleware)
			{
				estimates.GET("/settings", estimateH.GetSettings)
				estimates.PUT("/settings", estimateH.UpdateSettings)
				estimates.GET("", estimateH.ListPoints)
				estimates.POST("", estimateH.CreatePoint)
				estimates.GET("/:pointId", estimateH.GetPoint)
				estimates.PATCH("/:pointId", estimateH.UpdatePoint)
				estimates.DELETE("/:pointId", estimateH.DeletePoint)
				estimates.POST("/reorder", estimateH.ReorderPoints)
				estimates.POST("/bulk", estimateH.BulkCreatePoints)
				estimates.POST("/defaults", estimateH.CreateDefaultPoints)
			}

			estimateCategories := projects.Group("/:projectId/estimate-categories", authMiddleware)
			{
				estimateCategories.GET("", estimateH.ListCategories)
				estimateCategories.POST("", estimateH.CreateCategory)
				estimateCategories.POST("/defaults", estimateH.CreateDefaultCategories)
			}

			estimateTime := projects.Group("/:projectId/estimate-time", authMiddleware)
			{
				estimateTime.GET("", estimateH.ListTime)
				estimateTime.POST("", estimateH.CreateTime)
				estimateTime.POST("/defaults", estimateH.CreateDefaultTime)
			}

			// ---- Project Settings: States + Labels ----
			settings := projects.Group("/:projectId/settings", authMiddleware)
			{
				settingsManageProject := middleware.RequirePermission(db, "settings:manage", "project")

				// States
				settings.POST("/states", settingsManageProject, settingsH.CreateState)                 // ?workspace_id=
				settings.GET("/states", settingsH.ListStates)                                          // ?include_inactive=
				settings.POST("/states/default", settingsManageProject, settingsH.CreateDefaultStates) // ?workspace_id=
				settings.GET("/states/:stateId", settingsH.GetState)
				settings.PUT("/states/:stateId", settingsManageProject, settingsH.UpdateState)
				settings.DELETE("/states/:stateId", settingsManageProject, settingsH.DeleteState)

				// Labels
				settings.POST("/labels", settingsManageProject, settingsH.CreateLabel) // ?workspace_id=
				settings.GET("/labels", settingsH.ListLabels)
				settings.GET("/labels/search", settingsH.SearchLabels) // ?q=
				settings.GET("/labels/:labelId", settingsH.GetLabel)
				settings.PUT("/labels/:labelId", settingsManageProject, settingsH.UpdateLabel)
				settings.DELETE("/labels/:labelId", settingsManageProject, settingsH.DeleteLabel)
			}

			// ---- Project Page Tabs ----
			pageTabs := projects.Group("/:projectId/page-tabs", authMiddleware)
			{
				pageTabs.GET("", pageTabH.List)
				pageTabs.POST("", pageTabH.Create)
				pageTabs.PUT("/batch", pageTabH.BatchSave)
				pageTabs.PUT("/reorder", pageTabH.Reorder)
				pageTabs.PUT("/:tabId", pageTabH.Update)
				pageTabs.DELETE("/:tabId", pageTabH.Delete)
			}
		}

		// ---- Test route ----
		v1.GET("/test-issues", authMiddleware, issueH.List)

		// ---- Issues (protected) ----
		issues := v1.Group("/issues", authMiddleware)
		{
			// CRUD
			issues.POST("", middleware.RequirePermission(db, "issue:create", "project"), issueH.Create)               // ?project_id=&workspace_id=
			issues.GET("", issueH.List)                                                                               // ?project_id=&filters...
			issues.GET("/tree", issueH.Tree)                                                                          // ?project_id=&search=&limit=&offset=
			issues.GET("/statistics", issueH.GetStatistics)                                                           // ?project_id=
			issues.GET("/flow-metrics", issueH.GetFlowMetrics)                                                        // ?project_id=
			issues.GET("/search", issueH.Search)                                                                      // ?workspace_id=&query=
			issues.GET("/suggest", issueH.Suggest)                                                                    // ?project_id=&query=&limit=
			issues.POST("/bulk/update", middleware.RequirePermission(db, "issue:edit", "project"), issueH.BulkUpdate) // ?project_id=
			issues.POST("/bulk/delete", middleware.RequirePermission(db, "issue:delete", "project"), issueH.BulkDelete)
			issues.POST("/bulk/copy", middleware.RequirePermission(db, "issue:edit", "project"), issueH.BulkCopy)
			issues.POST("/bulk/convert-type", middleware.RequirePermission(db, "issue:edit", "project"), issueH.BulkConvertType)
			issues.POST("/bulk/move", middleware.RequirePermission(db, "issue:edit", "project"), issueH.BulkMove)
			issues.POST("/merge", issueH.MergeDuplicates)

			// Import
			issues.POST("/import/json", issueH.ImportJSON) // ?project_id=&workspace_id=
			issues.POST("/import/csv", issueH.ImportCSV)   // ?project_id=&workspace_id=
			issues.GET("/import/template", issueH.ExportCSVTemplate)

			// Export
			issues.GET("/export", issueH.Export) // ?project_id=&format=csv|json

			// Single issue
			issues.GET("/:issueId", issueH.Get)
			issues.PUT("/:issueId", middleware.RequirePermission(db, "issue:edit", "project"), issueH.Update)
			issues.DELETE("/:issueId", middleware.RequirePermission(db, "issue:delete", "project"), issueH.Delete)
			issues.POST("/:issueId/archive", middleware.RequirePermission(db, "issue:edit", "project"), issueH.Archive)
			issues.GET("/:issueId/children", issueH.Children) // tree lazy-load children
			issues.POST("/:issueId/reorder-sub-issues", middleware.RequirePermission(db, "issue:edit", "project"), issueH.ReorderSubIssues)
			issues.POST("/:issueId/watch", issueH.AddWatcher)
			issues.DELETE("/:issueId/watch", issueH.RemoveWatcher)
			issues.GET("/:issueId/watchers", issueH.ListWatchers)
			issues.POST("/:issueId/restore", middleware.RequirePermission(db, "issue:edit", "project"), issueH.Restore)
			issues.POST("/:issueId/convert-type", middleware.RequirePermission(db, "issue:edit", "project"), issueH.ConvertType)
			issues.GET("/:issueId/activities", issueH.GetActivities) // ?limit=&offset=
			// Chat (lazy get/create for an issue)
			issues.GET("/:issueId/chat", chatH.GetOrCreateForIssue)

			// Assignees
			issues.POST("/:issueId/assignees", middleware.RequirePermission(db, "issue:edit", "project"), issueH.AddAssignee) // ?user_id=
			issues.DELETE("/:issueId/assignees/:userId", middleware.RequirePermission(db, "issue:edit", "project"), issueH.RemoveAssignee)
			// Adopt the structured field proposals an agent attached to its comment.
			issues.POST("/:issueId/suggestions/apply", middleware.RequirePermission(db, "issue:edit", "project"), issueH.ApplySuggestions)
			issues.POST("/:issueId/suggestions/dismiss", middleware.RequirePermission(db, "issue:edit", "project"), issueH.DismissSuggestions)

			// Labels
			issues.POST("/:issueId/labels", middleware.RequirePermission(db, "issue:edit", "project"), issueH.AddLabel) // ?label_id=
			issues.DELETE("/:issueId/labels/:labelId", middleware.RequirePermission(db, "issue:edit", "project"), issueH.RemoveLabel)

			// Cycle
			issues.POST("/:issueId/cycle", middleware.RequirePermission(db, "issue:edit", "project"), issueH.SetCycle) // ?cycle_id=
			issues.DELETE("/:issueId/cycle", middleware.RequirePermission(db, "issue:edit", "project"), issueH.RemoveCycle)

			// Time Tracks
			issues.POST("/:issueId/time-tracks/start", timeTrackH.Start)
			issues.POST("/:issueId/time-tracks/stop", timeTrackH.Stop)
			issues.GET("/:issueId/time-tracks", timeTrackH.List)
			issues.GET("/:issueId/time-tracks/summary", timeTrackH.Summary)
			issues.DELETE("/:issueId/time-tracks/:id", timeTrackH.Delete)

			// Recurrence
			issues.POST("/:issueId/recurrence", recurrenceH.Create)
			issues.GET("/:issueId/recurrence", recurrenceH.Get)
			issues.POST("/:issueId/ai/comment", issueAccess, aiH.AssistComment)
			issues.POST("/:issueId/agents/:agentId/mention", issueAccess, agentH.HandleMention)
			issues.PUT("/:issueId/recurrence", recurrenceH.Update)
			issues.DELETE("/:issueId/recurrence", recurrenceH.Delete)

			// Pages
			issues.GET("/:issueId/pages", issueH.ListPages)
			issues.POST("/:issueId/pages", middleware.RequirePermission(db, "issue:edit", "project"), issueH.AddPage)      // ?page_id=
			issues.DELETE("/:issueId/pages", middleware.RequirePermission(db, "issue:edit", "project"), issueH.RemovePage) // ?page_id=

			// Attachments
			issues.GET("/:issueId/attachments", attachmentH.ListByIssue)
			issues.POST("/:issueId/attachments", middleware.RequirePermission(db, "issue:edit", "project"), attachmentH.Create)
			issues.GET("/:issueId/attachments/:attachmentId", attachmentH.Get)
			issues.GET("/:issueId/attachments/:attachmentId/download", attachmentH.Download)
			issues.DELETE("/:issueId/attachments/:attachmentId", middleware.RequirePermission(db, "issue:edit", "project"), attachmentH.Delete)

			// Approvals
			issues.POST("/:issueId/approvals", approvalH.Create)
		}

		// ---- Approval routes (top-level) ----
		approval := v1.Group("/approvals", authMiddleware)
		{
			approval.GET("/:id", approvalH.Get)
			approval.POST("/:id/decide", approvalH.Decide)
			approval.POST("/:id/cancel", approvalH.Cancel)
		}

		// ---- Modules (protected) ----
		modules := v1.Group("/modules", authMiddleware)
		{
			// CRUD
			modules.POST("", moduleH.Create)       // ?workspace_id=
			modules.GET("", moduleH.List)          // ?project_id=&workspace_id=&include_archived=
			modules.GET("/search", moduleH.Search) // ?project_id=&workspace_id=&q=
			modules.GET("/:moduleId", moduleH.Get)
			modules.PUT("/:moduleId", moduleH.Update)
			modules.DELETE("/:moduleId", moduleH.Delete)

			// Issues
			modules.POST("/:moduleId/issues", moduleH.AddIssue) // ?issue_id=
			modules.DELETE("/:moduleId/issues/:issueId", moduleH.RemoveIssue)
			modules.GET("/:moduleId/issues", moduleH.ListIssues)

			// Analysis
			modules.GET("/:moduleId/progress", moduleH.GetProgress)
			modules.GET("/:moduleId/statistics", moduleH.GetStatistics)

			// Tree
			modules.GET("/tree", moduleH.GetTree) // ?project_id=&workspace_id=
		}

		// ---- Module Inheritance Overrides (protected) ----
		projects.POST("/:projectId/modules/:moduleId/override", moduleH.CreateOrUpdateOverride)
		projects.DELETE("/:projectId/modules/:moduleId/override", moduleH.DeleteOverride)
		// ---- Cycles (protected) ----
		cycles := v1.Group("/cycles", authMiddleware)
		{
			cycles.GET("/:cycleId", cycleH.Get)
			cycles.PUT("/:cycleId", cycleH.Update)
			cycles.DELETE("/:cycleId", cycleH.Delete)
			cycles.POST("/:cycleId/start", cycleH.Start)
			cycles.POST("/:cycleId/end", cycleH.End)
			cycles.POST("/:cycleId/cancel", cycleH.Cancel)
			cycles.POST("/:cycleId/issues", cycleH.AddIssue)
			cycles.DELETE("/:cycleId/issues/:issueId", cycleH.RemoveIssue)
			cycles.GET("/:cycleId/issues", cycleH.ListIssues)
			cycles.GET("/:cycleId/progress", cycleH.GetProgress)
			cycles.GET("/:cycleId/statistics", cycleH.GetStatistics)
			cycles.GET("/:cycleId/burndown", cycleH.GetBurndown)
			// Automation
			cycles.POST("/:cycleId/apply-auto-add", cycleH.ApplyAutoAddRules)
			cycles.POST("/:cycleId/apply-auto-close", cycleH.ApplyAutoCloseRules)
		}

		// ---- Issue Types (protected) ----
		issueTypes := v1.Group("/issue-types", authMiddleware)
		{
			issueTypes.POST("", settingsManage, issueTypeH.Create)
			issueTypes.GET("", issueTypeH.List)
			issueTypes.GET("/:typeId", issueTypeH.Get)
			issueTypes.PUT("/:typeId", settingsManage, issueTypeH.Update)
			issueTypes.DELETE("/:typeId", settingsManage, issueTypeH.Delete)
			issueTypes.PATCH("/:typeId/disable", settingsManage, issueTypeH.Disable)
			issueTypes.PATCH("/reorder-workspace", settingsManage, issueTypeH.ReorderWorkspace) // ?workspace_id=
			issueTypes.GET("/:typeId/fields", issueTypeH.ListFields)
			issueTypes.POST("/:typeId/fields", settingsManage, issueTypeH.AddField)
			issueTypes.PUT("/:typeId/fields/:fieldId", settingsManage, issueTypeH.UpdateField)
			issueTypes.DELETE("/:typeId/fields/:fieldId", settingsManage, issueTypeH.RemoveField)
		}

		// ---- Custom Fields (protected) ----
		customFields := v1.Group("/custom-fields", authMiddleware)
		{
			customFields.POST("", settingsManage, customFieldH.Create)
			customFields.GET("", customFieldH.List)
			customFields.GET("/:fieldId", customFieldH.Get)
			customFields.PUT("/:fieldId", settingsManage, customFieldH.Update)
			customFields.DELETE("/:fieldId", settingsManage, customFieldH.Delete)
			customFields.POST("/:fieldId/options", settingsManage, customFieldH.CreateOption)
			customFields.PUT("/:fieldId/options/:optionId", settingsManage, customFieldH.UpdateOption)
			customFields.DELETE("/:fieldId/options/:optionId", settingsManage, customFieldH.DeleteOption)
			customFields.POST("/issues/:issueId/values", middleware.RequirePermission(db, "issue:edit", "project"), customFieldH.SetIssueValue)
			customFields.GET("/issues/:issueId/values", customFieldH.ListIssueValues)
			customFields.POST("/issues/:issueId/values/bulk", middleware.RequirePermission(db, "issue:edit", "project"), customFieldH.BulkSetIssueValues)
			customFields.PUT("/issues/:issueId/values/:fieldId", middleware.RequirePermission(db, "issue:edit", "project"), customFieldH.UpdateIssueValue)
			customFields.DELETE("/issues/:issueId/values/:fieldId", middleware.RequirePermission(db, "issue:edit", "project"), customFieldH.DeleteIssueValue)
			customFields.GET("/issues/:issueId/fields", customFieldH.GetIssueFieldsWithValues)
			customFields.GET("/workspace-fields", customFieldH.ListWorkspaceFieldsWithEnrollment)
		}

		// ---- Conditional Fields (protected) ----
		conditionalFields := v1.Group("/conditional-fields", authMiddleware)
		{
			conditionalFields.POST("", settingsManage, conditionalFieldH.Create)           // ?workspace_id=
			conditionalFields.GET("", conditionalFieldH.List)                              // ?workspace_id=&field_id=
			conditionalFields.GET("/:id", conditionalFieldH.Get)                           // ?workspace_id=
			conditionalFields.PUT("/:id", settingsManage, conditionalFieldH.Update)        // ?workspace_id=
			conditionalFields.DELETE("/:id", settingsManage, conditionalFieldH.Delete)     // ?workspace_id=
			conditionalFields.POST("/evaluate", conditionalFieldH.EvaluateFieldVisibility) // ?workspace_id=
		}

		// ---- Project Templates (protected) ----
		templates := v1.Group("/templates", authMiddleware)
		{
			templates.POST("", settingsManage, templateH.Create)
			templates.GET("", templateH.List)
			templates.GET("/:templateId", templateH.Get)
			templates.PUT("/:templateId", settingsManage, templateH.Update)
			templates.DELETE("/:templateId", settingsManage, templateH.Delete)
			templates.POST("/:templateId/types", settingsManage, templateH.AddType)
			templates.DELETE("/:templateId/types/:typeId", settingsManage, templateH.RemoveType)
			templates.POST("/:templateId/apply", settingsManage, templateH.Apply)
		}
		// ---- Type Templates (Workspace-level) ----
		typeTemplates := v1.Group("/type-templates", authMiddleware)
		{
			typeTemplates.POST("", settingsManage, typeTemplateH.Create)
			typeTemplates.GET("", typeTemplateH.List)
			typeTemplates.GET("/:id", typeTemplateH.Get)
			typeTemplates.PUT("/:id", settingsManage, typeTemplateH.Update)
			typeTemplates.DELETE("/:id", settingsManage, typeTemplateH.Delete)
			typeTemplates.POST("/:id/fields", settingsManage, typeTemplateH.BindField)
			typeTemplates.DELETE("/:id/fields/:fieldId", settingsManage, typeTemplateH.UnbindField)
		}
		// ---- Relations (Workspace-level) ----
		relations := v1.Group("/relations", authMiddleware)
		{
			relations.POST("/types", settingsManage, relationH.CreateType)
			relations.GET("/types", relationH.ListTypes)
			relations.PUT("/types/:id", settingsManage, relationH.UpdateType)
			relations.DELETE("/types/:id", settingsManage, relationH.DeleteType)
		}
		// ---- Issue Relations ----
		issues.POST("/:issueId/relations", relationH.CreateRelation)
		issues.GET("/:issueId/relations", relationH.ListRelations)
		v1.DELETE("/relations/:relationId", authMiddleware, relationH.DeleteRelation)
		// ---- Workflows (protected with RBAC) ----
		workflows := v1.Group("/projects/:projectId/workflows", authMiddleware, middleware.RequireProjectWorkflow(db))
		{
			workflows.GET("", workflowH.ListWorkflows)
			workflows.GET("/:workflowId", workflowH.GetWorkflow)
			workflows.POST("", middleware.RequirePermission(db, "workflow:manage", "project"), workflowH.CreateWorkflow)
			workflows.PUT("/:workflowId", middleware.RequirePermission(db, "workflow:manage", "project"), workflowH.UpdateWorkflow)
			workflows.DELETE("/:workflowId", middleware.RequirePermission(db, "workflow:manage", "project"), workflowH.DeleteWorkflow)
			// Transitions belong to state-machine workflows, served under
			// /workspaces/:wsParam/workflows. This group is agent orchestration:
			// its structure is nodes and edges, so there is no transition CRUD.
			// --- Agent-Project Integration: Workflow nodes, edges, execute, runs ---
			workflows.POST("/:workflowId/nodes", middleware.RequirePermission(db, "workflow:manage", "project"), workflowH.AddNode)
			workflows.PUT("/:workflowId/nodes/:nodeId", middleware.RequirePermission(db, "workflow:manage", "project"), workflowH.UpdateNode)
			workflows.DELETE("/:workflowId/nodes/:nodeId", middleware.RequirePermission(db, "workflow:manage", "project"), workflowH.DeleteNode)
			workflows.POST("/:workflowId/edges", middleware.RequirePermission(db, "workflow:manage", "project"), workflowH.AddEdge)
			workflows.PUT("/:workflowId/edges/:edgeId", middleware.RequirePermission(db, "workflow:manage", "project"), workflowH.UpdateEdge)
			workflows.DELETE("/:workflowId/edges/:edgeId", middleware.RequirePermission(db, "workflow:manage", "project"), workflowH.DeleteEdge)
			workflows.POST("/:workflowId/execute", workflowH.ExecuteWorkflow)
			workflows.GET("/:workflowId/runs", workflowH.ListRuns)
			workflows.GET("/:workflowId/runs/:runId", workflowH.GetRun)
			workflows.POST("/:workflowId/runs/:runId/cancel", workflowH.CancelRun)
		}
		// ---- Agent Members (project-level) ----
		agentMembers := v1.Group("/projects/:projectId/agent-members", authMiddleware, projectAccess)
		{
			agentMembers.GET("", agentMemberH.ListByProject)
			agentMembers.POST("", middleware.RequirePermission(db, "workflow:manage", "project"), agentMemberH.Add)
			agentMembers.PUT("/:memberId/role", middleware.RequirePermission(db, "workflow:manage", "project"), agentMemberH.UpdateRole)
			agentMembers.DELETE("/:memberId", middleware.RequirePermission(db, "workflow:manage", "project"), agentMemberH.Remove)
		}
		// ---- Issue-Agent assignments ----
		issues.POST("/:issueId/assign-agent", issueAccess, issueAgentH.AssignAgent)
		issues.DELETE("/:issueId/assign-agent", issueAccess, issueAgentH.UnassignAgent)
		issues.GET("/:issueId/agent-status", issueAccess, issueAgentH.GetAgentStatus)
		issues.POST("/:issueId/preview-execution", issueAccess, issueAgentH.PreviewExecution)
		issues.POST("/bulk-assign-agent", issueAgentH.BulkAssign)
		// ---- Budget & SLA (project-level) ----
		v1.GET("/projects/:projectId/budget", authMiddleware, projectAccess, budgetH.Get)
		v1.PUT("/projects/:projectId/budget", authMiddleware, projectAccess, middleware.RequirePermission(db, "workflow:manage", "project"), budgetH.Update)
		v1.GET("/projects/:projectId/sla", authMiddleware, projectAccess, slaH.Get)
		v1.PUT("/projects/:projectId/sla", authMiddleware, projectAccess, middleware.RequirePermission(db, "workflow:manage", "project"), slaH.Update)
		v1.GET("/projects/:projectId/decisions", authMiddleware, projectAccess, decisionH.ListByProject)
		// ---- Issue duplicate check (pre-create warn-only) ----
		v1.POST("/projects/:projectId/issues/duplicate-check", authMiddleware, projectAccess, issueH.CheckDuplicates)
		v1.GET("/projects/:projectId/issues/duplicate-check", authMiddleware, projectAccess, issueH.CheckDuplicates)
		// ---- Decision Records ----
		issues.GET("/:issueId/decisions", issueAccess, decisionH.ListByIssue)
		v1.GET("/agent-tasks/:taskId/decisions", authMiddleware, decisionH.ListByTask)
		// ---- Comments (protected) ----
		comments := v1.Group("/comments", authMiddleware)
		{
			comments.POST("", commentH.Create)
			comments.GET("/issue/:issueId", commentH.ListByIssue)
			comments.GET("/:commentId", commentH.Get)
			comments.PATCH("/:commentId", commentH.Update)
			comments.DELETE("/:commentId", commentH.Delete)
			comments.POST("/:commentId/resolve", commentH.Resolve)
			comments.POST("/:commentId/unresolve", commentH.Unresolve)
		}
		// ---- Chat & Messages ----
		// SSE stream is registered without authMiddleware because browser EventSource
		// cannot set the Authorization header; the handler performs its own JWT auth
		// via ?token= plus a membership check (defense in depth).
		v1.GET("/chats/:chatId/stream", chatH.Stream)
		chats := v1.Group("/chats", authMiddleware)
		{
			chats.GET("/:chatId", chatH.GetChat)
			chats.GET("/:chatId/messages", chatH.ListMessages)
			chats.POST("/:chatId/messages", chatH.SendMessage)
		}
		// Message-scoped routes (flat for stable URLs regardless of chat)
		messages := v1.Group("/messages", authMiddleware)
		{
			messages.PUT("/:messageId", chatH.EditMessage)
			messages.DELETE("/:messageId", chatH.DeleteMessage)
			messages.POST("/:messageId/reactions", chatH.AddReaction)
			messages.DELETE("/:messageId/reactions", chatH.RemoveReaction)
		}
		// ---- RQL (protected) ----
		v1.POST("/pages/:pageId/ai", authMiddleware, pageAccess, aiH.PageAI)

		rqlHandler := rql.NewRQLHandler(db)
		rqlGroup := v1.Group("/rql", authMiddleware)
		{
			rqlGroup.POST("/search", rqlHandler.Search)
		}
		// ---- AI ----
		projects.POST("/:projectId/ai/chat", projectAccess, aiH.Chat)
		projects.POST("/:projectId/ai/search", projectAccess, aiH.Search)
		projects.POST("/:projectId/ai/analyze", projectAccess, aiH.Analyze)
		projects.POST("/:projectId/ai/create", projectAccess, aiH.CreatePreview)
		projects.POST("/:projectId/ai/chart", projectAccess, aiH.Chart)
		projects.POST("/:projectId/ai/sprint-plan", projectAccess, aiH.SprintPlan)
		projects.POST("/:projectId/ai/suggest-labels", projectAccess, aiH.SuggestLabels)
		projects.POST("/:projectId/ai/automation-preview", projectAccess, aiH.AutomationPreview)

		// Agent routes (project-level)
		projects.POST("/:projectId/agent/auto-triage", projectAccess, agentH.AutoTriageProject)
		projects.POST("/:projectId/agent/auto-assign", projectAccess, agentH.AutoAssignProject)

		// Automation rules
		projects.GET("/:projectId/automations", automationH.List)
		projects.POST("/:projectId/automations", automationH.Create)
		projects.GET("/:projectId/automations/:id", automationH.Get)
		projects.PUT("/:projectId/automations/:id", automationH.Update)
		projects.DELETE("/:projectId/automations/:id", automationH.Delete)
		projects.POST("/:projectId/automations/:id/execute", automationH.Execute)

		// Execution history
		v1.GET("/issues/:issueId/automation-history", authMiddleware, automationH.GetExecutionHistory)
		v1.GET("/automations/:ruleId/execution-history", authMiddleware, automationH.GetRuleExecutionHistory)
		v1.GET("/projects/:projectId/automation-executions", authMiddleware, automationH.GetProjectExecutionHistory)

		// ---- Notifications (protected) ----
		notifications := v1.Group("/notifications", authMiddleware)
		{
			notifications.GET("", notificationH.List)
			notifications.GET("/summary", notificationH.GetSummary)
			notifications.GET("/:id", notificationH.Get)
			notifications.POST("", notificationH.Create)
			notifications.POST("/bulk", notificationH.CreateBulk)
			notifications.POST("/check-due-reminders", notificationH.CheckDueReminders)
			notifications.PATCH("/:id/read", notificationH.MarkRead)
			notifications.POST("/read-all", notificationH.MarkAllRead)
			notifications.DELETE("/:id", notificationH.Delete)
		}
	}
}

// Shutdown performs graceful shutdown of all background services
func Shutdown() {
	schedulerMu.Lock()
	if memScheduler != nil {
		memScheduler.Stop()
		memScheduler = nil
	}
	schedulerMu.Unlock()
}

// harnessSkillExecutorAdapter adapts harness.SkillExecutor to service.SkillExecutor interface.
type harnessSkillExecutorAdapter struct {
	executor *harness.SkillExecutor
}

func (a *harnessSkillExecutorAdapter) Execute(ctx context.Context, skill *model.Skill, params map[string]interface{}) (*service.SkillExecutionResult, error) {
	result, err := a.executor.Execute(ctx, skill, params)
	if err != nil {
		return nil, err
	}

	steps := make([]service.SkillStep, 0, len(result.Steps))
	for _, step := range result.Steps {
		steps = append(steps, service.SkillStep{
			Step:   step.Step,
			Action: step.Action,
			Tool:   step.Tool,
			Input:  step.Input,
			Output: step.Output,
			Error:  step.Error,
			Status: step.Status,
		})
	}

	return &service.SkillExecutionResult{
		SkillID:     result.SkillID,
		SkillName:   result.SkillName,
		Steps:       steps,
		FinalResult: result.FinalResult,
		Error:       result.Error,
		TokensUsed:  result.TokensUsed,
	}, nil
}

// aiAgentExecutorAdapter adapts aiservice.AgentService to service.AgentExecutorInterface.
type aiAgentExecutorAdapter struct {
	agentSvc *aiservice.AgentService
}

func (a *aiAgentExecutorAdapter) DispatchAgent(agentID uint64, userID uint64, task string, ctx *service.AgentDispatchContext) (*service.AgentDispatchResult, error) {
	// Convert service.AgentDispatchContext to aiservice.DispatchContext
	dispatchCtx := &aiservice.DispatchContext{
		IssueID:     ctx.IssueID,
		ProjectID:   ctx.ProjectID,
		WorkspaceID: ctx.WorkspaceID,
		TriggeredBy: ctx.TriggeredBy,
	}

	// Call the real AgentService
	result, err := a.agentSvc.DispatchAgent(agentID, userID, task, dispatchCtx)
	if err != nil {
		return nil, err
	}

	// Convert model.AgentActivity to service.AgentDispatchResult
	return &service.AgentDispatchResult{
		ResultSummary: result.ResultSummary,
	}, nil
}
