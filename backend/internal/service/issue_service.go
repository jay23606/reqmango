package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/dto/response"
	"github.com/reqmango/backend/internal/issuetype"
	"github.com/reqmango/backend/internal/model"
	"github.com/reqmango/backend/internal/rql"
	"github.com/reqmango/backend/internal/security"
	"github.com/reqmango/backend/internal/workflow"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IssueService struct {
	db              *gorm.DB
	notificationSvc *NotificationService
	webhookSvc      *WebhookService
	automationSvc   *AutomationService
	slackSvc        *SlackService
	chatSvc         *ChatService
}

func NewIssueService(db *gorm.DB, notificationSvc *NotificationService, webhookSvc *WebhookService, automationSvc *AutomationService, slackSvc *SlackService) *IssueService {
	return &IssueService{db: db, notificationSvc: notificationSvc, webhookSvc: webhookSvc, automationSvc: automationSvc, slackSvc: slackSvc}
}

// SetChatService injects the chat service for issue state-change -> agent reply hooks.
func (s *IssueService) SetChatService(chatSvc *ChatService) {
	s.chatSvc = chatSvc
}

// DB returns the database instance for use in handlers (for security checks).
func (s *IssueService) DB() *gorm.DB {
	return s.db
}

// validateLabelsBelongToProject returns 400 if any label ID does not belong to the project.
// validateEstimatePointID ensures the point belongs to the project.
// A nil id or id==0 means "no estimate" (cleared).
func (s *IssueService) validateEstimatePointID(projectID uint64, id *uint64) (*uint64, error) {
	if id == nil || *id == 0 {
		return nil, nil
	}
	var point model.EstimatePoint
	if err := s.db.Where("id = ? AND project_id = ?", *id, projectID).First(&point).Error; err != nil {
		return nil, common.BadRequest("Estimate point not found in this project")
	}
	return id, nil
}

// lookupEstimatePointID maps a CSV estimate numeric/name value to a project point id.
func (s *IssueService) lookupEstimatePointID(projectID uint64, value float64) *uint64 {
	intVal := int(value)
	var point model.EstimatePoint
	if err := s.db.Where("project_id = ? AND value = ?", projectID, intVal).Order("sequence ASC").First(&point).Error; err == nil {
		return &point.ID
	}
	name := strconv.FormatFloat(value, 'f', -1, 64)
	if err := s.db.Where("project_id = ? AND name = ?", projectID, name).Order("sequence ASC").First(&point).Error; err == nil {
		return &point.ID
	}
	return nil
}

func validateLabelsBelongToProject(db *gorm.DB, labelIDs []uint64, projectID uint64) error {
	if len(labelIDs) == 0 {
		return nil
	}
	// Deduplicate to handle potential duplicates in the input
	uniqueIDs := make([]uint64, 0, len(labelIDs))
	seen := make(map[uint64]struct{}, len(labelIDs))
	for _, id := range labelIDs {
		if _, ok := seen[id]; !ok {
			uniqueIDs = append(uniqueIDs, id)
			seen[id] = struct{}{}
		}
	}
	var count int64
	if err := db.Model(&model.Label{}).Where("id IN ? AND project_id = ?", uniqueIDs, projectID).Count(&count).Error; err != nil {
		return common.Internal("Failed to validate labels")
	}
	if count != int64(len(uniqueIDs)) {
		return common.BadRequest("Label does not belong to this project")
	}
	return nil
}

// validateTypeVisibleInProject checks that the given issue type ID belongs to
// the workspace and is usable in the project. Visibility matches issuetype.Visible
// (project-private, explicitly imported, or the legacy workspace fallback when
// the project has neither imports nor private types) so the picker and write
// path cannot disagree.
func (s *IssueService) validateTypeVisibleInProject(typeID, projectID, workspaceID uint64) error {
	var t model.IssueType
	if err := s.db.First(&t, typeID).Error; err != nil {
		return common.NotFound("Issue type not found")
	}
	if t.WorkspaceID != workspaceID {
		return common.BadRequest("Issue type does not belong to this workspace")
	}
	visible, err := issuetype.Visible(s.db, workspaceID, &projectID)
	if err != nil {
		return common.Internal("Failed to resolve visible issue types")
	}
	for _, vt := range visible {
		if vt.ID == typeID {
			return nil
		}
	}
	return common.BadRequest("This issue type has not been imported into the project")
}

// resolveProjectDefaultIssueTypeID picks the workspace default type marked is_default in the
// visible set — matching Project.vue defaultTypeId (no fallback to "first visible").
func (s *IssueService) resolveProjectDefaultIssueTypeID(projectID, workspaceID uint64) (*uint64, error) {
	visible, err := issuetype.Visible(s.db, workspaceID, &projectID)
	if err != nil {
		return nil, common.Internal("Failed to resolve issue types")
	}
	for i := range visible {
		if visible[i].IsDefault {
			id := visible[i].ID
			return &id, nil
		}
	}
	return nil, nil
}

// nextIssueSequence holds the project lock until the caller's transaction ends.
func nextIssueSequence(tx *gorm.DB, projectID uint64) (int, error) {
	var project model.Project
	if err := tx.Select("id").Clauses(clause.Locking{Strength: "UPDATE"}).First(&project, projectID).Error; err != nil {
		return 0, err
	}
	var maxSeq int
	if err := tx.Unscoped().Model(&model.Issue{}).Where("project_id = ?", projectID).
		Select("COALESCE(MAX(sequence_id), 0)").Scan(&maxSeq).Error; err != nil {
		return 0, err
	}
	return maxSeq + 1, nil
}

// Create creates a new issue.
func (s *IssueService) Create(req *request.IssueCreateRequest, projectID, workspaceID, userID uint64) (*response.IssueResponse, error) {
	// Validate project exists
	var project model.Project
	if err := s.db.First(&project, projectID).Error; err != nil {
		return nil, common.NotFound("Project not found")
	}

	explicitTypeID := req.TypeID != nil && *req.TypeID > 0
	// Validate issue type is visible in the project when the client sends one
	if explicitTypeID {
		if err := s.validateTypeVisibleInProject(*req.TypeID, projectID, workspaceID); err != nil {
			return nil, err
		}
	}

	hasParent := req.ParentID != nil && *req.ParentID > 0

	// UI default (is_default only) is used for top-level quick-create when type is omitted:
	// mandatory custom-field checks and persisted IssueTypeID. It must not be applied to
	// hierarchy validation or to sub-issues with omitted type (nil type inherits no root-type rules).
	var projectDefaultTypeID *uint64
	if !explicitTypeID {
		var err error
		projectDefaultTypeID, err = s.resolveProjectDefaultIssueTypeID(projectID, workspaceID)
		if err != nil {
			return nil, err
		}
	}

	var persistedTypeID *uint64
	var mandatoryTypeID *uint64
	if explicitTypeID {
		tid := *req.TypeID
		persistedTypeID = &tid
		mandatoryTypeID = &tid
	} else if !hasParent {
		persistedTypeID = projectDefaultTypeID
		mandatoryTypeID = projectDefaultTypeID
	} else {
		persistedTypeID = nil
		mandatoryTypeID = nil
	}

	// Get default state if not specified
	var stateID uint64
	if req.StateID != nil {
		stateID = *req.StateID
		var st model.State
		if err := s.db.Where("id = ? AND project_id = ? AND is_active = ?", stateID, projectID, true).First(&st).Error; err != nil {
			return nil, common.BadRequest("State does not belong to this project")
		}
	} else {
		var defaultState model.State
		if err := s.db.Where("project_id = ? AND is_default = ?", projectID, true).First(&defaultState).Error; err != nil {
			return nil, common.NotFound("No default state found for project")
		}
		stateID = defaultState.ID
	}

	priority := req.Priority
	if priority == "" {
		priority = common.PriorityNone
	}

	descHTML := security.SanitizeHTML(req.DescriptionHTML)
	descStripped := security.StripHTMLTags(descHTML)

	estimatePointID, err := s.validateEstimatePointID(projectID, req.EstimatePointID)
	if err != nil {
		return nil, err
	}

	issue := &model.Issue{
		Name:                req.Name,
		DescriptionHTML:     descHTML,
		DescriptionJSON:     req.DescriptionJSON,
		DescriptionStripped: &descStripped,
		Priority:            priority,
		SortOrder:           65535,
		IsDraft:             false,
		ProjectID:           projectID,
		WorkspaceID:         workspaceID,
		StateID:             stateID,
		ParentID:            req.ParentID,
		ExternalID:          req.ExternalID,
		ExternalSource:      req.ExternalSource,
		CoverImageURL:       req.CoverImageURL,
		IssueTypeID:         persistedTypeID,
		EstimatePointID:     estimatePointID,
	}
	issue.CreatedByID = &userID
	issue.UpdatedByID = &userID

	// Hierarchy validation
	if req.ParentID != nil && *req.ParentID > 0 {
		if *req.ParentID == issue.ID {
			return nil, common.BadRequest("An issue cannot be its own parent")
		}
		if s.checkCircularReference(issue.ID, *req.ParentID) {
			return nil, common.BadRequest("Cannot create circular reference in issue hierarchy")
		}
		var parent model.Issue
		if err := s.db.Preload("IssueType").First(&parent, *req.ParentID).Error; err != nil {
			return nil, common.NotFound("Parent issue not found")
		}
		if parent.Depth >= 5 {
			return nil, common.BadRequest("Maximum hierarchy depth (6 levels) exceeded")
		}
		issue.Depth = parent.Depth + 1
		// Validate type hierarchy (only when the client sent an explicit type)
		if explicitTypeID {
			var childType model.IssueType
			if s.db.First(&childType, *req.TypeID).Error == nil {
				// Root type check: types without parent_type_id and level 0 cannot have parents
				if childType.ParentTypeID == nil && childType.Level == 0 {
					return nil, common.BadRequest("Invalid hierarchy: root-level type cannot have a parent")
				}
				// Level check: child must be at least as deep as parent
				if parent.IssueType.ID != 0 && parent.IssueType.Level > 0 {
					if childType.Level < parent.IssueType.Level {
						return nil, common.BadRequest("Invalid hierarchy: child type level must not be shallower than parent type level")
					}
				}
				// ParentTypeID check: only enforce when childType explicitly defines a parent type
				if childType.ParentTypeID != nil && parent.IssueTypeID != nil {
					if *childType.ParentTypeID != *parent.IssueTypeID {
						return nil, common.BadRequest("Invalid hierarchy: parent type does not allow this child type")
					}
				}
				// AllowedChildTypeIDs whitelist check
				if parent.IssueType.ID != 0 && len(parent.IssueType.AllowedChildTypeIDs) > 0 {
					allowed := false
					for _, allowedID := range parent.IssueType.AllowedChildTypeIDs {
						if allowedID == childType.ID {
							allowed = true
							break
						}
					}
					if !allowed {
						return nil, common.BadRequest("Invalid hierarchy: this child type is not allowed under the parent type")
					}
				}
			}
		}
	}

	// Parse dates
	if req.StartDate != nil {
		if t, err := time.Parse(time.RFC3339, *req.StartDate); err == nil {
			issue.StartDate = &t
		} else {
			return nil, common.BadRequest("Invalid start_date format, expected RFC3339")
		}
	}
	if req.TargetDate != nil {
		if t, err := time.Parse(time.RFC3339, *req.TargetDate); err == nil {
			issue.TargetDate = &t
		} else {
			return nil, common.BadRequest("Invalid target_date format, expected RFC3339")
		}
	}

	// Resolve assignees: request wins; otherwise fall back to project default_assignee_id
	assigneeIDs := req.AssigneeIDs
	if len(assigneeIDs) == 0 && project.DefaultAssigneeID != nil && *project.DefaultAssigneeID > 0 {
		var memberCount int64
		s.db.Model(&model.ProjectMember{}).
			Where("project_id = ? AND user_id = ? AND is_active = ?", projectID, *project.DefaultAssigneeID, true).
			Count(&memberCount)
		if memberCount > 0 {
			assigneeIDs = []uint64{*project.DefaultAssigneeID}
		}
	}

	// Validate assignees are project members
	for _, assigneeID := range assigneeIDs {
		var count int64
		s.db.Model(&model.ProjectMember{}).
			Where("project_id = ? AND user_id = ? AND is_active = ?", projectID, assigneeID, true).
			Count(&count)
		if count == 0 {
			return nil, common.BadRequest(fmt.Sprintf("User %d is not a member of this project", assigneeID))
		}
	}

	// Validate labels belong to this project
	if err := validateLabelsBelongToProject(s.db, req.LabelIDs, projectID); err != nil {
		return nil, err
	}

	// Validate mandatory custom fields (only fields visible under the resolved context)
	var createTypeID uint64
	var createTypePtr *uint64
	if mandatoryTypeID != nil {
		createTypeID = *mandatoryTypeID
		if createTypeID > 0 {
			createTypePtr = &createTypeID
		}
	}
	if err := s.validateMandatoryCustomFields(workspaceID, createTypePtr, req.CustomFieldValues, fieldVisibilityContext{
		IssueTypeID: createTypeID,
		StateID:     stateID,
		Priority:    priority,
		AssigneeIDs: assigneeIDs,
		LabelIDs:    req.LabelIDs,
	}); err != nil {
		return nil, err
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, common.Internal("Failed to begin transaction")
	}
	sequenceID, err := nextIssueSequence(tx, projectID)
	if err != nil {
		tx.Rollback()
		return nil, common.Internal("Failed to allocate issue number")
	}
	issue.SequenceID = sequenceID

	if err := tx.Create(issue).Error; err != nil {
		tx.Rollback()
		return nil, common.Internal("Failed to create issue")
	}

	// Create activity: created
	s.createActivity(tx, issue.ID, "created", nil, nil, nil, nil, &userID)

	// Add assignees
	for _, assigneeID := range assigneeIDs {
		tx.Create(&model.IssueAssignee{IssueID: issue.ID, UserID: assigneeID})
	}

	// Add labels
	for _, labelID := range req.LabelIDs {
		tx.Create(&model.IssueLabel{IssueID: issue.ID, LabelID: labelID})
	}

	// Add cycle
	if req.CycleID != nil && *req.CycleID > 0 {
		tx.Create(&model.IssueCycle{IssueID: issue.ID, CycleID: *req.CycleID})
	}

	// Add modules
	for _, moduleID := range req.ModuleIDs {
		tx.Create(&model.ModuleIssue{IssueID: issue.ID, ModuleID: moduleID})
	}

	// Add release
	if req.ReleaseID != nil && *req.ReleaseID > 0 {
		tx.Create(&model.ReleaseIssue{ReleaseID: *req.ReleaseID, IssueID: issue.ID})
	}

	// Save custom field values
	for fieldID, value := range req.CustomFieldValues {
		var stringValue string
		switch v := value.(type) {
		case string:
			stringValue = v
		default:
			stringValue = fmt.Sprintf("%v", v)
		}
		tx.Create(&model.IssueCustomFieldValue{
			IssueID: issue.ID,
			FieldID: fieldID,
			Value:   stringValue,
		})
	}

	if err := tx.Commit().Error; err != nil {
		return nil, common.Internal("Failed to commit transaction")
	}

	if s.notificationSvc != nil && len(assigneeIDs) > 0 {
		actor := &userID
		projectIDPtr := issue.ProjectID
		issueIDPtr := issue.ID
		for _, aid := range assigneeIDs {
			title := fmt.Sprintf("工作项分配: %s", issue.Name)
			message := fmt.Sprintf("您被分配到工作项 #%d: %s", issue.SequenceID, issue.Name)
			_ = s.notificationSvc.TriggerNotification(s.db, "issue_assigned", title, message, aid, actor, &projectIDPtr, &issueIDPtr)
		}
	}

	// Automation trigger: issue_created (after commit, uses own DB connection)
	s.NotifyIssueCreated(issue)

	s.webhookSvc.Fire(projectID, "issue_created", map[string]interface{}{"issue_id": issue.ID, "name": issue.Name, "priority": issue.Priority})
	return s.buildResponse(issue.ID)
}

// NotifyIssueCreated publishes issue.created automations for an issue created
// outside Create (e.g. public Intake submit). Field names match Create's path.
func (s *IssueService) NotifyIssueCreated(issue *model.Issue) {
	if issue == nil {
		return
	}
	ctx := map[string]interface{}{
		"issue_id":   issue.ID,
		"priority":   issue.Priority,
		"state_id":   issue.StateID,
		"project_id": issue.ProjectID,
	}
	if issue.IntakeStatus != nil {
		ctx["intake_status"] = *issue.IntakeStatus
	}
	s.runAutomations(issue.ID, "issue.created", ctx)

	// Notify project lead + subscribers (settings that previously had no issue effect).
	if s.notificationSvc == nil {
		return
	}
	exclude := uint64(0)
	if issue.CreatedByID != nil {
		exclude = *issue.CreatedByID
	}
	recipients := make(map[uint64]bool)
	s.addProjectLeadAndSubscribers(issue.ProjectID, exclude, recipients)
	if len(recipients) == 0 {
		return
	}
	ids := make([]uint64, 0, len(recipients))
	for id := range recipients {
		ids = append(ids, id)
	}
	title := fmt.Sprintf("新工作项: %s", issue.Name)
	message := fmt.Sprintf("项目中创建了工作项 #%d", issue.SequenceID)
	issueIDPtr := issue.ID
	projectIDPtr := issue.ProjectID
	var actor *uint64
	if exclude > 0 {
		actor = &exclude
	}
	_ = s.notificationSvc.TriggerNotificationsBulk(s.db, "issue_created", title, message, ids, actor, &projectIDPtr, &issueIDPtr)
}

// GetByID returns an issue with all relations loaded.
func (s *IssueService) GetByID(issueID uint64) (*response.IssueResponse, error) {
	return s.buildResponse(issueID)
}

// buildSortClause builds ORDER BY clause from sort_by/sort_dir (single, backward compat)
// or sort_config (multi-field sort, preferred).
func (s *IssueService) buildSortClause(filters map[string]interface{}) string {
	colMap := map[string]string{
		"sequence_id": "issues.sequence_id",
		"name":        "issues.name",
		"priority":    "issues.priority",
		"state":       "issues.state_id",
		"issue_type":  "issues.issue_type_id",
		"created_at":  "issues.created_at",
		"updated_at":  "issues.updated_at",
		"start_date":  "issues.start_date",
		"target_date": "issues.target_date",
	}

	// Prefer sort_config (multi-field)
	if sortConfig, ok := filters["sort_config"].([]map[string]string); ok && len(sortConfig) > 0 {
		var parts []string
		for _, entry := range sortConfig {
			field := entry["field"]
			dir := entry["dir"]
			if dir != "asc" && dir != "desc" {
				dir = "desc"
			}
			if col, ok := colMap[field]; ok {
				parts = append(parts, fmt.Sprintf("%s %s", col, dir))
			}
		}
		if len(parts) > 0 {
			return fmt.Sprintf("%s, issues.sort_order ASC, issues.sequence_id DESC", strings.Join(parts, ", "))
		}
		return ""
	}

	// Fallback to sort_by / sort_dir (single, backward compat)
	sortBy, _ := filters["sort_by"].(string)
	sortDir, _ := filters["sort_dir"].(string)
	if sortBy == "" {
		return ""
	}
	if sortDir == "" {
		sortDir = "desc"
	}
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "desc"
	}
	if col, ok := colMap[sortBy]; ok {
		return fmt.Sprintf("%s %s, issues.sort_order ASC, issues.sequence_id DESC", col, sortDir)
	}
	return ""
}

// List returns issues for a project with optional filters and total count.
func (s *IssueService) List(projectID uint64, filters map[string]interface{}, limit, offset int) ([]response.IssueResponse, int64, error) {
	baseQuery := s.db.Model(&model.Issue{}).Where("issues.project_id = ?", projectID).Where(intakeVisibleClause)

	// Build dynamic ORDER BY from sort_by / sort_dir
	sortClause := s.buildSortClause(filters)

	// Handle RQL filter: use RQL service to get matching issue IDs
	if rqlQuery, ok := filters["rql"]; ok && rqlQuery != nil && rqlQuery != "" {
		queryStr := fmt.Sprint(rqlQuery)
		if queryStr != "" {
			rqlSvc := rql.NewRQLService()
			page := 1
			if limit > 0 {
				page = (offset / limit) + 1
			}
			// Extract current user ID for template variable resolution ($CURRENT_USER)
			var currentUserID uint64
			if uid, ok := filters["current_user_id"]; ok {
				switch v := uid.(type) {
				case uint64:
					currentUserID = v
				case uint:
					currentUserID = uint64(v)
				case float64:
					currentUserID = uint64(v)
				}
			}
			rqlIssues, rqlTotal, rqlErr := rqlSvc.SearchIssuesWithUser(s.db, projectID, queryStr, page, limit, currentUserID)
			if rqlErr != nil {
				return nil, 0, common.BadRequest("RQL parse error: " + rqlErr.Error())
			}
			if len(rqlIssues) == 0 {
				return []response.IssueResponse{}, rqlTotal, nil
			}
			ids := make([]uint64, len(rqlIssues))
			for i, issue := range rqlIssues {
				ids[i] = issue.ID
			}
			var issues []model.Issue
			q := s.db.Where("id IN ?", ids).
				Preload("State").
				Preload("IssueType").
				Preload("AssigneeLinks.User").
				Preload("LabelLinks.Label").
				Preload("CycleLink")
			if sortClause != "" {
				q = q.Order(sortClause)
			} else {
				q = q.Order("sort_order ASC, sequence_id DESC")
			}
			if err := q.Find(&issues).Error; err != nil {
				return nil, 0, common.Internal("Database error")
			}
			result := make([]response.IssueResponse, len(issues))
			for i, issue := range issues {
				resp, err := s.BuildIssueResponse(&issue)
				if err != nil {
					return nil, 0, err
				}
				result[i] = *resp
			}
			return result, rqlTotal, nil
		}
	}

	// Apply filters
	if stateID, ok := filters["state_id"]; ok && stateID != nil {
		baseQuery = baseQuery.Where("issues.state_id = ?", stateID)
	}
	if priority, ok := filters["priority"]; ok && priority != nil {
		baseQuery = baseQuery.Where("issues.priority = ?", priority)
	}
	if assigneeID, ok := filters["assignee_id"]; ok && assigneeID != nil {
		baseQuery = baseQuery.Joins("JOIN issue_assignees ON issue_assignees.issue_id = issues.id").
			Where("issue_assignees.user_id = ?", assigneeID)
	}
	if parentID, ok := filters["parent_id"]; ok && parentID != nil {
		baseQuery = baseQuery.Where("issues.parent_id = ?", parentID)
	}
	if cycleID, ok := filters["cycle_id"]; ok && cycleID != nil {
		baseQuery = baseQuery.Joins("JOIN issue_cycles ON issue_cycles.issue_id = issues.id").
			Where("issue_cycles.cycle_id = ?", cycleID)
	}
	if isDraft, ok := filters["is_draft"]; ok && isDraft != nil {
		baseQuery = baseQuery.Where("issues.is_draft = ?", isDraft)
	}
	if search, ok := filters["search"]; ok && search != nil {
		searchStr := fmt.Sprintf("%%%s%%", search)
		baseQuery = baseQuery.Where("issues.name ILIKE ? OR COALESCE(issues.description_stripped, '') ILIKE ?",
			searchStr, searchStr)
	}
	if issueTypeID, ok := filters["issue_type_id"]; ok && issueTypeID != nil {
		baseQuery = baseQuery.Where("issues.issue_type_id = ?", issueTypeID)
	}
	needDedup := false
	// Single CF filter (legacy)
	if cfFieldID, ok := filters["cf_field_id"]; ok && cfFieldID != nil {
		alias := fmt.Sprintf("cfv_%d", cfFieldID)
		baseQuery = baseQuery.Joins(
			fmt.Sprintf("JOIN issue_custom_field_values %s ON %s.issue_id = issues.id AND %s.field_id = ?",
				alias, alias, alias), cfFieldID)
		if cfValue, ok2 := filters["cf_value"]; ok2 && cfValue != nil && cfValue != "" {
			baseQuery = baseQuery.Where(fmt.Sprintf("%s.value ILIKE ?", alias), "%"+cfValue.(string)+"%")
		}
		needDedup = true
	}
	// Multi-CF AND filter
	if cfAndConditions, ok := filters["cf_and"]; ok && cfAndConditions != nil {
		if conditions, ok2 := cfAndConditions.([]map[string]interface{}); ok2 {
			for i, cond := range conditions {
				fid, _ := cond["field_id"].(float64)
				val, _ := cond["value"].(string)
				if fid > 0 {
					alias := fmt.Sprintf("cfv_and_%d", i)
					baseQuery = baseQuery.Joins(
						fmt.Sprintf("JOIN issue_custom_field_values %s ON %s.issue_id = issues.id AND %s.field_id = ?",
							alias, alias, alias), uint64(fid))
					if val != "" {
						baseQuery = baseQuery.Where(fmt.Sprintf("%s.value ILIKE ?", alias), "%"+val+"%")
					}
				}
			}
			needDedup = true
		}
	}

	// Count total (before limit/offset)
	var total int64
	countQuery := baseQuery.Session(&gorm.Session{})
	if needDedup {
		countQuery = countQuery.Distinct("issues.id")
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, common.Internal("Database error")
	}

	var issues []model.Issue
	defaultOrder := "issues.sort_order ASC, issues.sequence_id DESC"
	if sortClause != "" {
		defaultOrder = sortClause
	}

	if needDedup {
		// Use subquery to get distinct issue IDs, then fetch full rows (with sort applied in subquery)
		var ids []uint64
		subBase := baseQuery
		if sortClause != "" {
			subBase = subBase.Order(sortClause)
		}
		if err := subBase.Distinct("issues.id").Limit(limit).Offset(offset).Pluck("issues.id", &ids).Error; err != nil {
			return nil, 0, common.Internal("Database error")
		}
		if len(ids) > 0 {
			idOrder := "CASE issues.id "
			for i, id := range ids {
				idOrder += fmt.Sprintf("WHEN %d THEN %d ", id, i)
			}
			idOrder += "END"
			if err := s.db.Where("id IN ?", ids).
				Preload("State").
				Preload("IssueType").
				Preload("AssigneeLinks.User").
				Preload("LabelLinks.Label").
				Preload("CycleLink").
				Order(idOrder).
				Find(&issues).Error; err != nil {
				return nil, 0, common.Internal("Database error")
			}
		}
	} else {
		if err := baseQuery.
			Preload("State").
			Preload("IssueType").
			Preload("AssigneeLinks.User").
			Preload("LabelLinks.Label").
			Preload("CycleLink").
			Order(defaultOrder).
			Limit(limit).Offset(offset).
			Find(&issues).Error; err != nil {
			return nil, 0, common.Internal("Database error")
		}
	}

	result := make([]response.IssueResponse, len(issues))
	for i, issue := range issues {
		resp, err := s.BuildIssueResponse(&issue)
		if err != nil {
			return nil, 0, err
		}
		result[i] = *resp
	}
	return result, total, nil
}

// ListByWorkspace returns issues across all projects in a workspace.
func (s *IssueService) ListByWorkspace(workspaceID uint64, filters map[string]interface{}, limit, offset int) ([]response.IssueResponse, int64, error) {
	var projectIDs []uint64
	if err := s.db.Model(&model.Project{}).Where("workspace_id = ?", workspaceID).Pluck("id", &projectIDs).Error; err != nil {
		return nil, 0, err
	}
	if len(projectIDs) == 0 {
		return []response.IssueResponse{}, 0, nil
	}
	return s.listByProjects(projectIDs, filters, limit, offset)
}

func (s *IssueService) listByProjects(projectIDs []uint64, filters map[string]interface{}, limit, offset int) ([]response.IssueResponse, int64, error) {
	baseQuery := s.db.Model(&model.Issue{}).Where("issues.project_id IN ?", projectIDs).Where(intakeVisibleClause)

	sortClause := s.buildSortClause(filters)

	if stateID, ok := filters["state_id"]; ok && stateID != nil {
		baseQuery = baseQuery.Where("issues.state_id = ?", stateID)
	}
	if priority, ok := filters["priority"]; ok && priority != nil {
		baseQuery = baseQuery.Where("issues.priority = ?", priority)
	}
	if assigneeID, ok := filters["assignee_id"]; ok && assigneeID != nil {
		baseQuery = baseQuery.Joins("JOIN issue_assignees ON issue_assignees.issue_id = issues.id").
			Where("issue_assignees.user_id = ?", assigneeID)
	}
	if cycleID, ok := filters["cycle_id"]; ok && cycleID != nil {
		baseQuery = baseQuery.Joins("JOIN issue_cycles ON issue_cycles.issue_id = issues.id").
			Where("issue_cycles.cycle_id = ?", cycleID)
	}
	if search, ok := filters["search"]; ok && search != nil {
		searchStr := fmt.Sprintf("%%%s%%", search)
		baseQuery = baseQuery.Where("issues.name ILIKE ? OR COALESCE(issues.description_stripped, '') ILIKE ?",
			searchStr, searchStr)
	}
	if issueTypeID, ok := filters["issue_type_id"]; ok && issueTypeID != nil {
		baseQuery = baseQuery.Where("issues.issue_type_id = ?", issueTypeID)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	defaultOrder := "issues.sort_order ASC, issues.sequence_id DESC"
	if sortClause != "" {
		defaultOrder = sortClause
	}

	var issues []model.Issue
	if err := baseQuery.
		Preload("State").
		Preload("IssueType").
		Preload("AssigneeLinks.User").
		Preload("LabelLinks.Label").
		Preload("CycleLink").
		Preload("Parent").
		Preload("Project").
		Order(defaultOrder).
		Offset(offset).
		Limit(limit).
		Find(&issues).Error; err != nil {
		return nil, 0, err
	}

	result := make([]response.IssueResponse, len(issues))
	for i, issue := range issues {
		resp, err := s.BuildIssueResponse(&issue)
		if err != nil {
			return nil, 0, err
		}
		result[i] = *resp
	}
	return result, total, nil
}

// Update updates an issue and records activity for changes.
func (s *IssueService) Update(issueID uint64, req *request.IssueUpdateRequest, userID uint64) (*response.IssueResponse, error) {
	var issue model.Issue
	if err := s.db.First(&issue, issueID).Error; err != nil {
		return nil, common.NotFound("Issue not found")
	}

	var priorAssigneeIDs []uint64
	s.db.Model(&model.IssueAssignee{}).Where("issue_id = ?", issueID).Pluck("user_id", &priorAssigneeIDs)

	tx := s.db.Begin()

	// Track changes for activity
	hasChanges := false
	var oldStateID uint64         // Store old state ID for automation context
	oldPriority := issue.Priority // Store old priority for automation context
	var stateChangeNotify bool
	var stateChangeOldName, stateChangeNewName string

	if req.Name != nil && *req.Name != issue.Name {
		oldVal := issue.Name
		s.createActivity(tx, issueID, "updated", strPtr("name"), strPtr(oldVal), req.Name, nil, &userID)
		issue.Name = *req.Name
		hasChanges = true
	}
	if req.DescriptionHTML != nil && *req.DescriptionHTML != issue.DescriptionHTML {
		s.createActivity(tx, issueID, "updated", strPtr("description"), nil, nil, nil, &userID)
		issue.DescriptionHTML = security.SanitizeHTML(*req.DescriptionHTML)
		descStripped := security.StripHTMLTags(issue.DescriptionHTML)
		issue.DescriptionStripped = &descStripped
		hasChanges = true
	}
	if req.Priority != nil && *req.Priority != issue.Priority {
		oldVal := issue.Priority
		s.createActivity(tx, issueID, "updated", strPtr("priority"), strPtr(oldVal), req.Priority, nil, &userID)
		issue.Priority = *req.Priority
		hasChanges = true
	}
	if req.StateID != nil && *req.StateID != issue.StateID {
		oldStateID = issue.StateID
		newStateID := *req.StateID
		var st model.State
		if err := tx.Where("id = ? AND project_id = ? AND is_active = ?", newStateID, issue.ProjectID, true).First(&st).Error; err != nil {
			tx.Rollback()
			return nil, common.BadRequest("State does not belong to this project")
		}
		if err := s.validateStateTransition(tx, issue.ProjectID, issueID, oldStateID, newStateID, userID); err != nil {
			tx.Rollback()
			return nil, err
		}
		issue.StateID = newStateID

		var oldState, newState model.State
		s.db.First(&oldState, oldStateID)
		if err := s.db.First(&newState, newStateID).Error; err == nil {
			if newState.Group == common.StateGroupCompleted {
				now := time.Now()
				issue.CompletedAt = &now
			} else {
				issue.CompletedAt = nil
			}
		}

		s.createActivity(tx, issueID, "updated", strPtr("state"),
			strPtr(oldState.Name), strPtr(newState.Name), nil, &userID)
		hasChanges = true

		stateChangeNotify = true
		stateChangeOldName = oldState.Name
		stateChangeNewName = newState.Name
	}
	if req.SortOrder != nil && *req.SortOrder != issue.SortOrder {
		issue.SortOrder = *req.SortOrder
		hasChanges = true
	}

	// Save basic fields
	if hasChanges {
		if err := tx.Save(&issue).Error; err != nil {
			tx.Rollback()
			return nil, common.Internal("Failed to update issue")
		}
	}

	// Handle assignees
	if req.AssigneeIDs != nil {
		oldSet := make(map[uint64]bool, len(priorAssigneeIDs))
		for _, id := range priorAssigneeIDs {
			oldSet[id] = true
		}
		newSet := make(map[uint64]bool, len(req.AssigneeIDs))
		for _, aid := range req.AssigneeIDs {
			newSet[aid] = true
		}
		// Remove existing
		tx.Where("issue_id = ?", issueID).Delete(&model.IssueAssignee{})
		// Add new
		for _, aid := range req.AssigneeIDs {
			tx.Create(&model.IssueAssignee{IssueID: issueID, UserID: aid})
		}
		s.createActivity(tx, issueID, "updated", strPtr("assignees"), nil, nil, nil, &userID)
		hasChanges = true

		if s.notificationSvc != nil {
			projPtr := issue.ProjectID
			issuePtr := issueID
			for aid := range newSet {
				if oldSet[aid] {
					continue
				}
				title := fmt.Sprintf("工作项分配: %s", issue.Name)
				message := fmt.Sprintf("您被分配到工作项 #%d: %s", issue.SequenceID, issue.Name)
				_ = s.notificationSvc.TriggerNotification(tx, "issue_assigned", title, message, aid, &userID, &projPtr, &issuePtr)
			}
		}
	}

	if stateChangeNotify && s.notificationSvc != nil {
		recipientIDs := make(map[uint64]bool)
		var assignees []model.IssueAssignee
		tx.Where("issue_id = ?", issueID).Find(&assignees)
		for _, a := range assignees {
			if a.UserID != userID {
				recipientIDs[a.UserID] = true
			}
		}
		var watchers []model.IssueWatcher
		tx.Where("issue_id = ?", issueID).Find(&watchers)
		for _, w := range watchers {
			if w.UserID != userID {
				recipientIDs[w.UserID] = true
			}
		}
		s.addProjectLeadAndSubscribers(issue.ProjectID, userID, recipientIDs)
		if len(recipientIDs) > 0 {
			ids := make([]uint64, 0, len(recipientIDs))
			for id := range recipientIDs {
				ids = append(ids, id)
			}
			title := fmt.Sprintf("状态变更: %s", issue.Name)
			message := fmt.Sprintf("工作项 #%d 状态从 %s 变为 %s", issue.SequenceID, stateChangeOldName, stateChangeNewName)
			issueIDPtr := issueID
			projectIDPtr := issue.ProjectID
			_ = s.notificationSvc.TriggerNotificationsBulk(tx, "issue_state_changed", title, message, ids, &userID, &projectIDPtr, &issueIDPtr)
		}
	}

	// Handle labels
	if req.LabelIDs != nil {
		// Validate labels belong to this issue's project
		if err := validateLabelsBelongToProject(tx, req.LabelIDs, issue.ProjectID); err != nil {
			tx.Rollback()
			return nil, err
		}
		tx.Where("issue_id = ?", issueID).Delete(&model.IssueLabel{})
		for _, lid := range req.LabelIDs {
			tx.Create(&model.IssueLabel{IssueID: issueID, LabelID: lid})
		}
		s.createActivity(tx, issueID, "updated", strPtr("labels"), nil, nil, nil, &userID)
		hasChanges = true
	}

	// Handle cycle
	if req.CycleID != nil {
		tx.Where("issue_id = ?", issueID).Delete(&model.IssueCycle{})
		tx.Create(&model.IssueCycle{IssueID: issueID, CycleID: *req.CycleID})
		s.createActivity(tx, issueID, "updated", strPtr("cycle"), nil, nil, nil, &userID)
		hasChanges = true
	}

	// Handle modules
	if req.ModuleIDs != nil {
		tx.Where("issue_id = ?", issueID).Delete(&model.ModuleIssue{})
		for _, mid := range req.ModuleIDs {
			tx.Create(&model.ModuleIssue{IssueID: issueID, ModuleID: mid})
		}
		s.createActivity(tx, issueID, "updated", strPtr("module"), nil, nil, nil, &userID)
		hasChanges = true
	}

	// Handle estimate point (0 clears; omit leaves unchanged)
	if req.EstimatePointID != nil {
		pointID, estErr := s.validateEstimatePointID(issue.ProjectID, req.EstimatePointID)
		if estErr != nil {
			tx.Rollback()
			return nil, estErr
		}
		issue.EstimatePointID = pointID
		if err := tx.Model(&issue).Update("estimate_point_id", pointID).Error; err != nil {
			tx.Rollback()
			return nil, common.Internal("Failed to update estimate point")
		}
		s.createActivity(tx, issueID, "updated", strPtr("estimate_point"), nil, nil, nil, &userID)
		hasChanges = true
	}

	// Handle release
	if req.ReleaseID != nil {
		result := tx.Where("issue_id = ?", issueID).Delete(&model.ReleaseIssue{})
		if *req.ReleaseID > 0 {
			tx.Create(&model.ReleaseIssue{ReleaseID: *req.ReleaseID, IssueID: issueID})
		}
		if result.RowsAffected > 0 || *req.ReleaseID > 0 {
			s.createActivity(tx, issueID, "updated", strPtr("release"), nil, nil, nil, &userID)
			hasChanges = true
		}
	}

	// Handle issue type
	if req.TypeID != nil && (issue.IssueTypeID == nil || *req.TypeID != *issue.IssueTypeID) {
		var newType model.IssueType
		if err := s.db.First(&newType, *req.TypeID).Error; err == nil {
			// Validate issue type is visible in the project
			if err := s.validateTypeVisibleInProject(*req.TypeID, issue.ProjectID, issue.WorkspaceID); err != nil {
				tx.Rollback()
				return nil, err
			}
			// Validate: if issue has children, new type must be a higher level than children
			var childCount int64
			s.db.Model(&model.Issue{}).Where("parent_id = ?", issueID).Count(&childCount)
			if childCount > 0 {
				var childLevel int
				s.db.Model(&model.Issue{}).Select("COALESCE(MAX(depth),0)").Where("parent_id = ?", issueID).Scan(&childLevel)
				if newType.Level >= childLevel && childLevel > 0 {
					tx.Rollback()
					return nil, common.BadRequest("Type change blocked: children require parent type at a higher level")
				}
			}
			// Validate: if issue has parent, new type must be deeper than parent
			if issue.ParentID != nil {
				var parent model.Issue
				if s.db.Preload("IssueType").First(&parent, *issue.ParentID).Error == nil && parent.IssueType.ID != 0 {
					if newType.Level <= parent.IssueType.Level {
						tx.Rollback()
						return nil, common.BadRequest("Type change blocked: must be deeper than parent type")
					}
					// Also check ParentTypeID if explicitly set
					if newType.ParentTypeID != nil && parent.IssueTypeID != nil {
						if *newType.ParentTypeID != *parent.IssueTypeID {
							tx.Rollback()
							return nil, common.BadRequest("Type change blocked: parent type does not allow this child type")
						}
					}
				}
			}
		}
		issue.IssueTypeID = req.TypeID
		tx.Save(&issue)
		s.createActivity(tx, issueID, "updated", strPtr("issue_type"), nil, nil, nil, &userID)
		hasChanges = true
	}

	// Parse dates
	if req.StartDate != nil {
		t, err := time.Parse(time.RFC3339, *req.StartDate)
		if err != nil {
			tx.Rollback()
			return nil, common.BadRequest("Invalid start_date format, expected RFC3339")
		}
		s.createActivity(tx, issueID, "updated", strPtr("start_date"), nil, nil, nil, &userID)
		issue.StartDate = &t
		tx.Save(&issue)
		hasChanges = true
	}
	if req.TargetDate != nil {
		t, err := time.Parse(time.RFC3339, *req.TargetDate)
		if err != nil {
			tx.Rollback()
			return nil, common.BadRequest("Invalid target_date format, expected RFC3339")
		}
		s.createActivity(tx, issueID, "updated", strPtr("target_date"), nil, nil, nil, &userID)
		issue.TargetDate = &t
		tx.Save(&issue)
		hasChanges = true
	}

	if req.CoverImageURL != nil {
		issue.CoverImageURL = req.CoverImageURL
		tx.Save(&issue)
		hasChanges = true
	}

	// Handle parent
	if req.ParentID != nil {
		if *req.ParentID == 0 {
			issue.ParentID = nil
			issue.Depth = 0
		} else {
			if *req.ParentID == issueID {
				tx.Rollback()
				return nil, common.BadRequest("An issue cannot be its own parent")
			}
			if s.checkCircularReference(issueID, *req.ParentID) {
				tx.Rollback()
				return nil, common.BadRequest("Cannot create circular reference in issue hierarchy")
			}
			var parent model.Issue
			if err := tx.Preload("IssueType").First(&parent, *req.ParentID).Error; err != nil {
				tx.Rollback()
				return nil, common.NotFound("Parent issue not found")
			}
			if parent.Depth >= 5 {
				tx.Rollback()
				return nil, common.BadRequest("Maximum hierarchy depth (6 levels) exceeded")
			}
			issue.ParentID = req.ParentID
			issue.Depth = parent.Depth + 1
			// Validate type hierarchy
			var childType model.IssueType
			if issue.IssueTypeID != nil {
				if err := tx.First(&childType, *issue.IssueTypeID).Error; err == nil {
					// Root type check: types without parent_type_id and level 0 cannot have parents
					if childType.ParentTypeID == nil && childType.Level == 0 {
						tx.Rollback()
						return nil, common.BadRequest("Invalid hierarchy: root-level type cannot have a parent")
					}
					// Level check: child must be at least as deep as parent
					if parent.IssueType.ID != 0 && parent.IssueType.Level > 0 {
						if childType.Level < parent.IssueType.Level {
							tx.Rollback()
							return nil, common.BadRequest("Invalid hierarchy: child type level must not be shallower than parent type level")
						}
					}
					// ParentTypeID check: only enforce when childType explicitly defines a parent type
					if childType.ParentTypeID != nil && parent.IssueTypeID != nil {
						if *childType.ParentTypeID != *parent.IssueTypeID {
							tx.Rollback()
							return nil, common.BadRequest("Invalid hierarchy: parent type does not allow this child type")
						}
					}
					// AllowedChildTypeIDs whitelist check
					if parent.IssueType.ID != 0 && len(parent.IssueType.AllowedChildTypeIDs) > 0 {
						allowed := false
						for _, allowedID := range parent.IssueType.AllowedChildTypeIDs {
							if allowedID == childType.ID {
								allowed = true
								break
							}
						}
						if !allowed {
							tx.Rollback()
							return nil, common.BadRequest("Invalid hierarchy: this child type is not allowed under the parent type")
						}
					}
				}
			}
		}
		tx.Save(&issue)
		s.createActivity(tx, issueID, "updated", strPtr("parent"), nil, nil, nil, &userID)
		hasChanges = true
	}

	if !hasChanges {
		tx.Rollback()
		return s.buildResponse(issueID)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, common.Internal("Failed to commit update")
	}

	event := "issue_updated"
	if req.StateID != nil {
		event = "state_changed"
	}

	// Automation trigger: fire after commit for state changes
	if req.StateID != nil {
		var newState model.State
		s.db.First(&newState, *req.StateID)

		s.runAutomations(issueID, "issue.state_changed", map[string]interface{}{
			"issue_id":    issueID,
			"old_state":   fmt.Sprintf("%d", oldStateID),
			"new_state":   fmt.Sprintf("%d", *req.StateID),
			"state_group": newState.Group,
			"project_id":  issue.ProjectID,
			"priority":    issue.Priority,
		})

		// Chat hook: trigger agent auto-reply on issue state change (async, non-blocking)
		if s.chatSvc != nil && oldStateID != 0 {
			go s.chatSvc.OnIssueStateChanged(context.Background(), issueID, oldStateID, *req.StateID, userID)
		}
	}

	// Automation trigger: fire after commit for issue updates (including priority changes)
	if hasChanges && req.StateID == nil {
		log.Printf("[IssueService] Triggering automation: issue.updated for issue %d, priority: %s (old: %s)", issueID, issue.Priority, oldPriority)
		s.runAutomations(issueID, "issue.updated", map[string]interface{}{
			"issue_id":     issueID,
			"project_id":   issue.ProjectID,
			"priority":     issue.Priority,
			"old_priority": oldPriority,
			"new_priority": issue.Priority,
			"state_id":     issue.StateID,
		})
	}

	s.webhookSvc.Fire(issue.ProjectID, event, map[string]interface{}{"issue_id": issueID, "name": issue.Name, "priority": issue.Priority, "state_id": issue.StateID})

	var assignees []model.IssueAssignee
	s.db.Where("issue_id = ?", issueID).Find(&assignees)
	for _, a := range assignees {
		if a.UserID != userID {
			SSE.NotifyUser(a.UserID, "issue_updated", "工作项已更新", issue.Name)
		}
	}

	return s.buildResponse(issueID)
}

// Delete performs a soft delete on an issue.
func (s *IssueService) Delete(issueID, userID uint64) error {
	var issue model.Issue
	if err := s.db.Preload("Project").First(&issue, issueID).Error; err != nil {
		return common.NotFound("Issue not found")
	}

	if err := s.checkProjectMembership(issue.ProjectID, userID); err != nil {
		return err
	}

	return s.db.Delete(&issue).Error
}

// Archive archives an issue.
func (s *IssueService) Archive(issueID, userID uint64) error {
	var issue model.Issue
	if err := s.db.Preload("Project").First(&issue, issueID).Error; err != nil {
		return common.NotFound("Issue not found")
	}

	if err := s.checkProjectMembership(issue.ProjectID, userID); err != nil {
		return err
	}

	now := time.Now()
	issue.ArchivedAt = &now
	return s.db.Save(&issue).Error
}

// Restore restores an archived or deleted issue.
func (s *IssueService) Restore(issueID, userID uint64) (*response.IssueResponse, error) {
	var issue model.Issue
	if err := s.db.Unscoped().First(&issue, issueID).Error; err != nil {
		return nil, common.NotFound("Issue not found")
	}

	if err := s.checkProjectMembership(issue.ProjectID, userID); err != nil {
		return nil, err
	}

	issue.ArchivedAt = nil
	issue.DeletedAt = gorm.DeletedAt{}
	s.db.Unscoped().Save(&issue)

	return s.buildResponse(issueID)
}

func (s *IssueService) checkProjectMembership(projectID, userID uint64) error {
	var count int64
	s.db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND is_active = ?", projectID, userID, true).
		Count(&count)
	if count == 0 {
		return common.Forbidden("You must be a member of the project to perform this action")
	}
	return nil
}

func (s *IssueService) checkCircularReference(issueID, parentID uint64) bool {
	visited := make(map[uint64]bool)
	currentID := parentID
	for {
		if currentID == 0 {
			return false
		}
		if currentID == issueID {
			return true
		}
		if visited[currentID] {
			return false
		}
		visited[currentID] = true
		var issue model.Issue
		if err := s.db.Select("parent_id").First(&issue, currentID).Error; err != nil {
			return false
		}
		if issue.ParentID == nil {
			return false
		}
		currentID = *issue.ParentID
	}
}

// AddAssignee adds an assignee to an issue.
func (s *IssueService) AddAssignee(issueID, userID, actorID uint64) error {
	// Verify issue exists
	var issue model.Issue
	if err := s.db.Preload("Project").First(&issue, issueID).Error; err != nil {
		return common.NotFound("Issue not found")
	}

	// Check if already assigned
	var count int64
	s.db.Model(&model.IssueAssignee{}).
		Where("issue_id = ? AND user_id = ?", issueID, userID).Count(&count)
	if count > 0 {
		return common.Conflict("User is already assigned to this issue")
	}

	if err := s.db.Create(&model.IssueAssignee{IssueID: issueID, UserID: userID}).Error; err != nil {
		return common.Internal("Failed to add assignee")
	}

	s.createActivity(s.db, issueID, "updated", strPtr("assignees"), nil, nil, nil, &actorID)

	// Trigger notification for assignment
	if s.notificationSvc != nil {
		title := fmt.Sprintf("工作项分配: %s", issue.Name)
		message := fmt.Sprintf("您被分配到工作项 #%d: %s", issue.SequenceID, issue.Name)
		issueIDPtr := issueID
		projectIDPtr := issue.ProjectID
		_ = s.notificationSvc.TriggerNotification(s.db, "issue_assigned", title, message, userID, &actorID, &projectIDPtr, &issueIDPtr)
	}

	// Automation trigger: issue.assigned
	s.runAutomations(issueID, "issue.assigned", map[string]interface{}{
		"issue_id": issueID, "project_id": issue.ProjectID,
		"new_assignee": userID, "actor_id": actorID,
	})

	return nil
}

// RemoveAssignee removes an assignee from an issue.
func (s *IssueService) RemoveAssignee(issueID, userID, actorID uint64) error {
	result := s.db.Where("issue_id = ? AND user_id = ?", issueID, userID).Delete(&model.IssueAssignee{})
	if result.RowsAffected == 0 {
		return common.NotFound("Assignee not found on this issue")
	}

	s.createActivity(s.db, issueID, "updated", strPtr("assignees"), nil, nil, nil, &actorID)
	return nil
}

// AddLabel adds a label to an issue.
func (s *IssueService) AddLabel(issueID, labelID, actorID uint64) error {
	var issue model.Issue
	if err := s.db.First(&issue, issueID).Error; err != nil {
		return common.NotFound("Issue not found")
	}

	var label model.Label
	if err := s.db.First(&label, labelID).Error; err != nil {
		return common.NotFound("Label not found")
	}
	if label.ProjectID != issue.ProjectID {
		return common.BadRequest("Label does not belong to this project")
	}

	var count int64
	s.db.Model(&model.IssueLabel{}).
		Where("issue_id = ? AND label_id = ?", issueID, labelID).Count(&count)
	if count > 0 {
		return common.Conflict("Label already added to this issue")
	}

	if err := s.db.Create(&model.IssueLabel{IssueID: issueID, LabelID: labelID}).Error; err != nil {
		return common.Internal("Failed to add label")
	}

	s.createActivity(s.db, issueID, "updated", strPtr("labels"), nil, nil, nil, &actorID)
	return nil
}

// RemoveLabel removes a label from an issue.
func (s *IssueService) RemoveLabel(issueID, labelID, actorID uint64) error {
	result := s.db.Where("issue_id = ? AND label_id = ?", issueID, labelID).Delete(&model.IssueLabel{})
	if result.RowsAffected == 0 {
		return common.NotFound("Label not found on this issue")
	}

	s.createActivity(s.db, issueID, "updated", strPtr("labels"), nil, nil, nil, &actorID)
	return nil
}

// SetCycle sets the cycle for an issue.
func (s *IssueService) SetCycle(issueID, cycleID, actorID uint64) error {
	var issue model.Issue
	if err := s.db.First(&issue, issueID).Error; err != nil {
		return common.NotFound("Issue not found")
	}

	// Validate cycle exists
	var cycle model.Cycle
	if err := s.db.First(&cycle, cycleID).Error; err != nil {
		return common.NotFound("Cycle not found")
	}

	tx := s.db.Begin()
	// Remove existing cycle
	tx.Where("issue_id = ?", issueID).Delete(&model.IssueCycle{})
	// Set new cycle
	if err := tx.Create(&model.IssueCycle{IssueID: issueID, CycleID: cycleID}).Error; err != nil {
		tx.Rollback()
		return common.Internal("Failed to set cycle")
	}

	s.createActivity(tx, issueID, "updated", strPtr("cycle"), nil, nil, nil, &actorID)
	if err := tx.Commit().Error; err != nil {
		return common.Internal("Failed to commit transaction")
	}
	return nil
}

// RemoveCycle removes the cycle from an issue.
func (s *IssueService) RemoveCycle(issueID, actorID uint64) error {
	s.db.Where("issue_id = ?", issueID).Delete(&model.IssueCycle{})
	s.createActivity(s.db, issueID, "updated", strPtr("cycle"), nil, nil, nil, &actorID)
	return nil
}

// GetActivities returns issue activity history.
func (s *IssueService) GetActivities(issueID uint64, limit, offset int) ([]response.IssueActivityResponse, error) {
	var activities []model.IssueActivity
	if err := s.db.Preload("Actor").
		Where("issue_id = ?", issueID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&activities).Error; err != nil {
		return nil, common.Internal("Database error")
	}

	// Pre-load state name map for resolving state IDs to names.
	stateNameMap := make(map[uint64]string)
	for _, a := range activities {
		if a.Field != nil && *a.Field == "state" {
			if a.OldValue != nil {
				if id, err := strconv.ParseUint(*a.OldValue, 10, 64); err == nil {
					stateNameMap[id] = "" // mark for resolution
				}
			}
			if a.NewValue != nil {
				if id, err := strconv.ParseUint(*a.NewValue, 10, 64); err == nil {
					stateNameMap[id] = "" // mark for resolution
				}
			}
		}
	}
	if len(stateNameMap) > 0 {
		ids := make([]uint64, 0, len(stateNameMap))
		for id := range stateNameMap {
			ids = append(ids, id)
		}
		var states []model.State
		s.db.Where("id IN ?", ids).Find(&states)
		for _, st := range states {
			stateNameMap[st.ID] = st.Name
		}
	}

	result := make([]response.IssueActivityResponse, len(activities))
	for i, a := range activities {
		actorName := ""
		actorAvatar := ""
		if a.Actor != nil {
			actorName = a.Actor.DisplayName
			if a.Actor.Avatar != nil {
				actorAvatar = *a.Actor.Avatar
			}
		}
		oldVal := a.OldValue
		newVal := a.NewValue
		// Resolve state IDs to names for state-change activities.
		if a.Field != nil && *a.Field == "state" {
			if oldVal != nil {
				if id, err := strconv.ParseUint(*oldVal, 10, 64); err == nil {
					if name, ok := stateNameMap[id]; ok && name != "" {
						oldVal = &name
					}
				}
			}
			if newVal != nil {
				if id, err := strconv.ParseUint(*newVal, 10, 64); err == nil {
					if name, ok := stateNameMap[id]; ok && name != "" {
						newVal = &name
					}
				}
			}
		}
		result[i] = response.IssueActivityResponse{
			ID:               a.ID,
			IssueID:          a.IssueID,
			Verb:             a.Verb,
			Field:            a.Field,
			OldValue:         oldVal,
			NewValue:         newVal,
			Comment:          a.Comment,
			ActorID:          a.ActorID,
			ActorDisplayName: actorName,
			ActorAvatar:      actorAvatar,
			CreatedAt:        a.CreatedAt,
		}
	}
	return result, nil
}

// GetStatistics returns issue statistics for a project.
func (s *IssueService) GetStatistics(projectID uint64) (*response.IssueStatistics, error) {
	stats := &response.IssueStatistics{
		ByState:    make(map[string]int64),
		ByPriority: make(map[string]int64),
	}

	// Total count
	s.db.Model(&model.Issue{}).Where("project_id = ?", projectID).Count(&stats.Total)

	// Completed
	s.db.Model(&model.Issue{}).
		Joins("JOIN states ON states.id = issues.state_id").
		Where("issues.project_id = ? AND states.group = ?", projectID, common.StateGroupCompleted).
		Count(&stats.CompletedCount)

	// Drafts
	s.db.Model(&model.Issue{}).Where("project_id = ? AND is_draft = ?", projectID, true).
		Count(&stats.DraftCount)

	// By state
	var states []model.State
	s.db.Where("project_id = ?", projectID).Find(&states)
	for _, st := range states {
		var count int64
		s.db.Model(&model.Issue{}).Where("project_id = ? AND state_id = ?", projectID, st.ID).Count(&count)
		stats.ByState[st.Name] = count
	}

	// By priority
	for _, p := range []string{common.PriorityUrgent, common.PriorityHigh, common.PriorityMedium, common.PriorityLow, common.PriorityNone} {
		var count int64
		s.db.Model(&model.Issue{}).Where("project_id = ? AND priority = ?", projectID, p).Count(&count)
		stats.ByPriority[p] = count
	}

	return stats, nil
}

// Search searches issues by query across a workspace.
func (s *IssueService) Search(workspaceID uint64, query string, projectID *uint64, limit int) ([]response.IssueSearchResult, error) {
	q := s.db.Model(&model.Issue{}).
		Joins("JOIN projects ON projects.id = issues.project_id").
		Where("issues.workspace_id = ?", workspaceID).
		Where("issues.name ILIKE ? OR COALESCE(issues.description_stripped, '') ILIKE ?",
			"%"+query+"%", "%"+query+"%")

	if projectID != nil {
		q = q.Where("issues.project_id = ?", *projectID)
	}

	var issues []model.Issue
	if err := q.Preload("Project").Limit(limit).Find(&issues).Error; err != nil {
		return nil, common.Internal("Database error")
	}

	result := make([]response.IssueSearchResult, len(issues))
	for i, issue := range issues {
		result[i] = response.IssueSearchResult{
			ID:                issue.ID,
			Name:              issue.Name,
			SequenceID:        issue.SequenceID,
			ProjectIdentifier: issue.Project.Identifier,
			ProjectID:         issue.ProjectID,
		}
	}
	return result, nil
}

// Suggest returns search suggestions for issues based on query prefix.
func (s *IssueService) Suggest(projectID uint64, query string, limit int) ([]response.IssueSearchResult, error) {
	if query == "" {
		return []response.IssueSearchResult{}, nil
	}

	q := s.db.Model(&model.Issue{}).
		Preload("Project").
		Where("issues.project_id = ? AND issues.archived_at IS NULL", projectID).
		Where("issues.name ILIKE ? OR issues.sequence_id::text LIKE ?",
			"%"+query+"%", "%"+query+"%").
		Order("issues.updated_at DESC").
		Limit(limit)

	var issues []model.Issue
	if err := q.Find(&issues).Error; err != nil {
		return nil, common.Internal("Database error")
	}

	result := make([]response.IssueSearchResult, len(issues))
	for i, issue := range issues {
		result[i] = response.IssueSearchResult{
			ID:                issue.ID,
			Name:              issue.Name,
			SequenceID:        issue.SequenceID,
			ProjectIdentifier: issue.Project.Identifier,
			ProjectID:         issue.ProjectID,
		}
	}
	return result, nil
}

// BulkUpdate updates multiple issues with the same changes.
func (s *IssueService) BulkUpdate(projectID uint64, req *request.BulkUpdateRequest, userID uint64) (*response.BulkUpdateResultResponse, error) {
	if len(req.IssueIDs) == 0 {
		return &response.BulkUpdateResultResponse{}, nil
	}

	tx := s.db.Begin()

	var failedItems []response.BulkFailedItem
	var successIDs []uint64

	hasSimpleUpdates := req.Priority != nil || req.StartDate != nil || req.TargetDate != nil || req.SortOrder != nil

	// Always verify which issue IDs actually exist before applying any updates.
	// This prevents creating orphaned join-table records for non-existent issues.
	var existingIDs []uint64
	existingIDSet := make(map[uint64]bool, len(req.IssueIDs))
	{
		var foundIDs []uint64
		tx.Model(&model.Issue{}).Where("id IN ?", req.IssueIDs).Pluck("id", &foundIDs)
		for _, id := range foundIDs {
			existingIDSet[id] = true
		}
		existingIDs = foundIDs

		// Track IDs that don't exist as failures
		for _, id := range req.IssueIDs {
			if !existingIDSet[id] {
				failedItems = append(failedItems, response.BulkFailedItem{IssueID: id, Reason: "Issue not found"})
			}
		}
	}

	if hasSimpleUpdates {
		updateMap := make(map[string]interface{})
		if req.Priority != nil {
			updateMap["priority"] = *req.Priority
		}
		if req.StartDate != nil {
			if t, err := time.Parse(time.RFC3339, *req.StartDate); err == nil {
				updateMap["start_date"] = t
			}
		}
		if req.TargetDate != nil {
			if t, err := time.Parse(time.RFC3339, *req.TargetDate); err == nil {
				updateMap["target_date"] = t
			}
		}
		if req.SortOrder != nil {
			updateMap["sort_order"] = *req.SortOrder
		}
		if len(updateMap) > 0 && len(existingIDs) > 0 {
			if err := tx.Model(&model.Issue{}).Where("id IN ?", existingIDs).Updates(updateMap).Error; err != nil {
				tx.Rollback()
				return nil, common.Internal("Failed to bulk update issues")
			}
		}
	}

	if req.StateID != nil {
		var st model.State
		if err := tx.Where("id = ? AND project_id = ? AND is_active = ?", *req.StateID, projectID, true).First(&st).Error; err != nil {
			tx.Rollback()
			return nil, common.BadRequest("State does not belong to this project")
		}
		for _, issueID := range existingIDs {
			var issue model.Issue
			if err := tx.First(&issue, issueID).Error; err != nil {
				failedItems = append(failedItems, response.BulkFailedItem{IssueID: issueID, Reason: "Issue not found"})
				continue
			}
			oldStateID := issue.StateID
			newStateID := *req.StateID
			if oldStateID == newStateID {
				successIDs = append(successIDs, issueID)
				continue
			}
			if err := s.validateStateTransition(tx, projectID, issueID, oldStateID, newStateID, userID); err != nil {
				failedItems = append(failedItems, response.BulkFailedItem{IssueID: issueID, Reason: err.Error()})
				continue
			}
			issue.StateID = newStateID
			var newState model.State
			if err := tx.First(&newState, newStateID).Error; err == nil {
				if newState.Group == common.StateGroupCompleted {
					now := time.Now()
					issue.CompletedAt = &now
				} else {
					issue.CompletedAt = nil
				}
			}
			tx.Save(&issue)
			successIDs = append(successIDs, issueID)
		}
	} else {
		successIDs = existingIDs
	}

	if req.AssigneeIDs != nil && len(existingIDs) > 0 {
		tx.Where("issue_id IN ?", existingIDs).Delete(&model.IssueAssignee{})
		for _, issueID := range existingIDs {
			for _, aid := range req.AssigneeIDs {
				tx.Create(&model.IssueAssignee{IssueID: issueID, UserID: aid})
			}
		}
	}

	if req.LabelIDs != nil && len(existingIDs) > 0 {
		// Validate labels belong to this project
		if err := validateLabelsBelongToProject(tx, req.LabelIDs, projectID); err != nil {
			tx.Rollback()
			return nil, err
		}
		tx.Where("issue_id IN ?", existingIDs).Delete(&model.IssueLabel{})
		for _, issueID := range existingIDs {
			for _, lid := range req.LabelIDs {
				tx.Create(&model.IssueLabel{IssueID: issueID, LabelID: lid})
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, common.Internal("Failed to commit bulk update")
	}

	if req.StateID != nil {
		for _, issueID := range successIDs {
			s.runAutomations(issueID, "issue.state_changed", map[string]interface{}{
				"issue_id":   issueID,
				"new_state":  fmt.Sprintf("%d", *req.StateID),
				"project_id": projectID,
			})
		}
	} else if req.Priority != nil || req.AssigneeIDs != nil || req.LabelIDs != nil {
		for _, issueID := range successIDs {
			s.runAutomations(issueID, "issue.updated", map[string]interface{}{
				"issue_id":   issueID,
				"project_id": projectID,
			})
		}
	}

	var issues []model.Issue
	if err := s.db.Preload("State").Preload("IssueType").Preload("AssigneeLinks.User").Preload("LabelLinks.Label").Preload("CycleLink").Where("id IN ?", successIDs).Find(&issues).Error; err != nil {
		return nil, common.Internal("Failed to fetch updated issues")
	}

	updatedItems := make([]response.IssueResponse, 0, len(issues))
	for _, issue := range issues {
		resp, err := s.BuildIssueResponse(&issue)
		if err != nil {
			continue
		}
		updatedItems = append(updatedItems, *resp)
	}

	return &response.BulkUpdateResultResponse{
		SuccessCount: len(successIDs),
		FailedCount:  len(failedItems),
		FailedItems:  failedItems,
		UpdatedItems: updatedItems,
	}, nil
}

// BulkDelete deletes multiple issues.
func (s *IssueService) BulkDelete(issueIDs []uint64, userID uint64) (*response.BulkDeleteResultResponse, error) {
	if len(issueIDs) == 0 {
		return &response.BulkDeleteResultResponse{}, nil
	}

	var projectIDs []uint64
	s.db.Model(&model.Issue{}).Where("id IN ?", issueIDs).Pluck("DISTINCT project_id", &projectIDs)
	for _, pid := range projectIDs {
		if err := s.checkProjectMembership(pid, userID); err != nil {
			return nil, err
		}
	}

	var failedItems []response.BulkFailedItem
	var successIDs []uint64
	tx := s.db.Begin()

	for _, issueID := range issueIDs {
		var issue model.Issue
		if err := tx.First(&issue, issueID).Error; err != nil {
			failedItems = append(failedItems, response.BulkFailedItem{IssueID: issueID, Reason: "Issue not found"})
			continue
		}
		if err := tx.Delete(&issue).Error; err != nil {
			failedItems = append(failedItems, response.BulkFailedItem{IssueID: issueID, Reason: "Failed to delete issue"})
			continue
		}
		successIDs = append(successIDs, issueID)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, common.Internal("Failed to commit bulk delete")
	}

	return &response.BulkDeleteResultResponse{
		SuccessCount: len(successIDs),
		FailedCount:  len(failedItems),
		FailedItems:  failedItems,
	}, nil
}

// ==================== Private helpers ====================

func (s *IssueService) buildResponse(issueID uint64) (*response.IssueResponse, error) {
	var issue model.Issue
	if err := s.db.
		Preload("State").
		Preload("Project").
		Preload("IssueType").
		Preload("AssigneeLinks.User").
		Preload("LabelLinks.Label").
		Preload("CycleLink").
		Preload("ModuleLinks").
		Preload("Parent.State").
		Preload("Parent.IssueType").
		Preload("Parent.AssigneeLinks.User").
		Preload("SubIssues.State").
		Preload("SubIssues.IssueType").
		Preload("SubIssues.AssigneeLinks.User").
		First(&issue, issueID).Error; err != nil {
		return nil, common.NotFound("Issue not found")
	}
	return s.BuildIssueResponse(&issue)
}

func (s *IssueService) BuildIssueResponse(issue *model.Issue) (*response.IssueResponse, error) {
	resp := &response.IssueResponse{
		ID:              issue.ID,
		Name:            issue.Name,
		DescriptionHTML: issue.DescriptionHTML,
		DescriptionJSON: issue.DescriptionJSON,
		Priority:        issue.Priority,
		SequenceID:      issue.SequenceID,
		SortOrder:       issue.SortOrder,
		StartDate:       issue.StartDate,
		TargetDate:      issue.TargetDate,
		CompletedAt:     issue.CompletedAt,
		IsDraft:         issue.IsDraft,
		ArchivedAt:      issue.ArchivedAt,
		ProjectID:       issue.ProjectID,
		WorkspaceID:     issue.WorkspaceID,
		StateID:         issue.StateID,
		ParentID:        issue.ParentID,
		Depth:           issue.Depth,
		ExternalID:      issue.ExternalID,
		ExternalSource:  issue.ExternalSource,
		CoverImageURL:   issue.CoverImageURL,
		EstimatePointID: issue.EstimatePointID,
		LinkCount:       0,
		AttachmentCount: 0,
		CreatedAt:       issue.CreatedAt,
		UpdatedAt:       issue.UpdatedAt,
		CreatedByID:     issue.CreatedByID,
		UpdatedByID:     issue.UpdatedByID,
	}

	// State info
	if issue.State.ID != 0 {
		resp.StateName = issue.State.Name
		resp.StateGroup = issue.State.Group
	}

	// Project info
	if issue.Project.ID != 0 {
		resp.Project = &response.ProjectLite{
			ID:         issue.Project.ID,
			Name:       issue.Project.Name,
			Identifier: issue.Project.Identifier,
		}
	}

	// Issue Type info
	if issue.IssueType.ID != 0 {
		resp.IssueType = &response.IssueTypeLite{
			ID:                  issue.IssueType.ID,
			Name:                issue.IssueType.Name,
			Color:               issue.IssueType.Color,
			Icon:                issue.IssueType.Icon,
			Level:               issue.IssueType.Level,
			AllowedChildTypeIDs: issue.IssueType.AllowedChildTypeIDs,
		}
	}

	// Parent issue — use preloaded data, or load manually if preload failed
	if issue.ParentID != nil {
		parentIssue := issue.Parent
		if parentIssue == nil || parentIssue.ID == 0 {
			var p model.Issue
			if err := s.db.Preload("State").Preload("IssueType").Preload("AssigneeLinks.User").First(&p, *issue.ParentID).Error; err == nil {
				parentIssue = &p
			}
		}
		if parentIssue != nil && parentIssue.ID != 0 {
			parent := &response.RelatedIssueLite{
				ID:         parentIssue.ID,
				SequenceID: parentIssue.SequenceID,
				Name:       parentIssue.Name,
				StateID:    parentIssue.StateID,
				Priority:   parentIssue.Priority,
				TargetDate: parentIssue.TargetDate,
				Assignees:  []response.UserLite{},
			}
			if parentIssue.State.ID != 0 {
				parent.StateName = parentIssue.State.Name
				parent.StateGroup = parentIssue.State.Group
			}
			if parentIssue.IssueType.ID != 0 {
				parent.IssueType = &response.IssueTypeLite{
					ID:                  parentIssue.IssueType.ID,
					Name:                parentIssue.IssueType.Name,
					Color:               parentIssue.IssueType.Color,
					Icon:                parentIssue.IssueType.Icon,
					Level:               parentIssue.IssueType.Level,
					AllowedChildTypeIDs: parentIssue.IssueType.AllowedChildTypeIDs,
				}
			}
			for _, link := range parentIssue.AssigneeLinks {
				if link.User.ID != 0 {
					parent.Assignees = append(parent.Assignees, response.UserLite{
						ID:          link.User.ID,
						DisplayName: link.User.DisplayName,
						Email:       link.User.Email,
					})
				}
			}
			resp.Parent = parent
		}
	}

	// Sub-issues — use preloaded data, or load manually if preload failed
	subIssues := issue.SubIssues
	if len(subIssues) == 0 {
		s.db.Where("parent_id = ?", issue.ID).
			Preload("State").Preload("IssueType").Preload("AssigneeLinks.User").
			Find(&subIssues)
	}
	resp.SubIssues = make([]response.SubIssueLite, 0)
	for _, sub := range subIssues {
		si := response.SubIssueLite{
			ID:         sub.ID,
			SequenceID: sub.SequenceID,
			Name:       sub.Name,
			StateID:    sub.StateID,
			Priority:   sub.Priority,
			TargetDate: sub.TargetDate,
			Assignees:  []response.UserLite{},
		}
		if sub.State.ID != 0 {
			si.StateName = sub.State.Name
			si.StateGroup = sub.State.Group
		}
		if sub.IssueType.ID != 0 {
			si.IssueType = &response.IssueTypeLite{
				ID:                  sub.IssueType.ID,
				Name:                sub.IssueType.Name,
				Color:               sub.IssueType.Color,
				Icon:                sub.IssueType.Icon,
				Level:               sub.IssueType.Level,
				AllowedChildTypeIDs: sub.IssueType.AllowedChildTypeIDs,
			}
		}
		for _, link := range sub.AssigneeLinks {
			if link.User.ID != 0 {
				si.Assignees = append(si.Assignees, response.UserLite{
					ID:          link.User.ID,
					DisplayName: link.User.DisplayName,
					Email:       link.User.Email,
				})
			}
		}
		resp.SubIssues = append(resp.SubIssues, si)
	}

	// Assignees
	resp.Assignees = make([]response.UserLite, 0)
	resp.Assignees = []response.UserLite{} // ensure not null
	for _, link := range issue.AssigneeLinks {
		if link.User.ID != 0 {
			resp.Assignees = append(resp.Assignees, response.UserLite{
				ID:          link.User.ID,
				DisplayName: link.User.DisplayName,
				Email:       link.User.Email,
			})
		}
	}

	// Labels
	resp.Labels = make([]uint64, 0)
	resp.LabelDetails = make([]response.LabelLite, 0)
	for _, link := range issue.LabelLinks {
		if link.Label.ID != 0 {
			resp.Labels = append(resp.Labels, link.LabelID)
			// Note: GORM preload puts Label data in Label field
			if link.Label.Name != "" {
				resp.LabelDetails = append(resp.LabelDetails, response.LabelLite{
					ID:    link.Label.ID,
					Name:  link.Label.Name,
					Color: link.Label.Color,
				})
			}
		}
	}

	// Cycle
	if issue.CycleLink != nil {
		resp.CycleID = &issue.CycleLink.CycleID
	}

	// Modules
	resp.ModuleIDs = make([]uint64, 0)
	for _, link := range issue.ModuleLinks {
		if link.ModuleID != 0 {
			resp.ModuleIDs = append(resp.ModuleIDs, link.ModuleID)
		}
	}

	// Release
	var releaseLink model.ReleaseIssue
	if err := s.db.Where("issue_id = ?", issue.ID).First(&releaseLink).Error; err == nil {
		releaseID := releaseLink.ReleaseID
		resp.ReleaseID = &releaseID
	}

	// Sub-issues count
	resp.SubIssuesCount = int64(len(issue.SubIssues))

	var attachmentCount int64
	if err := s.db.Model(&model.Attachment{}).Where("issue_id = ?", issue.ID).Count(&attachmentCount).Error; err == nil {
		resp.AttachmentCount = int(attachmentCount)
	}

	// Soft delete
	if issue.DeletedAt.Valid {
		resp.DeletedAt = &issue.DeletedAt.Time
		resp.IsDeleted = true
	}

	return resp, nil
}

func (s *IssueService) createActivity(db *gorm.DB, issueID uint64, verb string, field, oldValue, newValue, comment *string, actorID *uint64) {
	activity := &model.IssueActivity{
		IssueID:  &issueID,
		Verb:     verb,
		Field:    field,
		OldValue: oldValue,
		NewValue: newValue,
		Comment:  comment,
		ActorID:  actorID,
	}
	db.Create(activity)
}

func strPtr(s string) *string {
	return &s
}

// ========== Workflow & Automation Engine ==========

// validateStateTransition checks if moving from oldState to newState is allowed
// by any active workflow in the project or workspace. Returns nil if allowed.
//
// Matching is by exact state ID first, then by state name (so workspace workflows
// whose transitions reference workspace/copied state IDs still apply to project
// copies with the same name). When applicable workflows define at least one
// transition and none match, the change is rejected (no soft-allow).
func (s *IssueService) validateStateTransition(db *gorm.DB, projectID, issueID, oldStateID, newStateID, userID uint64) error {
	// Get issue type ID if available for workflow filtering
	var issueTypeID *uint64
	var issue model.Issue
	if err := db.Select("issue_type_id, approval_status").First(&issue, issueID).Error; err == nil {
		if issue.IssueTypeID != nil {
			issueTypeID = issue.IssueTypeID
		}
		// Pending-approval guard: block state changes while an approval is pending
		if issue.ApprovalStatus != nil && *issue.ApprovalStatus == "pending" {
			return common.BadRequest("该工作项正在审批流程中，请等待审批完成后再进行状态变更")
		}
	}

	// Resolve the transition rules from the shared workflow engine. A failure to
	// load them is treated as "unconstrained" so a database hiccup cannot block
	// every state change in the product; the AI tools, which only advise, surface
	// the error instead of guessing.
	rules, err := workflow.Load(db, projectID, issueTypeID)
	if err != nil {
		return nil
	}
	if !rules.Restricts {
		return nil // no workflow constrains this project yet = allow all transitions
	}

	var oldState, newState model.State
	db.First(&oldState, oldStateID)
	db.First(&newState, newStateID)

	verdict := rules.Check(oldState, newState)
	switch verdict.Outcome {
	case workflow.Allowed:
		return nil
	case workflow.ApprovalRequired:
		return common.NewApprovalRequiredError(verdict.TransitionID, verdict.WorkflowID, verdict.WorkflowName, oldState.Name, newState.Name, oldStateID, newStateID)
	}
	return common.BadRequest(fmt.Sprintf("状态转换「%s → %s」不被当前工作流允许", oldState.Name, newState.Name))
}

// addProjectLeadAndSubscribers adds project lead and project subscribers into recipientIDs.
func (s *IssueService) addProjectLeadAndSubscribers(projectID, excludeUserID uint64, recipientIDs map[uint64]bool) {
	var project model.Project
	if err := s.db.Select("default_assignee_id, project_lead_id").First(&project, projectID).Error; err == nil {
		if project.ProjectLeadID != nil && *project.ProjectLeadID != excludeUserID {
			recipientIDs[*project.ProjectLeadID] = true
		}
	}
	var subs []model.ProjectSubscriber
	s.db.Where("project_id = ?", projectID).Find(&subs)
	for _, sub := range subs {
		if sub.UserID != excludeUserID {
			recipientIDs[sub.UserID] = true
		}
	}
}

// runAutomations executes automation rules for a given trigger type on an issue.
// Called after transaction commit — uses the automation service's event bus.
func (s *IssueService) runAutomations(issueID uint64, triggerType string, context map[string]interface{}) {
	if s.automationSvc == nil {
		return
	}

	var issue model.Issue
	if err := s.db.Preload("AssigneeLinks").First(&issue, issueID).Error; err != nil {
		log.Printf("[IssueService] Failed to fetch issue %d for automation: %v", issueID, err)
		return
	}

	if context == nil {
		context = make(map[string]interface{})
	}

	var state model.State
	if err := s.db.First(&state, issue.StateID).Error; err == nil {
		context["state"] = state.Name
	}

	context["issue_id"] = issueID
	context["project_id"] = issue.ProjectID
	context["state_id"] = issue.StateID
	if issue.ParentID != nil {
		context["parent_id"] = *issue.ParentID
	}
	context["priority"] = issue.Priority
	context["title"] = issue.Name
	context["description"] = issue.DescriptionHTML
	context["due_date"] = issue.TargetDate
	context["start_date"] = issue.StartDate

	var assigneeIDs []uint64
	for _, a := range issue.AssigneeLinks {
		assigneeIDs = append(assigneeIDs, a.UserID)
	}
	context["assignee"] = assigneeIDs

	var labelIDs []uint64
	if err := s.db.Model(&model.IssueLabel{}).Where("issue_id = ?", issueID).Pluck("label_id", &labelIDs).Error; err == nil {
		context["labels"] = labelIDs
	}

	var cycleID uint64
	if err := s.db.Model(&model.IssueCycle{}).Where("issue_id = ?", issueID).Pluck("cycle_id", &cycleID).Error; err == nil && cycleID > 0 {
		var cycle model.Cycle
		if err := s.db.First(&cycle, cycleID).Error; err == nil {
			context["cycle"] = cycle.Name
		} else {
			context["cycle"] = cycleID
		}
		context["cycle_id"] = cycleID
	}

	event := Event{
		Type:      triggerType,
		ProjectID: issue.ProjectID,
		IssueID:   issueID,
		Context:   context,
		Timestamp: time.Now(),
	}

	if err := s.automationSvc.PublishEvent(event); err != nil {
		log.Printf("[IssueService] Failed to publish automation event: %v", err)
	}
}

// maxImportRows caps a single bulk import so one request cannot pin the
// process, and so a truncated file is rejected instead of half-applied.
const maxImportRows = 5000

// importFieldAliases lists the CSV header spellings accepted for each
// canonical import field, covering ReqMango, Jira, Linear and Plane exports.
var importFieldAliases = map[string][]string{
	"name":         {"name", "标题", "summary", "title"},
	"description":  {"description", "描述", "详情"},
	"priority":     {"priority", "优先级"},
	"state":        {"state", "状态", "status"},
	"type":         {"type", "类型", "issue type", "类型名称"},
	"assignees":    {"assignees", "assignee", "负责人", "经办人"},
	"labels":       {"labels", "label", "标签"},
	"start_date":   {"start_date", "start date", "开始日期"},
	"target_date":  {"target_date", "target date", "due date", "截止日期", "结束日期"},
	"parent_title": {"parent_title", "parent", "父标题"},
	"module":       {"module", "模块"},
	"cycle":        {"cycle", "周期"},
	"estimate":     {"estimate", "估点", "story points"},
}

// ImportFields lists the canonical fields the CSV importer understands, in the
// order the mapping UI presents them.
var ImportFields = []string{
	"name", "description", "priority", "state", "type", "assignees", "labels",
	"start_date", "target_date", "parent_title", "module", "cycle", "estimate",
}

// ImportFromJSON imports issues from a JSON array.
func (s *IssueService) ImportFromJSON(projectID, workspaceID, userID uint64, items []request.ImportIssueItem, opts request.ImportOptions) (*response.ImportResult, error) {
	return s.importItems(projectID, workspaceID, userID, items, opts)
}

// importItems validates every row, then creates the valid ones unless
// opts.DryRun is set. Validation and creation share this single pass so a
// preview cannot disagree with the real import.
func (s *IssueService) importItems(projectID, workspaceID, userID uint64, items []request.ImportIssueItem, opts request.ImportOptions) (*response.ImportResult, error) {
	if len(items) > maxImportRows {
		return nil, common.BadRequest(fmt.Sprintf("单次导入最多 %d 行，当前 %d 行", maxImportRows, len(items)))
	}

	var project model.Project
	if err := s.db.Select("id", "workspace_id").First(&project, projectID).Error; err != nil {
		return nil, common.NotFound("Project not found")
	}
	if project.WorkspaceID != workspaceID {
		return nil, common.BadRequest("项目不属于所选工作空间")
	}

	result := &response.ImportResult{
		DryRun:      opts.DryRun,
		TotalRows:   len(items),
		Errors:      []response.ImportError{},
		Warnings:    []response.ImportWarning{},
		ImportedIDs: []uint64{},
	}

	stateMap, err := s.loadStateMap(projectID)
	if err != nil {
		return nil, err
	}
	var defaultImportState model.State
	if err := s.db.Where("project_id = ? AND is_default = ?", projectID, true).First(&defaultImportState).Error; err != nil {
		return nil, common.NotFound("No default state found for project")
	}
	importDefaultStateID := defaultImportState.ID
	typeMap, err := s.loadTypeMap(projectID)
	if err != nil {
		return nil, err
	}
	labelMap, err := s.loadLabelMap(projectID)
	if err != nil {
		return nil, err
	}
	userMap, err := s.loadUserMapByEmail(workspaceID)
	if err != nil {
		return nil, err
	}

	// Create() rejects assignees who are not on the project and types that were
	// never imported into it. Resolve both up front so the preview reports the
	// same outcome as the real import instead of failing later.
	var projectMemberIDs []uint64
	if err := s.db.Model(&model.ProjectMember{}).
		Where("project_id = ? AND is_active = ?", projectID, true).
		Pluck("user_id", &projectMemberIDs).Error; err != nil {
		return nil, err
	}
	projectMembers := make(map[uint64]bool, len(projectMemberIDs))
	for _, id := range projectMemberIDs {
		projectMembers[id] = true
	}
	visibleTypes, err := issuetype.Visible(s.db, workspaceID, &projectID)
	if err != nil {
		return nil, err
	}
	visibleTypeIDs := make(map[uint64]bool, len(visibleTypes))
	for _, vt := range visibleTypes {
		visibleTypeIDs[vt.ID] = true
	}

	// parent_title can only point at a row of this import, so collect the
	// titles up front and flag references that cannot be satisfied.
	titles := make(map[string]bool, len(items))
	for _, item := range items {
		if name := strings.TrimSpace(item.Name); name != "" {
			titles[name] = true
		}
	}

	type resolvedRow struct {
		row         int
		item        request.ImportIssueItem
		stateID     uint64
		typeID      uint64
		labelIDs    []uint64
		assigneeIDs []uint64
	}

	warn := func(row int, field, message string) {
		result.Warnings = append(result.Warnings, response.ImportWarning{Row: row, Field: field, Message: message})
	}

	rows := make([]resolvedRow, 0, len(items))
	for i, item := range items {
		row := i + 1
		if strings.TrimSpace(item.Name) == "" {
			result.FailCount++
			result.Errors = append(result.Errors, response.ImportError{
				Row:     row,
				Title:   item.Name,
				Message: "标题不能为空",
			})
			continue
		}

		stateID := uint64(0)
		if item.StateName != "" {
			if id, ok := stateMap[item.StateName]; ok {
				stateID = id
			} else {
				warn(row, "state", fmt.Sprintf("未找到状态 %q，将使用项目默认状态", item.StateName))
			}
		}

		typeID := uint64(0)
		if item.TypeName != "" {
			if id, ok := typeMap[item.TypeName]; ok {
				if visibleTypeIDs[id] {
					typeID = id
				} else {
					warn(row, "type", fmt.Sprintf("类型 %q 尚未导入该项目，该行将不带类型", item.TypeName))
				}
			} else {
				warn(row, "type", fmt.Sprintf("未找到类型 %q，该行将不带类型", item.TypeName))
			}
		}

		seenLabel := map[uint64]bool{}
		labelIDs := []uint64{}
		for _, ln := range item.LabelNames {
			if id, ok := labelMap[ln]; ok {
				if !seenLabel[id] {
					seenLabel[id] = true
					labelIDs = append(labelIDs, id)
				}
			} else {
				warn(row, "label", fmt.Sprintf("未找到标签 %q，已忽略", ln))
			}
		}

		seenAssignee := map[uint64]bool{}
		assigneeIDs := []uint64{}
		for _, email := range item.AssigneeEmails {
			if id, ok := userMap[strings.ToLower(email)]; ok {
				if !projectMembers[id] {
					warn(row, "assignee", fmt.Sprintf("负责人 %q 不是该项目成员，已忽略", email))
					continue
				}
				if !seenAssignee[id] {
					seenAssignee[id] = true
					assigneeIDs = append(assigneeIDs, id)
				}
			} else {
				warn(row, "assignee", fmt.Sprintf("负责人 %q 不是该工作空间成员，已忽略", email))
			}
		}

		if item.ParentTitle != "" && !titles[item.ParentTitle] {
			warn(row, "parent", fmt.Sprintf("父标题 %q 不在本次导入中，该行将作为顶层工作项", item.ParentTitle))
		}

		// Mandatory custom fields: same rule as Create (including UI default type when type is omitted).
		effectiveTypeID := typeID
		if effectiveTypeID == 0 {
			defaultTypeID, resolveErr := s.resolveProjectDefaultIssueTypeID(projectID, workspaceID)
			if resolveErr != nil {
				result.FailCount++
				msg := resolveErr.Error()
				var appErr *common.AppError
				if errors.As(resolveErr, &appErr) && appErr.Message != "" {
					msg = appErr.Message
				}
				result.Errors = append(result.Errors, response.ImportError{
					Row: row, Title: item.Name, Message: msg,
				})
				continue
			}
			if defaultTypeID != nil {
				effectiveTypeID = *defaultTypeID
			}
		}
		if effectiveTypeID > 0 {
			cfValues := importCustomFieldValues(item)
			visibilityStateID := stateID
			if visibilityStateID == 0 {
				visibilityStateID = importDefaultStateID
			}
			priority := strings.TrimSpace(item.Priority)
			if priority == "" {
				priority = common.PriorityNone
			}
			typePtr := effectiveTypeID
			if err := s.validateMandatoryCustomFields(workspaceID, &typePtr, cfValues, fieldVisibilityContext{
				IssueTypeID: effectiveTypeID,
				StateID:     visibilityStateID,
				Priority:    priority,
				AssigneeIDs: assigneeIDs,
				LabelIDs:    labelIDs,
			}); err != nil {
				result.FailCount++
				msg := err.Error()
				var appErr *common.AppError
				if errors.As(err, &appErr) && appErr.Message != "" {
					msg = appErr.Message
				}
				result.Errors = append(result.Errors, response.ImportError{
					Row:     row,
					Title:   item.Name,
					Message: msg,
				})
				continue
			}
		}

		rows = append(rows, resolvedRow{
			row:         row,
			item:        item,
			stateID:     stateID,
			typeID:      typeID,
			labelIDs:    labelIDs,
			assigneeIDs: assigneeIDs,
		})
	}

	if opts.DryRun {
		// Nothing is written; every row that passed validation would be created.
		result.SuccessCount = len(rows)
		return result, nil
	}

	titleToID := make(map[string]uint64, len(rows))
	create := func(r resolvedRow, parentID *uint64, originalRow int) {
		item := r.item
		createReq := &request.IssueCreateRequest{
			Name:            item.Name,
			DescriptionHTML: "<p>" + item.Description + "</p>",
			Priority:        item.Priority,
			LabelIDs:        r.labelIDs,
			AssigneeIDs:     r.assigneeIDs,
			ParentID:        parentID,
		}
		if r.stateID > 0 {
			stateID := r.stateID
			createReq.StateID = &stateID
		}
		if r.typeID > 0 {
			typeID := r.typeID
			createReq.TypeID = &typeID
		}
		if item.StartDate != "" {
			createReq.StartDate = &item.StartDate
		}
		if item.TargetDate != "" {
			createReq.TargetDate = &item.TargetDate
		}
		if item.EstimatePoints != nil {
			if pid := s.lookupEstimatePointID(projectID, *item.EstimatePoints); pid != nil {
				createReq.EstimatePointID = pid
			}
		}

		resp, svcErr := s.Create(createReq, projectID, workspaceID, userID)
		if svcErr != nil {
			result.FailCount++
			result.Errors = append(result.Errors, response.ImportError{
				Row:     originalRow,
				Title:   item.Name,
				Message: fmt.Sprintf("创建失败: %v", svcErr),
			})
			return
		}
		result.SuccessCount++
		result.ImportedIDs = append(result.ImportedIDs, resp.ID)
		titleToID[item.Name] = resp.ID
	}

	pending := make([]resolvedRow, 0)
	for _, r := range rows {
		if r.item.ParentTitle != "" {
			if id, ok := titleToID[r.item.ParentTitle]; ok {
				parentID := id
				create(r, &parentID, r.row)
				continue
			}
			pending = append(pending, r)
			continue
		}
		create(r, nil, r.row)
	}

	// Second pass: rows whose parent appears later in the file.
	for _, r := range pending {
		var parentID *uint64
		if id, ok := titleToID[r.item.ParentTitle]; ok {
			parentID = &id
		}
		create(r, parentID, r.row)
	}

	return result, nil
}

// parseCSVRecords turns CSV rows into import items. mapping wins over the
// built-in aliases; unresolved mapping targets are a request error so the UI
// cannot silently import into the wrong column.
func parseCSVRecords(records [][]string, mapping map[string]string) (headers []string, applied map[string]string, items []request.ImportIssueItem, err error) {
	if len(records) == 0 {
		return nil, nil, nil, common.BadRequest("CSV 内容为空")
	}

	headers = make([]string, len(records[0]))
	headerIdx := make(map[string]int, len(records[0]))
	for i, h := range records[0] {
		trimmed := strings.TrimSpace(h)
		headers[i] = trimmed
		headerIdx[normalizeCSVHeader(trimmed)] = i
	}

	column := make(map[string]int)
	applied = make(map[string]string)
	// An explicit mapping replaces alias detection entirely: fields the user
	// left unmapped stay unmapped instead of silently picking up a column with
	// a familiar name.
	explicit := len(mapping) > 0
	for field, source := range mapping {
		source = strings.TrimSpace(source)
		if source == "" {
			continue
		}
		idx, ok := headerIdx[normalizeCSVHeader(source)]
		if !ok {
			return nil, nil, nil, common.BadRequest(fmt.Sprintf("映射的列不存在: %s", source))
		}
		column[field] = idx
		applied[field] = headers[idx]
	}
	if !explicit {
		for field, aliases := range importFieldAliases {
			if _, ok := column[field]; ok {
				continue
			}
			for _, alias := range aliases {
				if idx, ok := headerIdx[normalizeCSVHeader(alias)]; ok {
					column[field] = idx
					applied[field] = headers[idx]
					break
				}
			}
		}
	}
	if _, ok := column["name"]; !ok {
		return nil, nil, nil, common.BadRequest("CSV 缺少标题列（name / 标题 / summary / title）")
	}

	cell := func(record []string, field string) string {
		idx, ok := column[field]
		if !ok || idx >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[idx])
	}

	items = make([]request.ImportIssueItem, 0, len(records)-1)
	for row := 1; row < len(records); row++ {
		record := records[row]
		if csvRecordEmpty(record) {
			continue
		}
		item := request.ImportIssueItem{
			Name:        cell(record, "name"),
			Description: cell(record, "description"),
			Priority:    cell(record, "priority"),
			StateName:   cell(record, "state"),
			TypeName:    cell(record, "type"),
			StartDate:   cell(record, "start_date"),
			TargetDate:  cell(record, "target_date"),
			ParentTitle: cell(record, "parent_title"),
			ModuleName:  cell(record, "module"),
			CycleName:   cell(record, "cycle"),
		}
		if v := cell(record, "assignees"); v != "" {
			item.AssigneeEmails = splitAndTrim(v, ",")
		}
		if v := cell(record, "labels"); v != "" {
			item.LabelNames = splitAndTrim(v, ",")
		}
		if v := cell(record, "estimate"); v != "" {
			if points, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
				item.EstimatePoints = &points
			}
		}
		items = append(items, item)
	}
	return headers, applied, items, nil
}

func normalizeCSVHeader(h string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(h), " ", ""))
}

func csvRecordEmpty(record []string) bool {
	for _, c := range record {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func stripBOM(r io.Reader) io.Reader {
	return &bomStripper{Reader: r, done: false}
}

type bomStripper struct {
	io.Reader
	done bool
}

func (bs *bomStripper) Read(p []byte) (int, error) {
	if !bs.done {
		bs.done = true
		buf := make([]byte, len(p)+3)
		n, err := bs.Reader.Read(buf)
		if err != nil {
			return n, err
		}
		if n >= 3 && buf[0] == 0xef && buf[1] == 0xbb && buf[2] == 0xbf {
			copy(p, buf[3:n])
			return n - 3, nil
		}
		copy(p, buf[:n])
		return n, nil
	}
	return bs.Reader.Read(p)
}

// ImportFromCSV imports issues from CSV content.
func (s *IssueService) ImportFromCSV(projectID, workspaceID, userID uint64, csvContent io.Reader, opts request.ImportOptions) (*response.ImportResult, error) {
	reader := csv.NewReader(stripBOM(csvContent))
	// Ragged rows are common in hand-edited exports; pad instead of failing.
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, common.BadRequest("CSV 解析失败: " + err.Error())
	}
	if len(records) < 2 {
		return nil, common.BadRequest("CSV 至少需要包含表头和一行数据")
	}

	headers, applied, items, parseErr := parseCSVRecords(records, opts.Mapping)
	if parseErr != nil {
		return nil, parseErr
	}

	result, err := s.importItems(projectID, workspaceID, userID, items, opts)
	if err != nil {
		return nil, err
	}
	result.Headers = headers
	result.Mapping = applied
	return result, nil
}

func (s *IssueService) loadStateMap(projectID uint64) (map[string]uint64, error) {
	var states []model.State
	if err := s.db.Where("project_id = ?", projectID).Find(&states).Error; err != nil {
		return nil, err
	}
	m := make(map[string]uint64)
	for _, st := range states {
		m[st.Name] = st.ID
	}
	return m, nil
}

func (s *IssueService) loadTypeMap(projectID uint64) (map[string]uint64, error) {
	var types []model.IssueType
	if err := s.db.Where("project_id = ?", projectID).Or("workspace_id = (SELECT workspace_id FROM projects WHERE id = ?)", projectID).Find(&types).Error; err != nil {
		return nil, err
	}
	m := make(map[string]uint64)
	for _, t := range types {
		m[t.Name] = t.ID
	}
	return m, nil
}

func (s *IssueService) loadLabelMap(projectID uint64) (map[string]uint64, error) {
	var labels []model.Label
	if err := s.db.Where("project_id = ?", projectID).Find(&labels).Error; err != nil {
		return nil, err
	}
	m := make(map[string]uint64)
	for _, l := range labels {
		m[l.Name] = l.ID
	}
	return m, nil
}

func (s *IssueService) loadUserMapByEmail(workspaceID uint64) (map[string]uint64, error) {
	type workspaceMember struct {
		UserID uint64
		Email  string
	}
	var members []workspaceMember
	err := s.db.Table("workspace_members").
		Select("workspace_members.user_id, users.email").
		Joins("JOIN users ON users.id = workspace_members.user_id").
		Where("workspace_members.workspace_id = ?", workspaceID).
		Scan(&members).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]uint64)
	for _, m2 := range members {
		m[strings.ToLower(m2.Email)] = m2.UserID
	}
	return m, nil
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := []string{}
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ==================== Flow Metrics & Export ====================

// FlowMetrics contains aggregated issue counts by state.
type FlowMetrics struct {
	StateCounts    map[string]int `json:"state_counts"`
	PriorityCounts map[string]int `json:"priority_counts"`
	TypeCounts     map[string]int `json:"type_counts"`
	Total          int            `json:"total"`
}

// GetFlowMetrics returns flow metrics for a project.
func (s *IssueService) GetFlowMetrics(projectID uint64) (*FlowMetrics, error) {
	metrics := &FlowMetrics{
		StateCounts:    make(map[string]int),
		PriorityCounts: make(map[string]int),
		TypeCounts:     make(map[string]int),
	}

	var issues []model.Issue
	if err := s.db.Where("project_id = ?", projectID).Preload("State").Preload("IssueType").Find(&issues).Error; err != nil {
		return nil, err
	}

	metrics.Total = len(issues)
	for _, issue := range issues {
		stateName := "未知"
		if issue.State.Name != "" {
			stateName = issue.State.Name
		}
		metrics.StateCounts[stateName]++

		priority := "none"
		if issue.Priority != "" {
			priority = issue.Priority
		}
		metrics.PriorityCounts[priority]++

		typeName := "未知"
		if issue.IssueType.Name != "" {
			typeName = issue.IssueType.Name
		}
		metrics.TypeCounts[typeName]++
	}

	return metrics, nil
}

// ExportIssueItem is a simplified issue struct for CSV/JSON export.
type ExportIssueItem struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Priority      string   `json:"priority"`
	StateName     string   `json:"state_name"`
	IssueTypeName string   `json:"issue_type_name"`
	AssigneeNames []string `json:"assignee_names"`
	LabelNames    []string `json:"label_names"`
	StartDate     string   `json:"start_date"`
	TargetDate    string   `json:"target_date"`
	ParentName    string   `json:"parent_name"`
}

// ExportIssues returns all issues for a project in export-friendly format.
func (s *IssueService) ExportIssues(projectID uint64) ([]ExportIssueItem, error) {
	var issues []model.Issue
	if err := s.db.Where("project_id = ?", projectID).
		Preload("State").Preload("IssueType").Preload("AssigneeLinks.User").
		Preload("LabelLinks.Label").Preload("Parent").
		Order("issues.sequence_id ASC").
		Find(&issues).Error; err != nil {
		return nil, err
	}

	result := make([]ExportIssueItem, 0, len(issues))
	for _, issue := range issues {
		item := ExportIssueItem{
			Name:       issue.Name,
			Priority:   issue.Priority,
			StartDate:  "",
			TargetDate: "",
		}
		if issue.DescriptionStripped != nil {
			item.Description = *issue.DescriptionStripped
		}
		if issue.StartDate != nil {
			item.StartDate = issue.StartDate.Format("2006-01-02")
		}
		if issue.TargetDate != nil {
			item.TargetDate = issue.TargetDate.Format("2006-01-02")
		}
		if issue.State.Name != "" {
			item.StateName = issue.State.Name
		}
		if issue.IssueType.Name != "" {
			item.IssueTypeName = issue.IssueType.Name
		}
		for _, link := range issue.AssigneeLinks {
			item.AssigneeNames = append(item.AssigneeNames, link.User.DisplayName)
		}
		for _, link := range issue.LabelLinks {
			item.LabelNames = append(item.LabelNames, link.Label.Name)
		}
		if issue.Parent != nil {
			item.ParentName = issue.Parent.Name
		}
		result = append(result, item)
	}
	return result, nil
}

// importCustomFieldValues maps parsed import row data to custom field IDs.
// CSV/JSON import does not carry custom fields yet; callers pass the result
// (nil / empty) into validateMandatoryCustomFields alongside the resolved type.
func importCustomFieldValues(item request.ImportIssueItem) map[uint64]interface{} {
	_ = item
	return nil
}

// fieldVisibilityContext carries issue context for conditional-field visibility
// when enforcing mandatory custom fields (write paths only).
type fieldVisibilityContext struct {
	IssueTypeID uint64
	StateID     uint64
	Priority    string
	AssigneeIDs []uint64
	LabelIDs    []uint64
}

func (s *IssueService) validateMandatoryCustomFields(workspaceID uint64, issueTypeID *uint64, cfValues map[uint64]interface{}, ctx fieldVisibilityContext) error {
	if issueTypeID == nil || *issueTypeID == 0 {
		return nil
	}
	typeID := *issueTypeID

	var requiredFields []struct {
		FieldID uint64
		Name    string
	}

	err := s.db.Table("issue_type_fields").
		Select("issue_type_fields.field_id, custom_fields.name").
		Joins("JOIN custom_fields ON custom_fields.id = issue_type_fields.field_id").
		Where("issue_type_fields.type_id = ? AND issue_type_fields.is_required = ?", typeID, true).
		Scan(&requiredFields).Error

	if err != nil {
		return common.Internal("Failed to fetch mandatory custom fields")
	}

	// Required custom fields must be present whenever the type defines them.
	// Treat a nil map the same as empty so quick-create cannot bypass required fields.
	if cfValues == nil {
		cfValues = map[uint64]interface{}{}
	}

	typePtr := &ctx.IssueTypeID
	if ctx.IssueTypeID == 0 {
		typePtr = nil
	}
	condSvc := NewConditionalFieldService(s.db)
	visibleFieldIDs, err := condSvc.EvaluateFieldVisibility(workspaceID, typePtr, ctx.StateID, ctx.Priority, ctx.AssigneeIDs, ctx.LabelIDs)
	if err != nil {
		return err
	}
	visibleSet := make(map[uint64]struct{}, len(visibleFieldIDs))
	for _, id := range visibleFieldIDs {
		visibleSet[id] = struct{}{}
	}

	var missingFields []string
	for _, rf := range requiredFields {
		if _, ok := visibleSet[rf.FieldID]; !ok {
			continue
		}
		if val, exists := cfValues[rf.FieldID]; !exists || val == nil || val == "" {
			missingFields = append(missingFields, rf.Name)
		}
	}

	if len(missingFields) > 0 {
		return common.BadRequest("缺少必填字段: " + strings.Join(missingFields, ", "))
	}

	return nil
}

func (s *IssueService) AddPage(issueID, pageID, actorID uint64) error {
	var issue model.Issue
	if err := s.db.First(&issue, issueID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.NotFound("Issue not found")
		}
		return common.Internal("Failed to fetch issue")
	}

	var page model.Page
	if err := s.db.First(&page, pageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.NotFound("Page not found")
		}
		return common.Internal("Failed to fetch page")
	}

	if page.ProjectID != issue.ProjectID {
		return common.Forbidden("Page is not in the same project as the issue")
	}

	// Idempotent: composite PK on issue_pages yields ON CONFLICT DO NOTHING (PostgreSQL)
	// and INSERT OR IGNORE semantics on SQLite with AutoMigrate unique constraint.
	res := s.db.Exec(
		"INSERT INTO issue_pages (issue_id, page_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
		issueID, pageID,
	)
	if res.Error != nil {
		return common.Internal("Failed to add page to issue")
	}
	return nil
}

func (s *IssueService) RemovePage(issueID, pageID, actorID uint64) error {
	var issue model.Issue
	if err := s.db.First(&issue, issueID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.NotFound("Issue not found")
		}
		return common.Internal("Failed to fetch issue")
	}

	var page model.Page
	if err := s.db.First(&page, pageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.NotFound("Page not found")
		}
		return common.Internal("Failed to fetch page")
	}

	if page.ProjectID != issue.ProjectID {
		return common.Forbidden("Page is not in the same project as the issue")
	}

	res := s.db.Exec("DELETE FROM issue_pages WHERE issue_id = ? AND page_id = ?", issueID, pageID)
	if res.Error != nil {
		return common.Internal("Failed to remove page from issue")
	}
	if res.RowsAffected == 0 {
		return common.NotFound("Link not found")
	}
	return nil
}

func (s *IssueService) ListPages(issueID uint64) ([]model.Page, error) {
	var issue model.Issue
	if err := s.db.First(&issue, issueID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, common.NotFound("Issue not found")
		}
		return nil, common.Internal("Failed to fetch issue")
	}

	var pages []model.Page
	if err := s.db.
		Table("pages").
		Joins("INNER JOIN issue_pages ON issue_pages.page_id = pages.id").
		Where("issue_pages.issue_id = ? AND pages.project_id = ?", issueID, issue.ProjectID).
		Order("pages.title ASC").
		Find(&pages).Error; err != nil {
		return nil, common.Internal("Failed to list pages for issue")
	}
	return pages, nil
}

func (s *IssueService) ReorderSubIssues(parentID uint64, issueIDs []uint64) error {
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		return common.Internal("Failed to begin transaction")
	}

	for i, issueID := range issueIDs {
		sortOrder := float64(i)
		if err := tx.Model(&model.Issue{}).
			Where("id = ? AND parent_id = ?", issueID, parentID).
			Update("sort_order", sortOrder).Error; err != nil {
			tx.Rollback()
			return common.Internal("Failed to update sort order")
		}
	}

	if err := tx.Commit().Error; err != nil {
		return common.Internal("Failed to commit transaction")
	}

	return nil
}

// BulkCopy copies issues to another project, creating new copies while preserving the originals.
func (s *IssueService) BulkCopy(req *request.BulkCopyRequest, userID uint64) (*response.BulkCopyMoveResultResponse, error) {
	var targetProject model.Project
	if err := s.db.First(&targetProject, req.TargetProjectID).Error; err != nil {
		return nil, common.NotFound("Target project not found")
	}

	var issues []model.Issue
	if err := s.db.Preload("AssigneeLinks").Preload("LabelLinks").Preload("IssueType").
		Where("id IN ?", req.IssueIDs).Find(&issues).Error; err != nil {
		return nil, common.Internal("Failed to fetch issues")
	}

	if len(issues) == 0 {
		return nil, common.BadRequest("No issues found")
	}

	var failedItems []response.BulkFailedItem
	var results []response.IssueResponse
	tx := s.db.Begin()

	for _, issue := range issues {
		copiedIssue := &model.Issue{
			Name:            issue.Name,
			DescriptionHTML: issue.DescriptionHTML,
			DescriptionJSON: issue.DescriptionJSON,
			Priority:        issue.Priority,
			SequenceID:      issue.SequenceID,
			SortOrder:       issue.SortOrder,
			IsDraft:         issue.IsDraft,
			ProjectID:       req.TargetProjectID,
			WorkspaceID:     targetProject.WorkspaceID,
			StateID:         issue.StateID,
			Depth:           issue.Depth,
			ExternalID:      issue.ExternalID,
			ExternalSource:  issue.ExternalSource,
			IssueTypeID:     issue.IssueTypeID,
		}

		if issue.StartDate != nil {
			copiedIssue.StartDate = issue.StartDate
		}
		if issue.TargetDate != nil {
			copiedIssue.TargetDate = issue.TargetDate
		}

		if err := tx.Create(copiedIssue).Error; err != nil {
			failedItems = append(failedItems, response.BulkFailedItem{IssueID: issue.ID, Reason: "Failed to copy issue"})
			continue
		}

		for _, assignee := range issue.AssigneeLinks {
			tx.Create(&model.IssueAssignee{IssueID: copiedIssue.ID, UserID: assignee.UserID})
		}

		// Only copy labels belonging to the target project
		if len(issue.LabelLinks) > 0 {
			ids := make([]uint64, len(issue.LabelLinks))
			for i, l := range issue.LabelLinks {
				ids[i] = l.LabelID
			}
			var validIDs []uint64
			tx.Model(&model.Label{}).Where("id IN ? AND project_id = ?", ids, req.TargetProjectID).Pluck("id", &validIDs)
			validSet := make(map[uint64]bool, len(validIDs))
			for _, id := range validIDs {
				validSet[id] = true
			}
			for _, label := range issue.LabelLinks {
				if validSet[label.LabelID] {
					tx.Create(&model.IssueLabel{IssueID: copiedIssue.ID, LabelID: label.LabelID})
				}
			}
		}

		if req.IncludeSubtasks {
			s.copySubtasks(tx, issue.ID, copiedIssue.ID, req.TargetProjectID, targetProject.WorkspaceID)
		}

		resp, _ := s.buildResponse(copiedIssue.ID)
		if resp != nil {
			results = append(results, *resp)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, common.Internal("Failed to commit transaction")
	}

	return &response.BulkCopyMoveResultResponse{
		SuccessCount: len(results),
		FailedCount:  len(failedItems),
		FailedItems:  failedItems,
		Results:      results,
	}, nil
}

// copySubtasks recursively copies subtasks to the target issue.
func (s *IssueService) copySubtasks(tx *gorm.DB, sourceParentID, targetParentID, targetProjectID, workspaceID uint64) {
	var subtasks []model.Issue
	s.db.Preload("AssigneeLinks").Preload("LabelLinks").
		Where("parent_id = ?", sourceParentID).Find(&subtasks)

	for _, subtask := range subtasks {
		copiedSubtask := &model.Issue{
			Name:            subtask.Name,
			DescriptionHTML: subtask.DescriptionHTML,
			DescriptionJSON: subtask.DescriptionJSON,
			Priority:        subtask.Priority,
			SequenceID:      subtask.SequenceID,
			SortOrder:       subtask.SortOrder,
			IsDraft:         subtask.IsDraft,
			ProjectID:       targetProjectID,
			WorkspaceID:     workspaceID,
			StateID:         subtask.StateID,
			ParentID:        &targetParentID,
			Depth:           subtask.Depth,
			ExternalID:      subtask.ExternalID,
			ExternalSource:  subtask.ExternalSource,
			IssueTypeID:     subtask.IssueTypeID,
		}

		if subtask.StartDate != nil {
			copiedSubtask.StartDate = subtask.StartDate
		}
		if subtask.TargetDate != nil {
			copiedSubtask.TargetDate = subtask.TargetDate
		}

		tx.Create(copiedSubtask)

		for _, assignee := range subtask.AssigneeLinks {
			tx.Create(&model.IssueAssignee{IssueID: copiedSubtask.ID, UserID: assignee.UserID})
		}

		// Only copy labels belonging to the target project
		if len(subtask.LabelLinks) > 0 {
			ids := make([]uint64, len(subtask.LabelLinks))
			for i, l := range subtask.LabelLinks {
				ids[i] = l.LabelID
			}
			var validIDs []uint64
			tx.Model(&model.Label{}).Where("id IN ? AND project_id = ?", ids, targetProjectID).Pluck("id", &validIDs)
			validSet := make(map[uint64]bool, len(validIDs))
			for _, id := range validIDs {
				validSet[id] = true
			}
			for _, label := range subtask.LabelLinks {
				if validSet[label.LabelID] {
					tx.Create(&model.IssueLabel{IssueID: copiedSubtask.ID, LabelID: label.LabelID})
				}
			}
		}

		s.copySubtasks(tx, subtask.ID, copiedSubtask.ID, targetProjectID, workspaceID)
	}
}

// BulkMove moves issues to another project, removing them from the source project.
func (s *IssueService) BulkMove(req *request.BulkMoveRequest, userID uint64) (*response.BulkCopyMoveResultResponse, error) {
	var targetProject model.Project
	if err := s.db.First(&targetProject, req.TargetProjectID).Error; err != nil {
		return nil, common.NotFound("Target project not found")
	}

	var issues []model.Issue
	if err := s.db.Preload("AssigneeLinks").Preload("LabelLinks").
		Where("id IN ?", req.IssueIDs).Find(&issues).Error; err != nil {
		return nil, common.Internal("Failed to fetch issues")
	}

	if len(issues) == 0 {
		return nil, common.BadRequest("No issues found")
	}

	var failedItems []response.BulkFailedItem
	var results []response.IssueResponse
	tx := s.db.Begin()

	for _, issue := range issues {
		sourceProjectID := issue.ProjectID

		if err := tx.Model(&issue).Updates(map[string]interface{}{
			"project_id":    req.TargetProjectID,
			"workspace_id":  targetProject.WorkspaceID,
			"updated_by_id": userID,
		}).Error; err != nil {
			failedItems = append(failedItems, response.BulkFailedItem{IssueID: issue.ID, Reason: "Failed to move issue"})
			continue
		}

		if req.IncludeSubtasks {
			s.moveSubtasks(tx, issue.ID, req.TargetProjectID, targetProject.WorkspaceID)
		}

		// When moving to a different project, delete stale label links
		if sourceProjectID != req.TargetProjectID {
			tx.Exec("DELETE FROM issue_labels WHERE issue_id = ? AND label_id IN (SELECT id FROM labels WHERE project_id <> ?)", issue.ID, req.TargetProjectID)
		}

		resp, _ := s.buildResponse(issue.ID)
		if resp != nil {
			results = append(results, *resp)
		}

		sourceProjectIDStr := strconv.FormatUint(sourceProjectID, 10)
		targetProjectIDStr := strconv.FormatUint(req.TargetProjectID, 10)
		s.createActivity(tx, issue.ID, "moved", nil,
			&sourceProjectIDStr, &targetProjectIDStr, nil, &userID)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, common.Internal("Failed to commit transaction")
	}

	return &response.BulkCopyMoveResultResponse{
		SuccessCount: len(results),
		FailedCount:  len(failedItems),
		FailedItems:  failedItems,
		Results:      results,
	}, nil
}

// moveSubtasks recursively moves subtasks to the target project.
func (s *IssueService) moveSubtasks(tx *gorm.DB, parentID, targetProjectID, workspaceID uint64) {
	var subtasks []model.Issue
	s.db.Where("parent_id = ?", parentID).Find(&subtasks)

	for _, subtask := range subtasks {
		tx.Model(&subtask).Updates(map[string]interface{}{
			"project_id":   targetProjectID,
			"workspace_id": workspaceID,
		})

		// Delete stale label links that don't belong to the target project
		tx.Exec("DELETE FROM issue_labels WHERE issue_id = ? AND label_id IN (SELECT id FROM labels WHERE project_id <> ?)", subtask.ID, targetProjectID)

		s.moveSubtasks(tx, subtask.ID, targetProjectID, workspaceID)
	}
}

// ConvertType converts an issue's type to the target type.
func (s *IssueService) ConvertType(issueID uint64, req *request.ConvertTypeRequest, userID uint64) (*response.IssueResponse, error) {
	var issue model.Issue
	if err := s.db.Preload("IssueType").Preload("AssigneeLinks").Preload("LabelLinks").First(&issue, issueID).Error; err != nil {
		return nil, common.NotFound("Issue not found")
	}

	var targetType model.IssueType
	if err := s.db.First(&targetType, req.TargetTypeID).Error; err != nil {
		return nil, common.NotFound("Target issue type not found")
	}

	if issue.IssueTypeID != nil && *issue.IssueTypeID == req.TargetTypeID {
		return nil, common.BadRequest("Issue already has this type")
	}

	var storedCF []model.IssueCustomFieldValue
	if err := s.db.Where("issue_id = ?", issueID).Find(&storedCF).Error; err != nil {
		return nil, common.Internal("Failed to load custom field values")
	}
	merged := make(map[uint64]interface{}, len(storedCF))
	for _, row := range storedCF {
		merged[row.FieldID] = row.Value
	}
	if req.CustomFieldValues != nil {
		for fieldID, v := range req.CustomFieldValues {
			merged[fieldID] = v
		}
	}

	assigneeIDs := make([]uint64, 0, len(issue.AssigneeLinks))
	for _, link := range issue.AssigneeLinks {
		assigneeIDs = append(assigneeIDs, link.UserID)
	}
	labelIDs := make([]uint64, 0, len(issue.LabelLinks))
	for _, link := range issue.LabelLinks {
		labelIDs = append(labelIDs, link.LabelID)
	}
	targetTypeID := req.TargetTypeID
	if err := s.validateMandatoryCustomFields(issue.WorkspaceID, &targetTypeID, merged, fieldVisibilityContext{
		IssueTypeID: targetTypeID,
		StateID:     issue.StateID,
		Priority:    issue.Priority,
		AssigneeIDs: assigneeIDs,
		LabelIDs:    labelIDs,
	}); err != nil {
		return nil, err
	}

	oldTypeName := ""
	if issue.IssueType.ID != 0 {
		oldTypeName = issue.IssueType.Name
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, common.Internal("Failed to begin transaction")
	}

	if req.CustomFieldValues != nil {
		for fieldID, value := range req.CustomFieldValues {
			var stringValue string
			switch v := value.(type) {
			case string:
				stringValue = v
			default:
				stringValue = fmt.Sprintf("%v", v)
			}
			if err := tx.Where("issue_id = ? AND field_id = ?", issueID, fieldID).Delete(&model.IssueCustomFieldValue{}).Error; err != nil {
				tx.Rollback()
				return nil, common.Internal("Failed to update custom field values")
			}
			if err := tx.Create(&model.IssueCustomFieldValue{
				IssueID: issueID,
				FieldID: fieldID,
				Value:   stringValue,
			}).Error; err != nil {
				tx.Rollback()
				return nil, common.Internal("Failed to update custom field values")
			}
		}
	}

	// Use Table().Where() to avoid GORM's Model(struct) behavior of
	// re-applying the struct's non-zero field values over the map values.
	if err := tx.Table("issues").Where("id = ?", issueID).Updates(map[string]interface{}{
		"issue_type_id": req.TargetTypeID,
		"updated_by_id": userID,
	}).Error; err != nil {
		tx.Rollback()
		return nil, common.Internal("Failed to update issue type")
	}

	s.createActivity(tx, issue.ID, "converted", nil,
		&oldTypeName, &targetType.Name, nil, &userID)

	if err := tx.Commit().Error; err != nil {
		return nil, common.Internal("Failed to commit transaction")
	}

	return s.buildResponse(issueID)
}

// ==================== Tree View ====================

// Tree returns paginated root-level issues for tree view, optionally with search.
func (s *IssueService) Tree(projectID uint64, filters map[string]interface{}, limit, offset int) ([]response.TreeIssueResponse, int64, error) {
	baseQuery := s.db.Model(&model.Issue{}).Where("issues.project_id = ?", projectID).Where("issues.parent_id IS NULL")

	// Apply simple filters (state, priority, search on root level)
	if stateID, ok := filters["state_id"]; ok && stateID != nil {
		baseQuery = baseQuery.Where("issues.state_id = ?", stateID)
	}
	if priority, ok := filters["priority"]; ok && priority != nil {
		baseQuery = baseQuery.Where("issues.priority = ?", priority)
	}
	if issueTypeID, ok := filters["issue_type_id"]; ok && issueTypeID != nil {
		baseQuery = baseQuery.Where("issues.issue_type_id = ?", issueTypeID)
	}

	searchTerm, hasSearch := filters["search"]
	if hasSearch && searchTerm != nil && searchTerm != "" {
		searchStr := fmt.Sprintf("%%%s%%", searchTerm)
		baseQuery = baseQuery.Where("issues.name ILIKE ? OR COALESCE(issues.description_stripped, '') ILIKE ?",
			searchStr, searchStr)
	}

	// Count total
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, common.Internal("Database error")
	}

	// Paginate
	var issues []model.Issue
	if err := baseQuery.
		Preload("State").
		Preload("IssueType").
		Order("issues.sort_order ASC, issues.sequence_id DESC").
		Limit(limit).Offset(offset).
		Find(&issues).Error; err != nil {
		return nil, 0, common.Internal("Database error")
	}

	// Collect issue IDs for batch sub-issues count
	issueIDs := make([]uint64, len(issues))
	for i, issue := range issues {
		issueIDs[i] = issue.ID
	}

	// Batch count sub-issues
	countMap := make(map[uint64]int64)
	if len(issueIDs) > 0 {
		type countResult struct {
			ParentID uint64
			Count    int64
		}
		var counts []countResult
		s.db.Model(&model.Issue{}).
			Select("parent_id, COUNT(*) as count").
			Where("parent_id IN ?", issueIDs).
			Group("parent_id").
			Scan(&counts)
		for _, c := range counts {
			countMap[c.ParentID] = c.Count
		}
	}

	result := make([]response.TreeIssueResponse, len(issues))
	for i, issue := range issues {
		resp := buildTreeIssueFromModel(&issue, countMap)
		result[i] = *resp
	}
	return result, total, nil
}

// TreeSearch finds issues matching the search term anywhere in the tree,
// and returns results with ancestor chains for tree view display.
func (s *IssueService) TreeSearch(projectID uint64, search string, limit, offset int) ([]response.TreeSearchResult, int64, error) {
	searchStr := fmt.Sprintf("%%%s%%", search)

	// Find all matching issues (not archived)
	baseQuery := s.db.Model(&model.Issue{}).
		Where("issues.project_id = ?", projectID).
		Where("issues.archived_at IS NULL").
		Where("issues.name ILIKE ? OR COALESCE(issues.description_stripped, '') ILIKE ?",
			searchStr, searchStr)

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, 0, common.Internal("Database error")
	}

	var matchedIssues []model.Issue
	if err := baseQuery.
		Preload("State").
		Preload("IssueType").
		Order("issues.depth ASC, issues.sort_order ASC, issues.sequence_id DESC").
		Limit(limit).Offset(offset).
		Find(&matchedIssues).Error; err != nil {
		return nil, 0, common.Internal("Database error")
	}

	// For each matched issue, find its root ancestor and build ancestor chain
	results := make([]response.TreeSearchResult, 0, len(matchedIssues))

	for _, matched := range matchedIssues {
		// If already root, no ancestor chain needed
		if matched.ParentID == nil || *matched.ParentID == 0 {
			rootResp := buildTreeIssueFromModel(&matched, nil)
			rootResp.IsSearchMatch = true
			results = append(results, response.TreeSearchResult{
				RootIssue:     *rootResp,
				MatchedIssue:  *rootResp,
				AncestorChain: []response.AncestorInfo{},
			})
			continue
		}

		// Build ancestor chain by walking up parent_id
		ancestorChain := make([]response.AncestorInfo, 0)
		currentParentID := *matched.ParentID
		var rootIssue *model.Issue

		for {
			var parent model.Issue
			if err := s.db.Select("id, name, sequence_id, parent_id, depth, priority, state_id, sort_order").
				First(&parent, currentParentID).Error; err != nil {
				break
			}

			if parent.ParentID == nil || *parent.ParentID == 0 {
				// This is the root
				rootIssue = &parent
				break
			}

			// Insert at beginning to maintain root->child order
			ancestorChain = append([]response.AncestorInfo{{
				ID:         parent.ID,
				Name:       parent.Name,
				SequenceID: parent.SequenceID,
			}}, ancestorChain...)
			currentParentID = *parent.ParentID
		}

		if rootIssue == nil {
			// Fallback: use matched issue itself as root
			rootResp := buildTreeIssueFromModel(&matched, nil)
			rootResp.IsSearchMatch = true
			results = append(results, response.TreeSearchResult{
				RootIssue:     *rootResp,
				MatchedIssue:  *rootResp,
				AncestorChain: []response.AncestorInfo{},
			})
			continue
		}

		// Load full root issue details
		var fullRoot model.Issue
		if err := s.db.Preload("State").Preload("IssueType").First(&fullRoot, rootIssue.ID).Error; err != nil {
			continue
		}
		rootResp := buildTreeIssueFromModel(&fullRoot, nil)

		matchedResp := buildTreeIssueFromModel(&matched, nil)
		matchedResp.IsSearchMatch = true

		results = append(results, response.TreeSearchResult{
			RootIssue:     *rootResp,
			MatchedIssue:  *matchedResp,
			AncestorChain: ancestorChain,
		})
	}

	return results, total, nil
}

// Children returns direct children of a parent issue (lazy loading for tree expansion).
func (s *IssueService) Children(parentID uint64) ([]response.TreeIssueResponse, error) {
	var children []model.Issue
	if err := s.db.
		Where("parent_id = ?", parentID).
		Preload("State").
		Preload("IssueType").
		Order("sort_order ASC, sequence_id DESC").
		Find(&children).Error; err != nil {
		return nil, common.Internal("Database error")
	}

	// Batch count sub-issues for each child
	childIDs := make([]uint64, len(children))
	for i, c := range children {
		childIDs[i] = c.ID
	}
	countMap := make(map[uint64]int64)
	if len(childIDs) > 0 {
		type countResult struct {
			ParentID uint64
			Count    int64
		}
		var counts []countResult
		s.db.Model(&model.Issue{}).
			Select("parent_id, COUNT(*) as count").
			Where("parent_id IN ?", childIDs).
			Group("parent_id").
			Scan(&counts)
		for _, c := range counts {
			countMap[c.ParentID] = c.Count
		}
	}

	result := make([]response.TreeIssueResponse, len(children))
	for i, child := range children {
		resp := buildTreeIssueFromModel(&child, countMap)
		result[i] = *resp
	}
	return result, nil
}

// buildTreeIssueFromModel converts a model.Issue to TreeIssueResponse with optional count map.
func buildTreeIssueFromModel(issue *model.Issue, countMap map[uint64]int64) *response.TreeIssueResponse {
	resp := &response.TreeIssueResponse{
		ID:          issue.ID,
		Name:        issue.Name,
		SequenceID:  issue.SequenceID,
		Priority:    issue.Priority,
		StateID:     issue.StateID,
		ParentID:    issue.ParentID,
		Depth:       issue.Depth,
		IssueTypeID: issue.IssueTypeID,
		StartDate:   issue.StartDate,
		TargetDate:  issue.TargetDate,
	}

	if issue.State.ID != 0 {
		resp.StateName = issue.State.Name
		resp.StateGroup = issue.State.Group
	}
	if issue.IssueType.ID != 0 {
		resp.IssueType = &response.IssueTypeLite{
			ID:                  issue.IssueType.ID,
			Name:                issue.IssueType.Name,
			Color:               issue.IssueType.Color,
			Icon:                issue.IssueType.Icon,
			Level:               issue.IssueType.Level,
			AllowedChildTypeIDs: issue.IssueType.AllowedChildTypeIDs,
		}
	}

	if countMap != nil {
		if count, ok := countMap[issue.ID]; ok {
			resp.SubIssuesCount = count
			resp.HasChildren = count > 0
		}
	} else {
		resp.SubIssuesCount = int64(len(issue.SubIssues))
		resp.HasChildren = len(issue.SubIssues) > 0
	}

	return resp
}

// MergeDuplicates soft-merges source issues into a target (SRS 2026-11-07-merge-duplicates).
// Sources are kept (not deleted): cancelled-like state when available, "重复" relation to target,
// labels/assignees unioned onto target, merge summary appended to target description.
func (s *IssueService) MergeDuplicates(req *request.MergeDuplicatesRequest, userID uint64) (*response.IssueResponse, error) {
	if len(req.SourceIssueIDs) == 0 {
		return nil, common.BadRequest("No source issues provided")
	}
	for _, sid := range req.SourceIssueIDs {
		if sid == req.TargetIssueID {
			return nil, common.BadRequest("Cannot merge an issue into itself")
		}
	}

	var targetIssue model.Issue
	if err := s.db.Preload("LabelLinks").Preload("AssigneeLinks").First(&targetIssue, req.TargetIssueID).Error; err != nil {
		return nil, common.NotFound("Target issue not found")
	}

	var sourceIssues []model.Issue
	if err := s.db.Preload("LabelLinks").Preload("AssigneeLinks").
		Where("id IN ?", req.SourceIssueIDs).Find(&sourceIssues).Error; err != nil {
		return nil, common.Internal("Failed to fetch source issues")
	}
	if len(sourceIssues) != len(req.SourceIssueIDs) {
		return nil, common.NotFound("One or more source issues not found")
	}
	for _, source := range sourceIssues {
		if source.ProjectID != targetIssue.ProjectID || source.WorkspaceID != targetIssue.WorkspaceID {
			return nil, common.BadRequest("Source and target issues must be in the same project")
		}
	}

	// P0 always unions labels and assignees onto the target (SRS).
	keepLabels, keepAssignees := true, true
	_ = req.KeepSourceComments // comments/attachments not migrated in P0

	dupRelType, err := s.findDuplicateRelationType(targetIssue.WorkspaceID)
	if err != nil {
		return nil, err
	}

	cancelledStateID, _ := s.findCancelledStateID(targetIssue.ProjectID, targetIssue.WorkspaceID)

	tx := s.db.Begin()
	if tx.Error != nil {
		return nil, common.Internal("Failed to begin transaction")
	}

	for _, source := range sourceIssues {
		already, existingTarget, err := s.existingMergeTarget(tx, source.ID, dupRelType.ID)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		if already {
			if existingTarget == targetIssue.ID {
				continue // idempotent no-op for this source
			}
			tx.Rollback()
			return nil, common.Conflict("Source issue already merged into a different target")
		}

		if keepLabels {
			for _, labelLink := range source.LabelLinks {
				var label model.Label
				if err := tx.Where("id = ? AND project_id = ?", labelLink.LabelID, targetIssue.ProjectID).First(&label).Error; err != nil {
					continue
				}
				var n int64
				tx.Model(&model.IssueLabel{}).Where("issue_id = ? AND label_id = ?", targetIssue.ID, labelLink.LabelID).Count(&n)
				if n == 0 {
					if err := tx.Create(&model.IssueLabel{IssueID: targetIssue.ID, LabelID: labelLink.LabelID}).Error; err != nil {
						tx.Rollback()
						return nil, common.Internal("Failed to copy label")
					}
				}
			}
		}

		if keepAssignees {
			for _, assigneeLink := range source.AssigneeLinks {
				var n int64
				tx.Model(&model.IssueAssignee{}).Where("issue_id = ? AND user_id = ?", targetIssue.ID, assigneeLink.UserID).Count(&n)
				if n == 0 {
					if err := tx.Create(&model.IssueAssignee{IssueID: targetIssue.ID, UserID: assigneeLink.UserID}).Error; err != nil {
						tx.Rollback()
						return nil, common.Internal("Failed to copy assignee")
					}
				}
			}
		}

		comment := fmt.Sprintf("merged into #%d", targetIssue.SequenceID)
		if err := tx.Create(&model.IssueRelation{
			IssueID:        source.ID,
			RelatedIssueID: targetIssue.ID,
			RelationTypeID: dupRelType.ID,
			Comment:        &comment,
		}).Error; err != nil {
			tx.Rollback()
			return nil, common.Internal("Failed to create duplicate relation")
		}

		if cancelledStateID != 0 && source.StateID != cancelledStateID {
			oldState := strconv.FormatUint(source.StateID, 10)
			newState := strconv.FormatUint(cancelledStateID, 10)
			if err := tx.Model(&model.Issue{}).Where("id = ?", source.ID).Update("state_id", cancelledStateID).Error; err != nil {
				tx.Rollback()
				return nil, common.Internal("Failed to update source state")
			}
			field := "state_id"
			s.createActivity(tx, source.ID, "updated", &field, &oldState, &newState, nil, &userID)
		}

		targetIDStr := strconv.FormatUint(targetIssue.ID, 10)
		s.createActivity(tx, source.ID, "merged", strPtr("merged_into"), nil, &targetIDStr, nil, &userID)
		sourceIDStr := strconv.FormatUint(source.ID, 10)
		s.createActivity(tx, targetIssue.ID, "merged", strPtr("merged_from"), nil, &sourceIDStr, nil, &userID)

		summary := fmt.Sprintf(
			`<p><em>Merged from #%d: %s</em></p>`,
			source.SequenceID, html.EscapeString(source.Name),
		)
		if stripped := strings.TrimSpace(source.DescriptionHTML); stripped != "" && stripped != "<p></p>" {
			// Keep a short excerpt; avoid unbounded growth.
			excerpt := stripped
			if len([]rune(excerpt)) > 500 {
				excerpt = string([]rune(excerpt)[:500]) + "…"
			}
			summary += excerpt
		}
		newDesc := targetIssue.DescriptionHTML + summary
		if err := tx.Model(&model.Issue{}).Where("id = ?", targetIssue.ID).Update("description_html", newDesc).Error; err != nil {
			tx.Rollback()
			return nil, common.Internal("Failed to append merge summary")
		}
		targetIssue.DescriptionHTML = newDesc
	}

	if err := tx.Commit().Error; err != nil {
		return nil, common.Internal("Failed to commit transaction")
	}
	return s.buildResponse(targetIssue.ID)
}

func (s *IssueService) findDuplicateRelationType(workspaceID uint64) (*model.RelationType, error) {
	var rt model.RelationType
	err := s.db.Where("workspace_id = ? AND (name = ? OR inward_name = ? OR outward_name = ?)",
		workspaceID, "重复", "重复于", "被重复于").First(&rt).Error
	if err == nil {
		return &rt, nil
	}
	// Create workspace default if missing (tests / sparse seeds).
	rt = model.RelationType{
		Name: "重复", InwardName: "重复于", OutwardName: "被重复于", WorkspaceID: workspaceID,
	}
	if err := s.db.Create(&rt).Error; err != nil {
		return nil, common.Internal("Failed to ensure duplicate relation type")
	}
	return &rt, nil
}

func (s *IssueService) findCancelledStateID(projectID, workspaceID uint64) (uint64, error) {
	var st model.State
	// Prefer project-scoped cancelled state.
	err := s.db.Where("workspace_id = ? AND project_id = ? AND \"group\" = ?",
		workspaceID, projectID, common.StateGroupCancelled).First(&st).Error
	if err == nil {
		return st.ID, nil
	}
	err = s.db.Where("workspace_id = ? AND project_id IS NULL AND \"group\" = ?",
		workspaceID, common.StateGroupCancelled).First(&st).Error
	if err != nil {
		return 0, err
	}
	return st.ID, nil
}

func (s *IssueService) existingMergeTarget(db *gorm.DB, sourceID, dupRelTypeID uint64) (bool, uint64, error) {
	var rel model.IssueRelation
	err := db.Where("issue_id = ? AND relation_type_id = ?", sourceID, dupRelTypeID).
		Order("id ASC").First(&rel).Error
	if err == gorm.ErrRecordNotFound {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, common.Internal("Failed to check existing merge relation")
	}
	return true, rel.RelatedIssueID, nil
}

// ==================== Watcher ====================

// AddWatcher adds a user as a watcher for an issue.
func (s *IssueService) AddWatcher(issueID, userID uint64) error {
	var watcher model.IssueWatcher
	if err := s.db.Where("issue_id = ? AND user_id = ?", issueID, userID).First(&watcher).Error; err == nil {
		return common.BadRequest("User is already watching this issue")
	}

	return s.db.Create(&model.IssueWatcher{IssueID: issueID, UserID: userID}).Error
}

// RemoveWatcher removes a user from watching an issue.
func (s *IssueService) RemoveWatcher(issueID, userID uint64) error {
	result := s.db.Where("issue_id = ? AND user_id = ?", issueID, userID).Delete(&model.IssueWatcher{})
	if result.RowsAffected == 0 {
		return common.BadRequest("User is not watching this issue")
	}
	return result.Error
}

// ListWatchers returns watchers for an issue.
func (s *IssueService) ListWatchers(issueID uint64) ([]uint64, error) {
	var watchers []model.IssueWatcher
	if err := s.db.Where("issue_id = ?", issueID).Find(&watchers).Error; err != nil {
		return nil, common.Internal("Failed to fetch watchers")
	}
	userIDs := make([]uint64, len(watchers))
	for i, w := range watchers {
		userIDs[i] = w.UserID
	}
	return userIDs, nil
}

// CheckDuplicates finds similar issues in the same project (soft-deleted excluded via GORM).
func (s *IssueService) CheckDuplicates(projectID uint64, name, description string, limit int) (*response.DuplicateCheckResponse, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	result := &response.DuplicateCheckResponse{Duplicates: []response.DuplicateIssueItem{}}
	if len([]rune(name)) < duplicateMinNameLenRunes {
		return result, nil
	}
	if limit <= 0 || limit > duplicateCheckLimit {
		limit = duplicateCheckLimit
	}

	pattern := "%" + escapeLike(name) + "%"
	var issues []model.Issue
	q := s.db.Model(&model.Issue{}).
		Where("project_id = ?", projectID).
		Where("archived_at IS NULL").
		Where("name ILIKE ? OR COALESCE(description_stripped, '') ILIKE ?", pattern, pattern).
		Order("updated_at DESC").
		Limit(duplicateCandidateLimit)
	if err := q.Find(&issues).Error; err != nil {
		return nil, common.Internal("Database error")
	}

	candidates := make([]scoredDuplicate, len(issues))
	for i, issue := range issues {
		candidates[i] = scoredDuplicate{
			ID:         issue.ID,
			SequenceID: issue.SequenceID,
			Name:       issue.Name,
			Priority:   issue.Priority,
			StateID:    issue.StateID,
		}
	}
	ranked := rankDuplicates(name, description, candidates)
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	for _, item := range ranked {
		stateID := item.StateID
		result.Duplicates = append(result.Duplicates, response.DuplicateIssueItem{
			ID:         item.ID,
			SequenceID: item.SequenceID,
			Name:       item.Name,
			Priority:   item.Priority,
			StateID:    &stateID,
			Similarity: item.Similarity,
		})
	}
	return result, nil
}
