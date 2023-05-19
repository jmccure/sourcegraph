package background

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/derision-test/glock"
	"github.com/keegancsmith/sqlf"
	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/sourcegraph/internal/database"
	"github.com/sourcegraph/sourcegraph/internal/database/basestore"
	"github.com/sourcegraph/sourcegraph/internal/database/dbtest"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

func TestOwnRepoIndexSchedulerJob_JobsAutoIndex(t *testing.T) {
	obsCtx := observation.TestContextTB(t)
	logger := obsCtx.Logger
	db := database.NewDB(logger, dbtest.NewDB(logger, t))
	ctx := context.Background()

	insertRepo(t, db, 500, "great-repo-1", true)
	insertRepo(t, db, 501, "great-repo-2", true)
	insertRepo(t, db, 502, "great-repo-3", true)
	insertRepo(t, db, 503, "great-repo-4", false)

	verifyCount := func(t *testing.T, signaName string, expected int) {
		store := basestore.NewWithHandle(db.Handle())
		// Check that correct rows were added to own_background_jobs

		count, _, err := basestore.ScanFirstInt(store.Query(ctx, sqlf.Sprintf("SELECT COUNT(*) FROM own_background_jobs WHERE job_type = (select id from own_signal_configurations where name = %s)", signaName)))
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, expected, count)
	}

	for _, jobType := range QueuePerRepoIndexJobs {
		t.Run(jobType.Name, func(t *testing.T) {
			ctx := context.Background()

			_, err := db.FeatureFlags().CreateBool(ctx, featureFlagName(jobType), true)
			if err != nil {
				t.Fatal(err)
			}

			job := newOwnRepoIndexSchedulerJob(db, jobType, logger)
			err = job.Handle(ctx)
			if err != nil {
				t.Fatal(err)
			}

			verifyCount(t, job.jobType.Name, 3)
		})
	}
}

func TestOwnRepoIndexSchedulerJob_JobsAreExcluded(t *testing.T) {
	obsCtx := observation.TestContextTB(t)
	logger := obsCtx.Logger
	db := database.NewDB(logger, dbtest.NewDB(logger, t))
	ctx := context.Background()

	verifyCount := func(t *testing.T, config database.SignalConfiguration, expected int) {
		store := basestore.NewWithHandle(db.Handle())
		// Check that correct rows were added to own_background_jobs

		count, _, err := basestore.ScanFirstInt(store.Query(context.Background(), sqlf.Sprintf("SELECT COUNT(*) FROM own_background_jobs WHERE job_type = %s", config.ID)))
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, expected, count)
	}
	jobType := IndexJobType{
		Name:          "recent-contributors",
		IndexInterval: time.Hour * 24,
	}

	config, err := loadConfig(ctx, jobType, db.OwnSignalConfigurations())
	require.NoError(t, err)

	err = db.OwnSignalConfigurations().UpdateConfiguration(ctx, database.UpdateSignalConfigurationArgs{
		Name:                 config.Name,
		Enabled:              true,
		ExcludedRepoPatterns: []string{"excluded-repo-1", "excluded-repo-2"},
	})
	require.NoError(t, err)

	clock := glock.NewMockClockAt(time.Now())

	halfInterval := clock.Now().UTC().Add(-1 * jobType.IndexInterval / 2)
	doubleInterval := clock.Now().UTC().Add(-1 * jobType.IndexInterval * 2)
	states := []string{"queued", "processing", "errored", "failed", "completed"}

	insertRepo(t, db, 500, "great-repo-1", true)
	insertRepo(t, db, 501, "excluded-repo-1", true)
	insertRepo(t, db, 502, "excluded-repo-2", true)

	for _, state := range states {
		doTest := func(t *testing.T, expected int) {
			defer clearJobs(t, db)
			job := newOwnRepoIndexSchedulerJob(db, jobType, logger)
			job.clock = clock
			err := job.Handle(ctx)
			if err != nil {
				t.Fatal(err)
			}
			verifyCount(t, config, expected)
		}

		t.Run(state+" half interval", func(t *testing.T) {
			insertJob(t, db, 500, config, state, halfInterval)
			doTest(t, 1) // expecting 1 means no new jobs were inserted
		})

		t.Run(state+" double interval", func(t *testing.T) {
			insertJob(t, db, 500, config, state, doubleInterval)
			expected := 1
			if state == "completed" || state == "failed" {
				expected = 2
				// only for completed / failed records do we retry, but only after 1 full interval
			}
			doTest(t, expected)
		})
	}
}

func insertRepo(t *testing.T, db database.DB, id int, name string, cloned bool) {
	if name == "" {
		name = fmt.Sprintf("n-%d", id)
	}

	deletedAt := sqlf.Sprintf("NULL")
	if strings.HasPrefix(name, "DELETED-") {
		deletedAt = sqlf.Sprintf("%s", time.Unix(1587396557, 0).UTC())
	}
	insertRepoQuery := sqlf.Sprintf(
		`INSERT INTO repo (id, name, deleted_at, private) VALUES (%s, %s, %s, %s) ON CONFLICT (id) DO NOTHING`,
		id,
		name,
		deletedAt,
		false,
	)
	if _, err := db.ExecContext(context.Background(), insertRepoQuery.Query(sqlf.PostgresBindVar), insertRepoQuery.Args()...); err != nil {
		t.Fatalf("unexpected error while upserting repository: %s", err)
	}

	status := "cloned"
	if strings.HasPrefix(name, "DELETED-") || !cloned {
		status = "not_cloned"
	}
	updateGitserverRepoQuery := sqlf.Sprintf(
		`UPDATE gitserver_repos SET clone_status = %s WHERE repo_id = %s`,
		status,
		id,
	)
	if _, err := db.ExecContext(context.Background(), updateGitserverRepoQuery.Query(sqlf.PostgresBindVar), updateGitserverRepoQuery.Args()...); err != nil {
		t.Fatalf("unexpected error while upserting gitserver repository: %s", err)
	}
}

func insertJob(t *testing.T, db database.DB, repoId int, config database.SignalConfiguration, state string, finishedAt time.Time) {
	q := sqlf.Sprintf("insert into own_background_jobs (repo_id, job_type, state, finished_at) values (%s, %s, %s, %s);", repoId, config.ID, state, finishedAt)
	if finishedAt.IsZero() {
		q = sqlf.Sprintf("insert into own_background_jobs (repo_id, job_type, state) values (%s, %s, %s);", repoId, config.ID, state)
	}
	if _, err := db.ExecContext(context.Background(), q.Query(sqlf.PostgresBindVar), q.Args()...); err != nil {
		t.Fatal(err)
	}
}

func clearJobs(t *testing.T, db database.DB) {
	if _, err := db.ExecContext(context.Background(), "truncate own_background_jobs;"); err != nil {
		t.Fatal(err)
	}
}
