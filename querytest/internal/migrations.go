package impl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// MigrateScriptFilename is the canonical name of the bash script that runs a project's
// database migrations against the postgres db described by PGHOST, PGPORT, etc. - we
// assume that querytest is only run in the context of a repo that has such a file in
// its root directory. Invoking the script directly (rather than running golang-migrate
// natively) ensures that our environment is set up the same way in production as in
// tests, with the same tools versions etc.
const MigrateScriptFilename = "db-migrate.sh"

// RunMigrations invokes the current project's db-migrate script, configured to run
// against the project's querytest database
func RunMigrations(ctx context.Context, rootDir string) error {
	migrateScriptPath := filepath.Join(rootDir, MigrateScriptFilename)
	_, err := os.Stat(migrateScriptPath)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, fmt.Sprintf("./%s", MigrateScriptFilename))
	cmd.Dir = rootDir
	cmd.Env = append(os.Environ(), GetPostgresClientEnv(GetProjectName(rootDir))...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
