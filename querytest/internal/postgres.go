package impl

import (
	"fmt"
	"hash/crc32"
)

const (
	PostgresImage       = "postgres:16"
	PostgresPassword    = "password"
	PostgresHostPortMin = 44000
	PostgresHostPortMax = 44999
)

// GetContainerName returns the canonical name for the docker container that runs a
// postgres container for this project's query tests
func GetContainerName(projectName string) string {
	return fmt.Sprintf("querytest-%s", projectName)
}

// GetPostgresHostPort returns an arbitrary but stable port number, representing a port
// on the host machine, that should be canonically used for the querytest database for
// the given project
func GetPostgresHostPort(projectName string) int {
	hash := crc32.NewIEEE()
	_, err := hash.Write([]byte(projectName))
	if err != nil {
		panic(err)
	}
	offset := int(hash.Sum32() % (PostgresHostPortMax - PostgresHostPortMin + 1))
	return PostgresHostPortMin + offset
}

// GetPostgresUri returns the 'postgres:' connection string that can be used to connect
// to the postgres server that's running in a container for the given project's query
// tests
func GetPostgresUri(projectName string) string {
	hostPort := GetPostgresHostPort(projectName)
	return fmt.Sprintf("postgres://postgres:%s@localhost:%d?sslmode=disable", PostgresPassword, hostPort)
}

// GetPostgresClientEnv returns querytest database connection details, as PG* env vars,
// for the given project's test database
func GetPostgresClientEnv(projectName string) []string {
	hostPort := GetPostgresHostPort(projectName)
	return []string{
		"PGHOST=localhost",
		fmt.Sprintf("PGPORT=%d", hostPort),
		"PGDATABASE=postgres",
		"PGUSER=postgres",
		fmt.Sprintf("PGPASSWORD=%s", PostgresPassword),
		"PGSSLMODE=disable",
	}
}
