/*
The querytest command manages a throwaway Postgres server running in Docker, configured
so that we can run unit tests for database queries against a live Postgres database with
migrations pre-applied.

Usage:

	go run github.com/golden-vcr/server-common/querytest/cmd [command]

Commands:

	up (default) | Ensures that a postgres server is running for this project
	down         | Shuts down any existing server for this project, if running
	restart      | Shuts down any existing server, then starts a new one

It's expected that this command will only be run within the repository for a Golden VCR
backend application. We make a few assumptions about the structure of such a project:

  - It must be a Go project (with a go.mod file in the root directory)
  - It must contain a db-migrate.sh script, in the root directory, which runs database
    migrations against a postgres server configured via PGHOST, PGPORT, PGDATABASE,
    PGUSER, PGPASSWORD, and PGSSLMODE

Each project is assigned its own postgres container, named 'querytest-<project-name>',
each postgres container has its own port number on the host machine, derived from a hash
of the project name.

Test functions for database queries can be written like so:

	func Test_foo(t *testing.T) {
		// PrepareTx will skip the test if our test db isn't running in docker
		tx := querytest.PrepareTx(t)
		q := queries.New(tx)

		// Run one of our sqlc-compiled database queries against our test DB
		err := queries.RecordSomething(context.Background())
		assert.NoError(t, err)

		// Verify that the new state (in our pending transaction) is what we expect
		querytest.AssertCount(t, tx, 1, "SELECT COUNT(*) FROM something")
	}

Once a querytest database has been started for the relevant project, tests that use
querytest.Prepare (or querytest.PrepareTx) will run against the querytest database for
the project. If the database is not running, those tests will be skipped.
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	impl "github.com/golden-vcr/server-common/querytest/internal"
)

func main() {
	// Parse an optional command to determine the desired final state of our postgres
	// container (running, shut down, or running from a fresh boot)
	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	if command != "up" && command != "down" && command != "restart" {
		log.Fatalf("Unknown command '%s' (expected up|down|restart)", command)
	}

	// Terminate on SIGINT etc.
	ctx, close := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill, syscall.SIGTERM)
	defer close()

	// This program uses the docker CLI to manage a postgres container; abort if
	// 'docker -v' fails
	if !impl.IsDockerInstalled(ctx) {
		log.Fatalf("docker is not installed")
	}

	// Infer the root directory of the project from our current working directory
	rootDir, err := impl.FindProjectRootDir()
	if err != nil {
		log.Fatalf("failed to find project root dir: %v", err)
	}

	// Each project has its own container name and host port; resolve and print the
	// details for the current project
	projectName := impl.GetProjectName(rootDir)
	containerName := impl.GetContainerName(projectName)
	postgresHostPort := impl.GetPostgresHostPort(projectName)
	fmt.Printf("Project:        %s\n", projectName)
	fmt.Printf("Container name: %s\n", containerName)
	fmt.Printf("Host port:      %d\n", postgresHostPort)

	// Check to see if we already have a postgres container running
	containerId, err := impl.FindContainerId(ctx, containerName)
	if err != nil && !errors.Is(err, impl.ErrNoSuchContainer) {
		log.Fatalf(err.Error())
	}
	if err == nil {
		// If the container is running: there's nothing to do for 'up'; otherwise we can
		// proceed with stopping the container to satisfy 'down' or 'restart'
		fmt.Printf("Container ID:   %s\n", containerId)
		if command == "up" {
			fmt.Printf("\n%s\n", impl.GetPostgresUri(projectName))
			os.Exit(0)
		}
		if command == "down" || command == "restart" {
			if err := impl.StopContainer(ctx, containerId); err != nil {
				log.Fatalf("failed to stop container %s: %v", containerId, err)
			}
			fmt.Printf("Container stopped.\n")

			// If the command is 'down', we're done; otherwise we can proceed with
			// starting a new container to satsify 'restart'
			if command == "down" {
				os.Exit(0)
			}
		}
	} else {
		// If the container isn't running: there's nothing to do for 'down'; otherwise
		// we can proceed with starting a container to satisfy 'up' or 'restart'
		fmt.Printf("Container is not running.\n")
		if command == "down" {
			os.Exit(0)
		}
	}

	// Sanity-check: if we're still running, our command should be 'up' or 'restart', as
	// we're about to spin up a brand new docker container running our postgres image
	if command != "up" && command != "restart" {
		panic(fmt.Sprintf("unexpected command '%s'", command))
	}

	// Start up a new docker container running postgres, configured appropriately for
	// this project
	envExports := []string{fmt.Sprintf("POSTGRES_PASSWORD=%s", impl.PostgresPassword)}
	portMappings := []string{fmt.Sprintf("%d:5432", postgresHostPort)}
	containerId, err = impl.StartContainer(ctx, containerName, impl.PostgresImage, envExports, nil, portMappings)
	if err != nil {
		log.Fatalf("failed to start postgres container: %v", err)
	}
	fmt.Printf("Container ID:   %s\n\n", containerId)

	// Tail log output from the new container until we can verify that it's ready to
	// accept connections
	tailCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	tailResult := make(chan error)
	lines := make(chan string)
	go func() {
		tailResult <- impl.TailContainerOutput(tailCtx, containerId, lines)
	}()
	done := false
	for !done {
		select {
		case err := <-tailResult:
			if err == nil {
				log.Fatalf("container output stopped before database became ready")
			} else {
				log.Fatalf("database did not become ready: %v", err)
			}
		case line := <-lines:
			if strings.Contains(line, "database system is ready to accept connections") {
				done = true
				cancel()
			}
		}
	}

	// Run our project's db-migrate.sh script
	if err := impl.RunMigrations(ctx, rootDir); err != nil {
		log.Fatalf("failed to apply database migrations: %v", err)
	}
	fmt.Printf("\n%s\n", impl.GetPostgresUri(projectName))
}
