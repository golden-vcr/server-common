package impl

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// GetProjectName infers the name of the current Go project, given the absolute path to
// its root directory
func GetProjectName(rootDir string) string {
	return filepath.Base(rootDir)
}

// FindProjectRootDir returns the absolute path to the root directory of the project
// (i.e. the directory where go.mod resides), or returns an error if unable to resolve
// that path. Searches each successive directory for a go.mod file, starting from the
// current working directory and progressing upwards through each parent directory.
//
// This code is only intended to be invoked by the Go toolchain during development, so
// we assume that the working directory for the current process will always be somewhere
// within the relevant Go project.
func FindProjectRootDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	dirpath := cwd
	for {
		candidateGoModPath := filepath.Join(dirpath, "go.mod")
		_, err := os.Stat(candidateGoModPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("failed to stat file at %s: %w", candidateGoModPath, err)
		}

		// If we found go.mod in dirpath, we've found our project root dir
		if err == nil {
			return dirpath, nil
		}

		// Otherwise, continuing iterating, checking the parent directory until we hit
		// a dead end
		parentDirpath := filepath.Dir(dirpath)
		if parentDirpath == dirpath {
			return "", fmt.Errorf("reached root directory without finding go.mod")
		}
		dirpath = parentDirpath
	}
}
