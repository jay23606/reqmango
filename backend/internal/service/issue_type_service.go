package service

import (
	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/dto/response"
	"github.com/reqmango/backend/internal/issuetype"
	"github.com/reqmango/backend/internal/model"
	"gorm.io/gorm"
)

type IssueTypeService struct {
	db *gorm.DB
}

func NewIssueTypeService(db *gorm.DB) *IssueTypeService {
	return &IssueTypeService{db: db}
}

// ResolveWorkspaceID returns the workspace_id for a given project ID.
// Returns 0 and an error if the project does not exist.
func (s *IssueTypeService) ResolveWorkspaceID(projectID uint64) (uint64, error) {
	var project model.Project
	if err := s.db.Select("workspace_id").First(&project, projectID).Error; err != nil {
		return 0, common.NotFound("Project not found")
	}
	return project.WorkspaceID, nil
}

// buildResponse converts an IssueType model to its API response shape.
// isImported indicates the project has explicitly imported this workspace-level
// type via the workspace-type Import model (only meaningful in project context).
func (s *IssueTypeService) buildResponse(t model.IssueType, isImported bool) *response.IssueTypeResponse {
	return &response.IssueTypeResponse{
		ID:                  t.ID,
		Name:                t.Name,
		Color:               t.Color,
		Icon:                t.Icon,
		Description:         t.Description,
		Level:               t.Level,
		ParentTypeID:        t.ParentTypeID,
		AllowedChildTypeIDs: t.AllowedChildTypeIDs,
		IsDefault:           t.IsDefault,
		Sequence:            t.Sequence,
		IsActive:            t.IsActive,
		ProjectID:           t.ProjectID,
		WorkspaceID:         t.WorkspaceID,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
		IsInherited:         t.ProjectID == nil,
		IsImported:          isImported,
	}
}

// ensureOneDefault ensures only one default type exists in the given scope.
func (s *IssueTypeService) ensureOneDefault(workspaceID, excludeID uint64, projectID *uint64) error {
	query := s.db.Model(&model.IssueType{}).Where("workspace_id = ? AND is_default = ?", workspaceID, true)
	if projectID != nil {
		query = query.Where("project_id = ?", *projectID)
	} else {
		query = query.Where("project_id IS NULL")
	}
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	return query.Update("is_default", false).Error
}

// listImportedTypeIDs returns the set of workspace-level type IDs that the
// given project has explicitly imported via the Import model.
func (s *IssueTypeService) listImportedTypeIDs(projectID uint64) (map[uint64]bool, error) {
	var imports []model.IssueTypeImport
	if err := s.db.Where("project_id = ?", projectID).Find(&imports).Error; err != nil {
		return nil, err
	}
	m := make(map[uint64]bool, len(imports))
	for _, i := range imports {
		m[i.WorkspaceTypeID] = true
	}
	return m, nil
}

// ==================== CRUD ====================

func (s *IssueTypeService) Create(workspaceID, userID uint64, req request.IssueTypeCreate) (*response.IssueTypeResponse, error) {
	if req.IsDefault {
		if err := s.ensureOneDefault(workspaceID, 0, req.ProjectID); err != nil {
			return nil, common.Internal("Failed to update defaults")
		}
	}
	if req.Sequence == 0 {
		req.Sequence = 1
	}
	if req.Color == "" {
		req.Color = "#6366F1"
	}
	if req.Icon == "" {
		req.Icon = "circle"
	}

	t := model.IssueType{
		Name:         req.Name,
		Color:        req.Color,
		Icon:         req.Icon,
		Description:  req.Description,
		Level:        req.Level,
		ParentTypeID: req.ParentTypeID,
		IsDefault:    req.IsDefault,
		Sequence:     req.Sequence,
		IsActive:     true,
		ProjectID:    req.ProjectID,
		WorkspaceID:  workspaceID,
	}
	if req.AllowedChildTypeIDs != nil {
		t.AllowedChildTypeIDs = req.AllowedChildTypeIDs
	}
	t.CreatedByID = &userID

	if err := s.db.Create(&t).Error; err != nil {
		return nil, common.Internal("Failed to create issue type")
	}

	return s.buildResponse(t, false), nil
}

// List returns issue types visible in the given scope.
//
//   - workspace scope (projectID == nil): returns all workspace-level types
//     (project_id IS NULL). IsImported is always false (no project context).
//   - project scope (projectID != nil): returns project-private types plus
//     workspace-level types that the project has explicitly imported via the
//     workspace-type Import model. Legacy auto-inherit is removed — workspace
//     types are only visible when imported.
//
// The scoping rules live in issuetype.Visible so the agent tool layer hands out
// the same ids this endpoint advertises.
func (s *IssueTypeService) List(workspaceID uint64, projectID *uint64) ([]response.IssueTypeResponse, error) {
	types, err := issuetype.Visible(s.db, workspaceID, projectID)
	if err != nil {
		return nil, common.Internal("Failed to list issue types")
	}

	// In project scope, any workspace-level type (project_id IS NULL) is
	// visible because the project imported it.
	result := make([]response.IssueTypeResponse, len(types))
	for i, t := range types {
		result[i] = *s.buildResponse(t, projectID != nil && t.ProjectID == nil)
	}
	return result, nil
}

func (s *IssueTypeService) Get(typeID uint64) (*response.IssueTypeResponse, error) {
	var t model.IssueType
	if err := s.db.First(&t, typeID).Error; err != nil {
		return nil, common.NotFound("Issue type not found")
	}
	return s.buildResponse(t, false), nil
}

func (s *IssueTypeService) Update(typeID, userID uint64, req request.IssueTypeUpdate) (*response.IssueTypeResponse, error) {
	var t model.IssueType
	if err := s.db.First(&t, typeID).Error; err != nil {
		return nil, common.NotFound("Issue type not found")
	}

	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Color != nil {
		t.Color = *req.Color
	}
	if req.Icon != nil {
		t.Icon = *req.Icon
	}
	if req.IsDefault != nil {
		if *req.IsDefault {
			if err := s.ensureOneDefault(t.WorkspaceID, t.ID, t.ProjectID); err != nil {
				return nil, common.Internal("Failed to update defaults")
			}
		}
		t.IsDefault = *req.IsDefault
	}
	if req.Sequence != nil {
		t.Sequence = *req.Sequence
	}
	if req.IsActive != nil {
		t.IsActive = *req.IsActive
	}
	if req.ProjectID != nil {
		t.ProjectID = req.ProjectID
	}
	if req.Description != nil {
		t.Description = *req.Description
	}
	if req.Level != nil {
		t.Level = *req.Level
	}
	if req.ParentTypeID != nil {
		t.ParentTypeID = req.ParentTypeID
	}
	if req.AllowedChildTypeIDs != nil {
		t.AllowedChildTypeIDs = req.AllowedChildTypeIDs
	}

	t.UpdatedByID = &userID
	if err := s.db.Save(&t).Error; err != nil {
		return nil, common.Internal("Failed to update issue type")
	}

	return s.buildResponse(t, false), nil
}

func (s *IssueTypeService) Delete(typeID uint64) error {
	var t model.IssueType
	if err := s.db.First(&t, typeID).Error; err != nil {
		return common.NotFound("Issue type not found")
	}

	// Check if any issues are using this type
	var count int64
	if err := s.db.Model(&model.Issue{}).Where("issue_type_id = ?", typeID).Count(&count).Error; err != nil {
		return common.Internal("Failed to check issue type usage")
	}
	if count > 0 {
		return common.BadRequest("Cannot delete issue type that is in use by issues")
	}

	// Clean up join table entries
	s.db.Where("type_id = ?", typeID).Delete(&model.IssueTypeField{})

	// Clean up import records referencing this workspace type
	s.db.Where("workspace_type_id = ?", typeID).Delete(&model.IssueTypeImport{})

	return s.db.Delete(&t).Error
}

func (s *IssueTypeService) Disable(typeID uint64, isActive bool) error {
	var t model.IssueType
	if err := s.db.First(&t, typeID).Error; err != nil {
		return common.NotFound("Issue type not found")
	}
	return s.db.Model(&t).Update("is_active", isActive).Error
}

// ==================== Field Association ====================

func (s *IssueTypeService) AddField(typeID, fieldID uint64, isRequired bool, sequence int) (*response.IssueTypeFieldResponse, error) {
	// Verify type exists
	var t model.IssueType
	if err := s.db.First(&t, typeID).Error; err != nil {
		return nil, common.NotFound("Issue type not found")
	}

	// Verify field exists
	var f model.CustomField
	if err := s.db.First(&f, fieldID).Error; err != nil {
		return nil, common.NotFound("Custom field not found")
	}

	// Check for duplicate
	var count int64
	s.db.Model(&model.IssueTypeField{}).Where("type_id = ? AND field_id = ?", typeID, fieldID).Count(&count)
	if count > 0 {
		return nil, common.Conflict("Field is already associated with this issue type")
	}

	link := model.IssueTypeField{
		TypeID:     typeID,
		FieldID:    fieldID,
		IsRequired: isRequired,
		Sequence:   sequence,
	}

	if err := s.db.Create(&link).Error; err != nil {
		return nil, common.Internal("Failed to add field to issue type")
	}

	return &response.IssueTypeFieldResponse{
		FieldID:     fieldID,
		TypeID:      typeID,
		IsRequired:  isRequired,
		Sequence:    sequence,
		Name:        f.Name,
		FieldType:   f.FieldType,
		Description: f.Description,
	}, nil
}

func (s *IssueTypeService) RemoveField(typeID, fieldID uint64) error {
	result := s.db.Where("type_id = ? AND field_id = ?", typeID, fieldID).Delete(&model.IssueTypeField{})
	if result.RowsAffected == 0 {
		return common.NotFound("Field association not found")
	}
	return nil
}

func (s *IssueTypeService) UpdateField(typeID, fieldID uint64, req request.IssueTypeFieldUpdate) (*response.IssueTypeFieldResponse, error) {
	var link model.IssueTypeField
	if err := s.db.Where("type_id = ? AND field_id = ?", typeID, fieldID).First(&link).Error; err != nil {
		return nil, common.NotFound("Field association not found")
	}

	if req.IsRequired != nil {
		link.IsRequired = *req.IsRequired
	}
	if req.Sequence != nil {
		link.Sequence = *req.Sequence
	}

	if err := s.db.Save(&link).Error; err != nil {
		return nil, common.Internal("Failed to update field association")
	}

	// Load field info for response
	var f model.CustomField
	s.db.First(&f, fieldID)

	return &response.IssueTypeFieldResponse{
		FieldID:     fieldID,
		TypeID:      typeID,
		IsRequired:  link.IsRequired,
		Sequence:    link.Sequence,
		Name:        f.Name,
		FieldType:   f.FieldType,
		Description: f.Description,
	}, nil
}

// ==================== Project-scoped Operations ====================

// Reorder updates the sequence for issue types within a project.
func (s *IssueTypeService) Reorder(projectID uint64, typeIDs []uint64) error {
	for i, typeID := range typeIDs {
		if err := s.db.Model(&model.IssueType{}).Where("id = ? AND project_id = ?", typeID, projectID).
			Update("sequence", i+1).Error; err != nil {
			return common.Internal("Failed to reorder issue types")
		}
	}
	return nil
}

// ReorderWorkspace updates the sequence for issue types within a workspace.
func (s *IssueTypeService) ReorderWorkspace(workspaceID uint64, typeIDs []uint64) error {
	for i, typeID := range typeIDs {
		if err := s.db.Model(&model.IssueType{}).Where("id = ? AND workspace_id = ?", typeID, workspaceID).
			Update("sequence", i+1).Error; err != nil {
			return common.Internal("Failed to reorder issue types")
		}
	}
	return nil
}

// ListFields returns the custom fields attached to a type.
//
// All attached fields are always visible — no enrollment filter is needed:
//   - Project-private types: fields are naturally visible within the project.
//   - Workspace types visible in a project: they are always imported (legacy
//     auto-inherit has been removed), so custom fields "follow" the type
//     automatically without requiring separate enrollment.
func (s *IssueTypeService) ListFields(typeID uint64, projectID ...uint64) ([]response.IssueTypeFieldResponse, error) {
	var t model.IssueType
	if err := s.db.First(&t, typeID).Error; err != nil {
		return nil, common.NotFound("Issue type not found")
	}

	var links []model.IssueTypeField
	if err := s.db.Preload("Field").Where("type_id = ?", typeID).Order("sequence").Find(&links).Error; err != nil {
		return nil, common.Internal("Failed to list fields")
	}

	result := make([]response.IssueTypeFieldResponse, 0)
	for _, link := range links {
		if link.Field.ID == 0 || !link.Field.IsActive {
			continue
		}
		fr := response.IssueTypeFieldResponse{
			FieldID:     link.FieldID,
			TypeID:      link.TypeID,
			IsRequired:  link.IsRequired,
			Sequence:    link.Sequence,
			Name:        link.Field.Name,
			FieldType:   link.Field.FieldType,
			Description: link.Field.Description,
			Options:     make([]response.CustomFieldOptionResponse, 0),
		}
		if link.Field.FieldType == "dropdown" {
			var opts []model.CustomFieldOption
			s.db.Where("field_id = ?", link.FieldID).Order("sequence").Find(&opts)
			for _, o := range opts {
				fr.Options = append(fr.Options, response.CustomFieldOptionResponse{
					ID:       o.ID,
					FieldID:  o.FieldID,
					Value:    o.Value,
					Color:    o.Color,
					Sequence: o.Sequence,
				})
			}
		}
		result = append(result, fr)
	}
	if result == nil {
		result = []response.IssueTypeFieldResponse{}
	}
	return result, nil
}

// ==================== workspace-type Import Model ====================

// ListImportable returns workspace-level types that the project has NOT yet
// imported. These are candidates for the Import dialog in the project UI.
func (s *IssueTypeService) ListImportable(workspaceID, projectID uint64) ([]response.IssueTypeResponse, error) {
	importedIDs, err := s.listImportedTypeIDs(projectID)
	if err != nil {
		return nil, common.Internal("Failed to load import records")
	}

	var types []model.IssueType
	if err := s.db.Where("workspace_id = ? AND project_id IS NULL AND is_active = ?", workspaceID, true).
		Order("sequence, created_at").Find(&types).Error; err != nil {
		return nil, common.Internal("Failed to list workspace issue types")
	}

	result := make([]response.IssueTypeResponse, 0, len(types))
	for _, t := range types {
		if importedIDs[t.ID] {
			continue
		}
		r := s.buildResponse(t, false)
		r.IsImported = false
		result = append(result, *r)
	}
	if result == nil {
		result = []response.IssueTypeResponse{}
	}
	return result, nil
}

// ImportType records a project's reference to a workspace-level type (workspace-type
// Import model). After import, custom fields attached to the type become
// visible in the project automatically — no separate enrollment required.
func (s *IssueTypeService) ImportType(projectID, workspaceTypeID uint64) error {
	var t model.IssueType
	if err := s.db.First(&t, workspaceTypeID).Error; err != nil {
		return common.NotFound("Issue type not found")
	}
	if t.ProjectID != nil {
		return common.BadRequest("Cannot import a project-level type")
	}

	rec := model.IssueTypeImport{
		ProjectID:       projectID,
		WorkspaceTypeID: workspaceTypeID,
		WorkspaceID:     t.WorkspaceID,
	}
	if err := s.db.Create(&rec).Error; err != nil {
		if common.IsUniqueViolation(err) {
			return common.Conflict("Type already imported by this project")
		}
		return common.Internal("Failed to import issue type")
	}
	return nil
}

// UnimportType removes a project's reference to a workspace-level type.
// The type itself is not deleted — it remains available at the workspace level.
// After unimporting, the type is no longer visible in the project's type list
// (legacy auto-inherit has been removed; only explicitly imported workspace
// types appear in project scope).
//
// Hard-deletes (Unscoped) instead of soft-deletes because IssueTypeImport is a
// pure reference/join table: a soft-deleted row would still occupy the
// (project_id, workspace_type_id) unique index and block a subsequent
// re-import with a 409 Conflict. Unimport is fully reversible via ImportType.
func (s *IssueTypeService) UnimportType(projectID, workspaceTypeID uint64) error {
	result := s.db.Unscoped().
		Where("project_id = ? AND workspace_type_id = ?", projectID, workspaceTypeID).
		Delete(&model.IssueTypeImport{})
	if result.RowsAffected == 0 {
		return common.NotFound("Import record not found")
	}
	return nil
}

// IsImported reports whether the project has explicitly imported the given
// workspace-level type via the workspace-type Import model.
func (s *IssueTypeService) IsImported(projectID, workspaceTypeID uint64) bool {
	var count int64
	s.db.Model(&model.IssueTypeImport{}).
		Where("project_id = ? AND workspace_type_id = ?", projectID, workspaceTypeID).
		Count(&count)
	return count > 0
}
