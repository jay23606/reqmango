package request

// IssueTypeCreate request for creating a new issue type.
type IssueTypeCreate struct {
	Name                string   `json:"name" binding:"required"`
	Color               string   `json:"color"`
	Icon                string   `json:"icon"`
	Description         string   `json:"description"`
	Level               int      `json:"level"`          // hierarchy level 0-5
	ParentTypeID        *uint64  `json:"parent_type_id"` // parent type in hierarchy
	AllowedChildTypeIDs []uint64 `json:"allowed_child_type_ids"`
	IsDefault           bool     `json:"is_default"`
	Sequence            int      `json:"sequence"`
	ProjectID           *uint64  `json:"project_id"` // nil = workspace-shared
}

// IssueTypeUpdate request for updating an issue type.
type IssueTypeUpdate struct {
	Name                *string  `json:"name"`
	Color               *string  `json:"color"`
	Icon                *string  `json:"icon"`
	Description         *string  `json:"description"`
	Level               *int     `json:"level"`
	ParentTypeID        *uint64  `json:"parent_type_id"`
	AllowedChildTypeIDs []uint64 `json:"allowed_child_type_ids"`
	IsDefault           *bool    `json:"is_default"`
	Sequence            *int     `json:"sequence"`
	IsActive            *bool    `json:"is_active"`
	ProjectID           *uint64  `json:"project_id"`
}

// IssueTypeDisable request for toggling active status.
type IssueTypeDisable struct {
	IsActive bool `json:"is_active"`
}

// IssueTypeFieldCreate request for associating a custom field to an issue type.
type IssueTypeFieldCreate struct {
	FieldID    uint64 `json:"field_id" binding:"required"`
	IsRequired bool   `json:"is_required"`
	Sequence   int    `json:"sequence"`
}

// IssueTypeFieldUpdate request for updating a field association.
type IssueTypeFieldUpdate struct {
	IsRequired *bool `json:"is_required"`
	Sequence   *int  `json:"sequence"`
}
