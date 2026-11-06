package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/reqmango/backend/internal/common"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/dto/response"
	"github.com/reqmango/backend/internal/model"
	"gorm.io/gorm"
)

type CustomFieldService struct {
	db *gorm.DB
}

func NewCustomFieldService(db *gorm.DB) *CustomFieldService {
	return &CustomFieldService{db: db}
}

var validFieldTypes = map[string]bool{
	"text": true, "number": true, "dropdown": true,
	"boolean": true, "date": true, "member": true, "url": true,
}

func (s *CustomFieldService) buildResponse(f model.CustomField) *response.CustomFieldResponse {
	r := &response.CustomFieldResponse{
		ID:            f.ID,
		Name:          f.Name,
		Description:   f.Description,
		FieldType:     f.FieldType,
		IsRequired:    f.IsRequired,
		DefaultValue:  f.DefaultValue,
		Placeholder:   f.Placeholder,
		IsActive:      f.IsActive,
		IsReadonly:    f.IsReadonly,
		IsMultiSelect: f.IsMultiSelect,
		NumberMin:     f.NumberMin,
		NumberMax:     f.NumberMax,
		TypeNames:     make([]string, 0),
		ProjectID:     f.ProjectID,
		WorkspaceID:   f.WorkspaceID,
		CreatedAt:     f.CreatedAt,
		UpdatedAt:     f.UpdatedAt,
		Options:       make([]response.CustomFieldOptionResponse, 0),
	}

	if f.FieldType == "dropdown" {
		var opts []model.CustomFieldOption
		s.db.Where("field_id = ?", f.ID).Order("sequence").Find(&opts)
		r.Options = make([]response.CustomFieldOptionResponse, len(opts))
		for i, o := range opts {
			r.Options[i] = response.CustomFieldOptionResponse{
				ID:       o.ID,
				FieldID:  o.FieldID,
				Value:    o.Value,
				Color:    o.Color,
				Sequence: o.Sequence,
			}
		}
	}
	return r
}

func (s *CustomFieldService) validateFieldType(fieldType string) error {
	if !validFieldTypes[fieldType] {
		return common.Validation("Invalid field type. Must be one of: text, number, dropdown, boolean, date, member, url")
	}
	return nil
}

// validateFieldValue validates a value against the field's type definition.
func (s *CustomFieldService) validateFieldValue(field *model.CustomField, value string) error {
	if field.FieldType == "dropdown" && value != "" {
		values := []string{value}
		if field.IsMultiSelect && strings.HasPrefix(value, "[") {
			if err := json.Unmarshal([]byte(value), &values); err != nil {
				return common.Validation("Multi-select value must be a JSON array of option values")
			}
		}
		for _, v := range values {
			var count int64
			s.db.Model(&model.CustomFieldOption{}).Where("field_id = ? AND value = ?", field.ID, v).Count(&count)
			if count == 0 {
				return common.Validation(fmt.Sprintf("Value '%s' is not a valid option for field '%s'", v, field.Name))
			}
		}
	}
	if field.FieldType == "number" && value != "" {
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return common.Validation(fmt.Sprintf("Field '%s' must be a number", field.Name))
		}
		if field.NumberMin != nil && n < *field.NumberMin {
			return common.Validation(fmt.Sprintf("Field '%s' must be >= %v", field.Name, *field.NumberMin))
		}
		if field.NumberMax != nil && n > *field.NumberMax {
			return common.Validation(fmt.Sprintf("Field '%s' must be <= %v", field.Name, *field.NumberMax))
		}
	}
	if field.FieldType == "boolean" && value != "" {
		if value != "true" && value != "false" {
			return common.Validation("Boolean field must be 'true' or 'false'")
		}
	}
	return nil
}

// ==================== Field CRUD ====================

func (s *CustomFieldService) Create(workspaceID, userID uint64, req request.CustomFieldCreate) (*response.CustomFieldResponse, error) {
	if err := s.validateFieldType(req.FieldType); err != nil {
		return nil, err
	}

	f := model.CustomField{
		Name:          req.Name,
		Description:   req.Description,
		FieldType:     req.FieldType,
		IsRequired:    req.IsRequired,
		DefaultValue:  req.DefaultValue,
		Placeholder:   req.Placeholder,
		IsActive:      true,
		IsReadonly:    req.IsReadonly,
		IsMultiSelect: req.IsMultiSelect,
		NumberMin:     req.NumberMin,
		NumberMax:     req.NumberMax,
		ProjectID:     req.ProjectID,
		WorkspaceID:   workspaceID,
	}
	f.CreatedByID = &userID
	if err := validateNumberRange(f.NumberMin, f.NumberMax); err != nil {
		return nil, err
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&f).Error; err != nil {
			return common.Internal("Failed to create custom field")
		}
		// Project pages only list workspace fields their project is enrolled in.
		if f.ProjectID == nil {
			if err := tx.Exec(`INSERT INTO project_custom_field_enrollments (project_id, field_id, is_enabled)
				SELECT id, ?, true FROM projects WHERE workspace_id = ? AND deleted_at IS NULL
				ON CONFLICT (project_id, field_id) DO NOTHING`, f.ID, workspaceID).Error; err != nil {
				return common.Internal("Failed to enroll custom field")
			}
		}
		if f.FieldType == "dropdown" {
			return syncFieldOptions(tx, f.ID, req.Options)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.buildResponse(f), nil
}

func validateNumberRange(min, max *float64) error {
	if min != nil && max != nil && *min > *max {
		return common.Validation("number_min must not exceed number_max")
	}
	return nil
}

// syncFieldOptions makes the field's option list match opts. Options with an
// ID are updated in place; renaming one rewrites issue values that used the
// old label, since values store the option label.
func syncFieldOptions(tx *gorm.DB, fieldID uint64, opts []request.CustomFieldOptionInput) error {
	var existing []model.CustomFieldOption
	if err := tx.Where("field_id = ?", fieldID).Find(&existing).Error; err != nil {
		return common.Internal("Failed to load options")
	}
	byID := make(map[uint64]model.CustomFieldOption, len(existing))
	for _, o := range existing {
		byID[o.ID] = o
	}

	seen := make(map[string]bool)
	keep := make(map[uint64]bool)
	for i, in := range opts {
		value := strings.TrimSpace(in.Value)
		if value == "" {
			continue
		}
		if seen[value] {
			return common.Validation(fmt.Sprintf("Duplicate option '%s'", value))
		}
		seen[value] = true
		seq := in.Sequence
		if seq == 0 {
			seq = i + 1
		}

		if in.ID != nil {
			if old, ok := byID[*in.ID]; ok {
				keep[old.ID] = true
				if old.Value != value {
					tx.Model(&model.IssueCustomFieldValue{}).
						Where("field_id = ? AND value = ?", fieldID, old.Value).
						Update("value", value)
				}
				if err := tx.Model(&model.CustomFieldOption{}).Where("id = ?", old.ID).
					Updates(map[string]interface{}{"value": value, "color": in.Color, "sequence": seq}).Error; err != nil {
					return common.Internal("Failed to update option")
				}
				continue
			}
		}
		opt := model.CustomFieldOption{FieldID: fieldID, Value: value, Color: in.Color, Sequence: seq}
		if err := tx.Create(&opt).Error; err != nil {
			return common.Internal("Failed to create option")
		}
		keep[opt.ID] = true
	}

	for _, o := range existing {
		if !keep[o.ID] {
			tx.Delete(&model.CustomFieldOption{}, o.ID)
		}
	}
	return nil
}

// attachUsage fills issue counts and bound type names for a list of fields.
func (s *CustomFieldService) attachUsage(fields []response.CustomFieldResponse) {
	if len(fields) == 0 {
		return
	}
	ids := make([]uint64, len(fields))
	index := make(map[uint64]int, len(fields))
	for i, f := range fields {
		ids[i] = f.ID
		index[f.ID] = i
	}

	var counts []struct {
		FieldID uint64
		N       int64
	}
	s.db.Model(&model.IssueCustomFieldValue{}).
		Select("field_id, COUNT(DISTINCT issue_id) AS n").
		Where("field_id IN ? AND value <> ''", ids).
		Group("field_id").Scan(&counts)
	for _, c := range counts {
		fields[index[c.FieldID]].IssueCount = c.N
	}

	var types []struct {
		FieldID uint64
		Name    string
	}
	s.db.Table("issue_type_fields").
		Select("DISTINCT issue_type_fields.field_id, issue_types.name").
		Joins("JOIN issue_types ON issue_types.id = issue_type_fields.type_id AND issue_types.deleted_at IS NULL").
		Where("issue_type_fields.field_id IN ?", ids).
		Order("issue_types.name").Scan(&types)
	for _, t := range types {
		i := index[t.FieldID]
		fields[i].TypeNames = append(fields[i].TypeNames, t.Name)
	}
}

func (s *CustomFieldService) List(workspaceID uint64, projectID *uint64, issueTypeID *uint64) ([]response.CustomFieldResponse, error) {
	query := s.db.Model(&model.CustomField{}).Where("workspace_id = ?", workspaceID)

	if projectID != nil {
		// Three-way union (workspace-type Import model coexists with legacy flows):
		//   1. Project-private fields (project_id = ?)
		//   2. Workspace-level fields explicitly enrolled by the project (legacy)
		//   3. Workspace-level fields attached to a type the project has imported
		//      via the workspace-type Import model (fields "follow" the type)
		query = query.Where(`project_id = ?
			OR (project_id IS NULL AND EXISTS (
				SELECT 1 FROM project_custom_field_enrollments
				WHERE project_custom_field_enrollments.field_id = custom_fields.id
				AND project_custom_field_enrollments.project_id = ?
				AND project_custom_field_enrollments.is_enabled = true
			))
			OR (project_id IS NULL AND EXISTS (
				SELECT 1 FROM issue_type_fields itf
				JOIN issue_type_imports iti ON iti.workspace_type_id = itf.type_id
				WHERE itf.field_id = custom_fields.id
				AND iti.project_id = ?
			))`, *projectID, *projectID, *projectID)
	} else {
		query = query.Where("project_id IS NULL")
	}

	if issueTypeID != nil {
		query = query.Joins("JOIN issue_type_fields ON issue_type_fields.field_id = custom_fields.id").
			Where("issue_type_fields.type_id = ?", *issueTypeID)
	}

	var fields []model.CustomField
	if err := query.Order("created_at").Find(&fields).Error; err != nil {
		return nil, common.Internal("Failed to list custom fields")
	}

	result := make([]response.CustomFieldResponse, len(fields))
	for i, f := range fields {
		result[i] = *s.buildResponse(f)
	}
	s.attachUsage(result)
	return result, nil
}

func (s *CustomFieldService) EnrollField(projectID, fieldID uint64) error {
	var field model.CustomField
	if err := s.db.First(&field, fieldID).Error; err != nil {
		return common.NotFound("Custom field not found")
	}

	if field.ProjectID != nil {
		return common.BadRequest("Cannot enroll project-level field")
	}

	var enrollment model.ProjectCustomFieldEnrollment
	if err := s.db.Where("project_id = ? AND field_id = ?", projectID, fieldID).First(&enrollment).Error; err == nil {
		enrollment.IsEnabled = true
		return s.db.Save(&enrollment).Error
	}

	enrollment = model.ProjectCustomFieldEnrollment{
		ProjectID: projectID,
		FieldID:   fieldID,
		IsEnabled: true,
	}
	return s.db.Create(&enrollment).Error
}

func (s *CustomFieldService) UnenrollField(projectID, fieldID uint64) error {
	var field model.CustomField
	if err := s.db.First(&field, fieldID).Error; err != nil {
		return common.NotFound("Custom field not found")
	}

	if field.ProjectID != nil {
		return common.BadRequest("Cannot unenroll project-level field")
	}

	var enrollment model.ProjectCustomFieldEnrollment
	if err := s.db.Where("project_id = ? AND field_id = ?", projectID, fieldID).First(&enrollment).Error; err != nil {
		return common.NotFound("Field not enrolled for this project")
	}

	enrollment.IsEnabled = false
	return s.db.Save(&enrollment).Error
}

func (s *CustomFieldService) ListWorkspaceFieldsWithEnrollment(workspaceID, projectID uint64) ([]map[string]interface{}, error) {
	var fields []model.CustomField
	if err := s.db.Where("workspace_id = ? AND project_id IS NULL", workspaceID).Order("created_at").Find(&fields).Error; err != nil {
		return nil, common.Internal("Failed to list workspace fields")
	}

	var enrollments []model.ProjectCustomFieldEnrollment
	if err := s.db.Where("project_id = ?", projectID).Find(&enrollments).Error; err != nil {
		return nil, common.Internal("Failed to list enrollments")
	}

	enrollmentMap := make(map[uint64]bool)
	for _, e := range enrollments {
		enrollmentMap[e.FieldID] = e.IsEnabled
	}

	result := make([]map[string]interface{}, len(fields))
	for i, f := range fields {
		resp := s.buildResponse(f)
		result[i] = map[string]interface{}{
			"field":      resp,
			"is_enabled": enrollmentMap[f.ID],
		}
	}
	return result, nil
}

func (s *CustomFieldService) Get(fieldID uint64) (*response.CustomFieldResponse, error) {
	var f model.CustomField
	if err := s.db.First(&f, fieldID).Error; err != nil {
		return nil, common.NotFound("Custom field not found")
	}
	return s.buildResponse(f), nil
}

func (s *CustomFieldService) Update(fieldID, userID uint64, req request.CustomFieldUpdate) (*response.CustomFieldResponse, error) {
	var f model.CustomField
	if err := s.db.First(&f, fieldID).Error; err != nil {
		return nil, common.NotFound("Custom field not found")
	}

	if req.Name != nil {
		f.Name = *req.Name
	}
	if req.Description != nil {
		f.Description = *req.Description
	}
	if req.FieldType != nil {
		if err := s.validateFieldType(*req.FieldType); err != nil {
			return nil, err
		}
		// If changing away from dropdown, warn but allow
		f.FieldType = *req.FieldType
	}
	if req.IsRequired != nil {
		f.IsRequired = *req.IsRequired
	}
	if req.DefaultValue != nil {
		f.DefaultValue = *req.DefaultValue
	}
	if req.Placeholder != nil {
		f.Placeholder = *req.Placeholder
	}
	if req.IsActive != nil {
		f.IsActive = *req.IsActive
	}
	if req.ProjectID != nil {
		f.ProjectID = req.ProjectID
	}
	if req.IsReadonly != nil {
		f.IsReadonly = *req.IsReadonly
	}
	if req.IsMultiSelect != nil {
		f.IsMultiSelect = *req.IsMultiSelect
	}
	if req.NumberMin != nil || req.NumberMax != nil {
		f.NumberMin = req.NumberMin
		f.NumberMax = req.NumberMax
	}
	if err := validateNumberRange(f.NumberMin, f.NumberMax); err != nil {
		return nil, err
	}

	f.UpdatedByID = &userID
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&f).Error; err != nil {
			return common.Internal("Failed to update custom field")
		}
		if req.Options != nil && f.FieldType == "dropdown" {
			return syncFieldOptions(tx, f.ID, *req.Options)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	resp := s.buildResponse(f)
	list := []response.CustomFieldResponse{*resp}
	s.attachUsage(list)
	return &list[0], nil
}

func (s *CustomFieldService) Delete(fieldID uint64) error {
	var f model.CustomField
	if err := s.db.First(&f, fieldID).Error; err != nil {
		return common.NotFound("Custom field not found")
	}

	// Clean up related data
	s.db.Where("field_id = ?", fieldID).Delete(&model.IssueTypeField{})
	s.db.Where("field_id = ?", fieldID).Delete(&model.IssueCustomFieldValue{})
	s.db.Where("field_id = ?", fieldID).Delete(&model.CustomFieldOption{})

	return s.db.Delete(&f).Error
}

// ==================== Options ====================

func (s *CustomFieldService) CreateOption(fieldID uint64, req request.CustomFieldOptionCreate) (*response.CustomFieldOptionResponse, error) {
	var f model.CustomField
	if err := s.db.First(&f, fieldID).Error; err != nil {
		return nil, common.NotFound("Custom field not found")
	}
	if f.FieldType != "dropdown" {
		return nil, common.BadRequest("Options can only be added to dropdown fields")
	}

	opt := model.CustomFieldOption{
		FieldID:  fieldID,
		Value:    req.Value,
		Color:    req.Color,
		Sequence: req.Sequence,
	}

	// Check for duplicate value
	var count int64
	s.db.Model(&model.CustomFieldOption{}).Where("field_id = ? AND value = ?", fieldID, req.Value).Count(&count)
	if count > 0 {
		return nil, common.Conflict("Option value already exists for this field")
	}

	if err := s.db.Create(&opt).Error; err != nil {
		return nil, common.Internal("Failed to create option")
	}

	return &response.CustomFieldOptionResponse{
		ID:       opt.ID,
		FieldID:  opt.FieldID,
		Value:    opt.Value,
		Color:    opt.Color,
		Sequence: opt.Sequence,
	}, nil
}

func (s *CustomFieldService) UpdateOption(fieldID, optionID uint64, req request.CustomFieldOptionUpdate) (*response.CustomFieldOptionResponse, error) {
	var opt model.CustomFieldOption
	if err := s.db.Where("id = ? AND field_id = ?", optionID, fieldID).First(&opt).Error; err != nil {
		return nil, common.NotFound("Option not found")
	}

	if req.Value != nil {
		opt.Value = *req.Value
	}
	if req.Color != nil {
		opt.Color = *req.Color
	}
	if req.Sequence != nil {
		opt.Sequence = *req.Sequence
	}

	if err := s.db.Save(&opt).Error; err != nil {
		return nil, common.Internal("Failed to update option")
	}

	return &response.CustomFieldOptionResponse{
		ID:       opt.ID,
		FieldID:  opt.FieldID,
		Value:    opt.Value,
		Color:    opt.Color,
		Sequence: opt.Sequence,
	}, nil
}

func (s *CustomFieldService) DeleteOption(fieldID, optionID uint64) error {
	result := s.db.Where("id = ? AND field_id = ?", optionID, fieldID).Delete(&model.CustomFieldOption{})
	if result.RowsAffected == 0 {
		return common.NotFound("Option not found")
	}
	return nil
}

// ==================== Issue Values ====================

func (s *CustomFieldService) SetIssueValue(issueID uint64, req request.IssueCustomFieldValueCreate) (*response.IssueCustomFieldValueResponse, error) {
	// Verify issue exists (preload links for conditional-field visibility)
	var issue model.Issue
	if err := s.db.Preload("AssigneeLinks").Preload("LabelLinks").First(&issue, issueID).Error; err != nil {
		return nil, common.NotFound("Issue not found")
	}

	// Verify field exists
	var field model.CustomField
	if err := s.db.First(&field, req.FieldID).Error; err != nil {
		return nil, common.NotFound("Custom field not found")
	}

	if !s.fieldBoundToIssueType(&issue, req.FieldID) {
		return nil, common.BadRequest("Custom field is not available for this issue type")
	}

	// Validate value against field type
	if err := s.validateFieldValue(&field, req.Value); err != nil {
		return nil, err
	}

	newVal := strings.TrimSpace(req.Value)
	if newVal == "" && issue.IssueTypeID != nil && *issue.IssueTypeID != 0 {
		required, name, err := s.isTypeFieldRequired(*issue.IssueTypeID, req.FieldID)
		if err != nil {
			return nil, err
		}
		if required {
			visible, err := s.isCustomFieldVisibleForIssue(&issue, req.FieldID)
			if err != nil {
				return nil, err
			}
			if visible {
				return nil, common.BadRequest("缺少必填字段: " + name)
			}
		}
	}

	// Read old value before upsert
	var old model.IssueCustomFieldValue
	oldExists := s.db.Where("issue_id = ? AND field_id = ?", issueID, req.FieldID).First(&old).Error == nil

	// Upsert: delete existing, then create
	s.db.Where("issue_id = ? AND field_id = ?", issueID, req.FieldID).Delete(&model.IssueCustomFieldValue{})
	v := model.IssueCustomFieldValue{
		IssueID: issueID,
		FieldID: req.FieldID,
		Value:   newVal,
	}

	if err := s.db.Create(&v).Error; err != nil {
		return nil, common.Internal("Failed to set custom field value")
	}

	// Record activity: "changed custom field X from A to B"
	oldStr := ""
	if oldExists {
		oldStr = old.Value
	}
	fieldLabel := field.Name
	s.db.Create(&model.IssueActivity{
		IssueID:  &issueID,
		Verb:     "updated",
		Field:    strPtr("custom_field"),
		OldValue: &oldStr,
		NewValue: &newVal,
		Comment:  &fieldLabel,
	})

	return &response.IssueCustomFieldValueResponse{
		IssueID:   issueID,
		FieldID:   req.FieldID,
		Value:     v.Value,
		FieldName: field.Name,
		FieldType: field.FieldType,
	}, nil
}

func (s *CustomFieldService) BulkSetIssueValues(req request.BulkCustomFieldValueUpdate) ([]response.IssueCustomFieldValueResponse, error) {
	var issue model.Issue
	if err := s.db.Preload("AssigneeLinks").Preload("LabelLinks").First(&issue, req.IssueID).Error; err != nil {
		return nil, common.NotFound("Issue not found")
	}

	// Last write wins per field; only bound fields participate (same as apply loop).
	finalVals := make(map[uint64]string)
	for _, item := range req.Values {
		var field model.CustomField
		if err := s.db.First(&field, item.FieldID).Error; err != nil {
			continue
		}
		if !s.fieldBoundToIssueType(&issue, item.FieldID) {
			continue
		}
		finalVals[item.FieldID] = strings.TrimSpace(item.Value)
	}
	if issue.IssueTypeID != nil && *issue.IssueTypeID != 0 {
		var missing []string
		seen := make(map[string]bool)
		for fieldID, v := range finalVals {
			if v != "" {
				continue
			}
			required, name, err := s.isTypeFieldRequired(*issue.IssueTypeID, fieldID)
			if err != nil {
				return nil, err
			}
			if !required {
				continue
			}
			visible, err := s.isCustomFieldVisibleForIssue(&issue, fieldID)
			if err != nil {
				return nil, err
			}
			if visible && !seen[name] {
				seen[name] = true
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			sort.Strings(missing)
			return nil, common.BadRequest("缺少必填字段: " + strings.Join(missing, "、"))
		}
	}

	result := make([]response.IssueCustomFieldValueResponse, 0, len(req.Values))
	for _, item := range req.Values {
		// Validate field exists and is bound to this issue's type
		var field model.CustomField
		if err := s.db.First(&field, item.FieldID).Error; err != nil {
			continue // skip invalid field IDs
		}
		if !s.fieldBoundToIssueType(&issue, item.FieldID) {
			continue
		}

		// Upsert
		s.db.Where("issue_id = ? AND field_id = ?", req.IssueID, item.FieldID).Delete(&model.IssueCustomFieldValue{})
		v := model.IssueCustomFieldValue{
			IssueID: req.IssueID,
			FieldID: item.FieldID,
			Value:   strings.TrimSpace(item.Value),
		}
		if err := s.db.Create(&v).Error; err != nil {
			continue
		}

		result = append(result, response.IssueCustomFieldValueResponse{
			IssueID:   req.IssueID,
			FieldID:   item.FieldID,
			Value:     v.Value,
			FieldName: field.Name,
			FieldType: field.FieldType,
		})
	}

	return result, nil
}

func (s *CustomFieldService) ListIssueValues(issueID uint64) ([]response.IssueCustomFieldValueResponse, error) {
	var issue model.Issue
	if err := s.db.First(&issue, issueID).Error; err != nil {
		return nil, common.NotFound("Issue not found")
	}

	var values []model.IssueCustomFieldValue
	if err := s.db.Preload("Field").Where("issue_id = ?", issueID).Find(&values).Error; err != nil {
		return nil, common.Internal("Failed to list custom field values")
	}

	result := make([]response.IssueCustomFieldValueResponse, len(values))
	for i, v := range values {
		result[i] = response.IssueCustomFieldValueResponse{
			IssueID:   v.IssueID,
			FieldID:   v.FieldID,
			Value:     v.Value,
			FieldName: v.Field.Name,
			FieldType: v.Field.FieldType,
		}
	}
	return result, nil
}

func (s *CustomFieldService) UpdateIssueValue(issueID, fieldID uint64, val string) (*response.IssueCustomFieldValueResponse, error) {
	var issue model.Issue
	if err := s.db.Preload("AssigneeLinks").Preload("LabelLinks").First(&issue, issueID).Error; err != nil {
		return nil, common.NotFound("Issue not found")
	}

	var field model.CustomField
	if err := s.db.First(&field, fieldID).Error; err != nil {
		return nil, common.NotFound("Custom field not found")
	}

	if !s.fieldBoundToIssueType(&issue, fieldID) {
		return nil, common.BadRequest("Custom field is not available for this issue type")
	}

	if err := s.validateFieldValue(&field, val); err != nil {
		return nil, err
	}

	newVal := strings.TrimSpace(val)
	if newVal == "" && issue.IssueTypeID != nil && *issue.IssueTypeID != 0 {
		required, name, err := s.isTypeFieldRequired(*issue.IssueTypeID, fieldID)
		if err != nil {
			return nil, err
		}
		if required {
			visible, err := s.isCustomFieldVisibleForIssue(&issue, fieldID)
			if err != nil {
				return nil, err
			}
			if visible {
				return nil, common.BadRequest("缺少必填字段: " + name)
			}
		}
	}

	var v model.IssueCustomFieldValue
	result := s.db.Where("issue_id = ? AND field_id = ?", issueID, fieldID).First(&v)
	if result.Error != nil {
		// Create if not exists
		v = model.IssueCustomFieldValue{
			IssueID: issueID,
			FieldID: fieldID,
			Value:   newVal,
		}
		if err := s.db.Create(&v).Error; err != nil {
			return nil, common.Internal("Failed to create custom field value")
		}
	} else {
		v.Value = newVal
		if err := s.db.Save(&v).Error; err != nil {
			return nil, common.Internal("Failed to update custom field value")
		}
	}

	return &response.IssueCustomFieldValueResponse{
		IssueID:   issueID,
		FieldID:   fieldID,
		Value:     v.Value,
		FieldName: field.Name,
		FieldType: field.FieldType,
	}, nil
}

func (s *CustomFieldService) DeleteIssueValue(issueID, fieldID uint64) error {
	var issue model.Issue
	if err := s.db.Preload("AssigneeLinks").Preload("LabelLinks").First(&issue, issueID).Error; err != nil {
		return common.NotFound("Issue not found")
	}
	if !s.fieldBoundToIssueType(&issue, fieldID) {
		return common.BadRequest("Custom field is not available for this issue type")
	}
	if issue.IssueTypeID != nil && *issue.IssueTypeID != 0 {
		required, name, err := s.isTypeFieldRequired(*issue.IssueTypeID, fieldID)
		if err != nil {
			return err
		}
		if required {
			visible, err := s.isCustomFieldVisibleForIssue(&issue, fieldID)
			if err != nil {
				return err
			}
			if visible {
				return common.BadRequest("缺少必填字段: " + name)
			}
		}
	}

	result := s.db.Where("issue_id = ? AND field_id = ?", issueID, fieldID).Delete(&model.IssueCustomFieldValue{})
	if result.RowsAffected == 0 {
		return common.NotFound("Custom field value not found")
	}
	return nil
}

func (s *CustomFieldService) GetIssueFieldsWithValues(issueID uint64) (*response.IssueCustomFieldsResponse, error) {
	var issue model.Issue
	if err := s.db.First(&issue, issueID).Error; err != nil {
		return nil, common.NotFound("Issue not found")
	}

	resp := &response.IssueCustomFieldsResponse{
		IssueID: issueID,
		Fields:  make([]response.FieldWithValue, 0),
	}

	// Parity with Issue Create: only fields linked to the issue's type via
	// issue_type_fields (fields follow the type; no enrollment filter).
	if issue.IssueTypeID == nil || *issue.IssueTypeID == 0 {
		return resp, nil
	}

	var links []model.IssueTypeField
	if err := s.db.Preload("Field").
		Where("type_id = ?", *issue.IssueTypeID).
		Order("sequence").
		Find(&links).Error; err != nil {
		return nil, common.Internal("Failed to list type fields")
	}

	valueMap := make(map[uint64]string)
	var values []model.IssueCustomFieldValue
	s.db.Where("issue_id = ?", issueID).Find(&values)
	for _, v := range values {
		valueMap[v.FieldID] = v.Value
	}

	for _, link := range links {
		f := link.Field
		if f.ID == 0 || !f.IsActive {
			continue
		}
		cr := s.buildResponse(f)
		// Prefer binding-level required flag (same as create form).
		cr.IsRequired = link.IsRequired
		fwv := response.FieldWithValue{
			CustomFieldResponse: *cr,
			Value:               valueMap[f.ID],
		}
		resp.Fields = append(resp.Fields, fwv)
	}

	return resp, nil
}

// isCustomFieldVisibleForIssue reports whether fieldID is visible for the issue's current context
// (conditional hide rules). When false, required-field clears are allowed on the custom-field API.
func (s *CustomFieldService) isCustomFieldVisibleForIssue(issue *model.Issue, fieldID uint64) (bool, error) {
	if issue == nil {
		return true, nil
	}
	condSvc := NewConditionalFieldService(s.db)
	assigneeIDs := make([]uint64, 0, len(issue.AssigneeLinks))
	for _, link := range issue.AssigneeLinks {
		assigneeIDs = append(assigneeIDs, link.UserID)
	}
	labelIDs := make([]uint64, 0, len(issue.LabelLinks))
	for _, link := range issue.LabelLinks {
		labelIDs = append(labelIDs, link.LabelID)
	}
	visibleFieldIDs, err := condSvc.EvaluateFieldVisibility(issue.WorkspaceID, issue.IssueTypeID, issue.StateID, issue.Priority, assigneeIDs, labelIDs)
	if err != nil {
		return false, err
	}
	for _, id := range visibleFieldIDs {
		if id == fieldID {
			return true, nil
		}
	}
	return false, nil
}

// isTypeFieldRequired reports whether fieldID is required on the given issue type (binding is authoritative).
func (s *CustomFieldService) isTypeFieldRequired(typeID, fieldID uint64) (bool, string, error) {
	var row struct {
		IsRequired bool
		Name       string
	}
	err := s.db.Table("issue_type_fields").
		Select("issue_type_fields.is_required AS is_required, custom_fields.name AS name").
		Joins("JOIN custom_fields ON custom_fields.id = issue_type_fields.field_id").
		Where("issue_type_fields.type_id = ? AND issue_type_fields.field_id = ?", typeID, fieldID).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, "", nil
		}
		return false, "", common.Internal("Failed to load type field binding")
	}
	return row.IsRequired, row.Name, nil
}

// fieldBoundToIssueType reports whether fieldID is attached to the issue's type.
func (s *CustomFieldService) fieldBoundToIssueType(issue *model.Issue, fieldID uint64) bool {
	if issue == nil || issue.IssueTypeID == nil || *issue.IssueTypeID == 0 {
		return false
	}
	var count int64
	s.db.Model(&model.IssueTypeField{}).
		Where("type_id = ? AND field_id = ?", *issue.IssueTypeID, fieldID).
		Count(&count)
	return count > 0
}
