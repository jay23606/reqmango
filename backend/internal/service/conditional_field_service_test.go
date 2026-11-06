package service

import (
	"slices"
	"strconv"
	"testing"

	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
)

func TestConditionalField_CreateAndList(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.CustomField{}, &model.ConditionalField{}))

	wsID, projectID, _ := seedTestData(t, db)

	done := &model.State{Name: "Done", Color: "#22c55e", ProjectID: &projectID, WorkspaceID: wsID}
	require.NoError(t, db.Create(done).Error)

	estField := &model.CustomField{Name: "预估工时", FieldType: "number", WorkspaceID: wsID}
	require.NoError(t, db.Create(estField).Error)

	svc := NewConditionalFieldService(db)
	doneStr := strconv.FormatUint(done.ID, 10)

	created, err := svc.Create(&request.ConditionalFieldCreateRequest{
		WorkspaceID:     wsID,
		FieldID:         estField.ID,
		ConditionType:   "state",
		Operator:        "equals",
		ConditionValues: []string{doneStr},
		IsEnabled:       true,
		Priority:        0,
		Description:     "hide estimate when done",
	})
	require.NoError(t, err)
	require.NotZero(t, created.ID)
	require.Equal(t, wsID, created.WorkspaceID)
	require.Equal(t, estField.ID, created.FieldID)
	require.True(t, created.IsEnabled, "AC-01.1: rule is enabled after create")

	list, err := svc.List(wsID, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, created.ID, list[0].ID)
	require.True(t, list[0].IsEnabled)
	require.Equal(t, "state", list[0].ConditionType)
	require.Equal(t, []string{doneStr}, list[0].ConditionValues)
}

func TestEvaluateFieldVisibility_HidesMatchedField(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.CustomField{}, &model.ConditionalField{}))

	wsID, projectID, _ := seedTestData(t, db)

	var backlog model.State
	require.NoError(t, db.Where("project_id = ? AND name = ?", projectID, "Backlog").First(&backlog).Error)

	done := &model.State{Name: "Done", Color: "#22c55e", ProjectID: &projectID, WorkspaceID: wsID}
	require.NoError(t, db.Create(done).Error)

	estField := &model.CustomField{Name: "预估工时", FieldType: "number", WorkspaceID: wsID}
	require.NoError(t, db.Create(estField).Error)

	svc := NewConditionalFieldService(db)
	doneStr := strconv.FormatUint(done.ID, 10)

	_, err := svc.Create(&request.ConditionalFieldCreateRequest{
		WorkspaceID:     wsID,
		FieldID:         estField.ID,
		ConditionType:   "state",
		Operator:        "equals",
		ConditionValues: []string{doneStr},
		IsEnabled:       true,
		Priority:        0,
	})
	require.NoError(t, err)

	// Done → field hidden (AC-02.1)
	visibleDone, err := svc.EvaluateFieldVisibility(wsID, nil, done.ID, "", nil, nil)
	require.NoError(t, err)
	require.False(t, slices.Contains(visibleDone, estField.ID), "estimated hours must be hidden when state is Done")

	// Backlog → rule does not match → field visible
	visibleBacklog, err := svc.EvaluateFieldVisibility(wsID, nil, backlog.ID, "", nil, nil)
	require.NoError(t, err)
	require.True(t, slices.Contains(visibleBacklog, estField.ID))
}

func TestEvaluateFieldVisibility_DisabledRuleShowsField(t *testing.T) {
	t.Run("disabled", func(t *testing.T) {
		db := setupTestDB(t)
		require.NoError(t, db.AutoMigrate(&model.CustomField{}, &model.ConditionalField{}))

		wsID, projectID, _ := seedTestData(t, db)

		done := &model.State{Name: "Done", Color: "#22c55e", ProjectID: &projectID, WorkspaceID: wsID}
		require.NoError(t, db.Create(done).Error)

		estField := &model.CustomField{Name: "预估工时", FieldType: "number", WorkspaceID: wsID}
		require.NoError(t, db.Create(estField).Error)

		svc := NewConditionalFieldService(db)
		doneStr := strconv.FormatUint(done.ID, 10)

		created, err := svc.Create(&request.ConditionalFieldCreateRequest{
			WorkspaceID:     wsID,
			FieldID:         estField.ID,
			ConditionType:   "state",
			Operator:        "equals",
			ConditionValues: []string{doneStr},
			IsEnabled:       true,
			Priority:        0,
		})
		require.NoError(t, err)

		visibleBefore, err := svc.EvaluateFieldVisibility(wsID, nil, done.ID, "", nil, nil)
		require.NoError(t, err)
		require.False(t, slices.Contains(visibleBefore, estField.ID))

		disabled := false
		_, err = svc.Update(created.ID, wsID, &request.ConditionalFieldUpdateRequest{IsEnabled: &disabled})
		require.NoError(t, err)

		visibleAfter, err := svc.EvaluateFieldVisibility(wsID, nil, done.ID, "", nil, nil)
		require.NoError(t, err)
		require.True(t, slices.Contains(visibleAfter, estField.ID), "AC-05.1: disabled rule → field visible again")
	})

	t.Run("deleted", func(t *testing.T) {
		db := setupTestDB(t)
		require.NoError(t, db.AutoMigrate(&model.CustomField{}, &model.ConditionalField{}))

		wsID, projectID, _ := seedTestData(t, db)

		done := &model.State{Name: "Done", Color: "#22c55e", ProjectID: &projectID, WorkspaceID: wsID}
		require.NoError(t, db.Create(done).Error)

		estField := &model.CustomField{Name: "预估工时", FieldType: "number", WorkspaceID: wsID}
		require.NoError(t, db.Create(estField).Error)

		svc := NewConditionalFieldService(db)
		doneStr := strconv.FormatUint(done.ID, 10)

		created, err := svc.Create(&request.ConditionalFieldCreateRequest{
			WorkspaceID:     wsID,
			FieldID:         estField.ID,
			ConditionType:   "state",
			Operator:        "equals",
			ConditionValues: []string{doneStr},
			IsEnabled:       true,
			Priority:        0,
		})
		require.NoError(t, err)

		visibleBefore, err := svc.EvaluateFieldVisibility(wsID, nil, done.ID, "", nil, nil)
		require.NoError(t, err)
		require.False(t, slices.Contains(visibleBefore, estField.ID))

		require.NoError(t, svc.Delete(created.ID, wsID))

		visibleAfter, err := svc.EvaluateFieldVisibility(wsID, nil, done.ID, "", nil, nil)
		require.NoError(t, err)
		require.True(t, slices.Contains(visibleAfter, estField.ID), "AC-05.1: deleted rule → field visible again")
	})
}
