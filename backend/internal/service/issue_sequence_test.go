package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/reqmango/backend/internal/dto/request"
	"github.com/reqmango/backend/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func sequenceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL sequence regression tests")
	}
	config := &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)}
	admin, err := gorm.Open(postgres.Open(dsn), config)
	require.NoError(t, err)
	adminSQL, err := admin.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, adminSQL.Close()) })
	schema := "sequence_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, admin.Exec("CREATE SCHEMA "+schema).Error)
	t.Cleanup(func() { require.NoError(t, admin.Exec("DROP SCHEMA "+schema+" CASCADE").Error) })
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Contains(t, []string{"postgres", "postgresql"}, parsed.Scheme, "TEST_DATABASE_URL must be a PostgreSQL URL")
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(parsed.String()), config)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(20)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })
	require.NoError(t, db.AutoMigrate(&model.Issue{}, &model.ProjectIntakeSetting{}, &model.Webhook{},
		&model.CustomField{}, &model.ConditionalField{}, &model.IssueType{}, &model.IssueTypeImport{},
		&model.IssueTypeField{}, &model.IssueActivity{}, &model.IssueAssignee{}, &model.IssueLabel{},
		&model.IssueCycle{}, &model.ModuleIssue{}, &model.ReleaseIssue{}))
	return db
}

func sequenceTestProject(t *testing.T, db *gorm.DB, identifier string) model.Project {
	t.Helper()
	project := model.Project{Name: "Sequence regression", Identifier: identifier, WorkspaceID: 1}
	require.NoError(t, db.Create(&project).Error)
	require.NoError(t, db.Create(&model.State{Name: "Backlog", Group: "backlog", IsDefault: true, ProjectID: &project.ID, WorkspaceID: 1}).Error)
	require.NoError(t, db.Create(&model.ProjectIntakeSetting{ProjectID: project.ID, FormEnabled: true, Token: uuid.NewString()}).Error)
	return project
}

func TestIssueSequenceConcurrentCreation(t *testing.T) {
	db := sequenceTestDB(t)
	// Keep inserts overlapping so the old unlocked MAX+1 fails reliably.
	require.NoError(t, db.Exec(`CREATE FUNCTION slow_issue_insert() RETURNS trigger AS $$
	BEGIN PERFORM pg_sleep(0.05); RETURN NEW; END;
	$$ LANGUAGE plpgsql`).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER slow_issue_insert BEFORE INSERT ON issues
	FOR EACH ROW EXECUTE FUNCTION slow_issue_insert()`).Error)
	issueSvc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	intakeSvc := NewIntakeService(db, nil)
	for _, mode := range []string{"issue", "intake", "mixed"} {
		t.Run(mode, func(t *testing.T) {
			project := sequenceTestProject(t, db, strings.ToUpper(mode))
			const workers = 12
			start := make(chan struct{})
			sequences := make([]int, workers)
			errors := make([]error, workers)
			var wg sync.WaitGroup
			for i := 0; i < workers; i++ {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					<-start
					name := fmt.Sprintf("Concurrent %s %d", mode, i)
					if mode == "intake" || mode == "mixed" && i%2 == 0 {
						issue, err := intakeSvc.Submit(project.ID, "form", IntakeSubmission{Name: name})
						errors[i] = err
						if issue != nil {
							sequences[i] = issue.SequenceID
						}
					} else {
						issue, err := issueSvc.Create(&request.IssueCreateRequest{Name: name}, project.ID, 1, 1)
						errors[i] = err
						if issue != nil {
							sequences[i] = issue.SequenceID
						}
					}
				}(i)
			}
			close(start)
			wg.Wait()
			for _, err := range errors {
				require.NoError(t, err)
			}
			sort.Ints(sequences)
			for i, seq := range sequences {
				require.Equal(t, i+1, seq, "duplicate or missing sequence: %v", sequences)
			}
			var stored []int
			require.NoError(t, db.Model(&model.Issue{}).Where("project_id = ?", project.ID).Order("sequence_id").Pluck("sequence_id", &stored).Error)
			require.Equal(t, sequences, stored)
		})
	}
}

func TestIssueSequenceDoesNotReuseDeletedNumbers(t *testing.T) {
	db := sequenceTestDB(t)
	project := sequenceTestProject(t, db, "DELETED")
	intakeSvc := NewIntakeService(db, nil)
	issueSvc := NewIssueService(db, nil, NewWebhookService(db), nil, nil)
	for i, submit := range []bool{true, false} {
		highest := 50 + i*10
		latest := model.Issue{Name: "Deleted highest number", ProjectID: project.ID, WorkspaceID: 1, StateID: 1, SequenceID: highest}
		require.NoError(t, db.Create(&latest).Error)
		require.NoError(t, db.Delete(&latest).Error)
		if submit {
			issue, err := intakeSvc.Submit(project.ID, "form", IntakeSubmission{Name: "Next intake"})
			require.NoError(t, err)
			require.Equal(t, highest+1, issue.SequenceID)
		} else {
			issue, err := issueSvc.Create(&request.IssueCreateRequest{Name: "Next issue"}, project.ID, 1, 1)
			require.NoError(t, err)
			require.Equal(t, highest+1, issue.SequenceID)
		}
	}
	other := sequenceTestProject(t, db, "OTHER")
	issue, err := intakeSvc.Submit(other.ID, "form", IntakeSubmission{Name: "Independent numbering"})
	require.NoError(t, err)
	require.Equal(t, 1, issue.SequenceID)
}

func TestIssueSequenceRollbackAndProjectIsolation(t *testing.T) {
	db := sequenceTestDB(t)
	project := sequenceTestProject(t, db, "ROLLBACK")
	other := sequenceTestProject(t, db, "UNLOCKED")
	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { tx.Rollback() })
	seq, err := nextIssueSequence(tx, project.ID)
	require.NoError(t, err)
	require.Equal(t, 1, seq)
	require.NoError(t, tx.Create(&model.Issue{Name: "Rolled back", ProjectID: project.ID, WorkspaceID: 1, StateID: 1, SequenceID: seq}).Error)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	intakeSvc := NewIntakeService(db.WithContext(ctx), nil)
	independent, err := intakeSvc.Submit(other.ID, "form", IntakeSubmission{Name: "Not blocked by another project"})
	require.NoError(t, err)
	require.Equal(t, 1, independent.SequenceID)
	require.NoError(t, tx.Rollback().Error)
	issue, err := intakeSvc.Submit(project.ID, "form", IntakeSubmission{Name: "After rollback"})
	require.NoError(t, err)
	require.Equal(t, 1, issue.SequenceID)
}

func TestIssueSequenceAllocationErrors(t *testing.T) {
	db := sequenceTestDB(t)
	project := sequenceTestProject(t, db, "ERRORS")
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		seq, err := nextIssueSequence(tx, project.ID+1)
		require.ErrorIs(t, err, gorm.ErrRecordNotFound)
		require.Zero(t, seq)
		return nil
	}))
	require.NoError(t, db.Exec("DROP TABLE issues CASCADE").Error)
	require.Error(t, db.Transaction(func(tx *gorm.DB) error {
		seq, err := nextIssueSequence(tx, project.ID)
		require.Error(t, err)
		require.Zero(t, seq)
		return err
	}))
}
