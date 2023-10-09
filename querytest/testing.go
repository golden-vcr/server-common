package querytest

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	impl "github.com/golden-vcr/server-common/querytest/internal"

	_ "github.com/lib/pq"
)

// Prepare returns a sql.DB configured so that we can run unit tests on database queries
// for the current project, if and only if a properly-configured querytest database
// container is running in docker. If no such database container is running, skips the
// test with a descriptive message.
//
// To start a test database, run the following from anywhere in your project's repo:
//
// - go run github.com/golden-vcr/server-common/querytest/cmd
//
// See querytest/cmd/main.go for more details.
func Prepare(t *testing.T) *sql.DB {
	uri := resolvePostgresUri(t)
	db, err := sql.Open("postgres", uri)
	if err != nil {
		t.Fatalf("sql.Open failed with querytest uri %s: %v", uri, err)
	}
	return db
}

// PrepareTx prepares a sql.DB via Prepare, then initializes a database transaction
// which will be automatically rolled back when the test is done
func PrepareTx(t *testing.T) *sql.Tx {
	db := Prepare(t)
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("db.Begin failed: %v", err)
	}
	t.Cleanup(func() {
		if err := tx.Rollback(); err != nil {
			t.Logf("failed to roll back transaction created via PrepareTx: %v", err)
		}
	})
	return tx
}

// resolvePostgresUri uses the docker CLI to check for a postgres container that's been
// configured for us to run query tests against in the current project: if found,
// returns a 'postgres://...' connection string that will allow us to connect to that
// DB; otherwise skips the test.
func resolvePostgresUri(t *testing.T) string {
	// Figure out what our container should be called
	rootDir, err := impl.FindProjectRootDir()
	if err != nil {
		t.Fatalf("unable to resolve root directory for project that uses querytest: %v", err)
	}
	projectName := impl.GetProjectName(rootDir)
	containerName := impl.GetContainerName(projectName)

	// If we don't have docker installed, we can't check for a container (and we can
	// assume it's not running): skip the test with a warning
	containerIsRunning := false
	hasDocker := impl.IsDockerInstalled(context.Background())
	if hasDocker {
		_, err := impl.FindContainerId(context.Background(), containerName)
		if err != nil && !errors.Is(err, impl.ErrNoSuchContainer) {
			t.Fatalf("unable to check querytest container status: %v", err)
		}
		containerIsRunning = err == nil
	}

	// If we don't have a querytest container running for this project, skip the test
	// with a non-fatal warning
	if !containerIsRunning {
		prefix := ""
		if !hasDocker {
			prefix = "install docker and "
		}
		t.Skipf("%srun 'go run github.com/golden-vcr/server-common/querytest/cmd' to start up a querytest db and enable query tests to run", prefix)
	}

	// Otherwise, we should be good to run database query tests: return the correct
	// postgres URI that will initialize a connection to the postgres server running in
	// this container
	return impl.GetPostgresUri(projectName)
}
