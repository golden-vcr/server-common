package impl

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// ErrNoSuchContainer is returned to indicate that a docker command targeting a specific
// container finished normally, but the targeted container was not found
var ErrNoSuchContainer = errors.New("no such container")

// ContainerId is a hexadecimal string that identifies a valid docker container, as
// reported in the output of docker CLI commands
type ContainerId string

// IsDockerInstalled returns true if the docker CLI is installed and in the PATH
func IsDockerInstalled(ctx context.Context) bool {
	return exec.Command("docker", "-v").Run() == nil
}

// FindContainerId checks for an existing docker container with the given name, and
// returns its container ID (as a 12-digit hex string) if found. If the command
// completes without error but no such container exists, returns ErrNoSuchContainer.
func FindContainerId(ctx context.Context, containerName string) (ContainerId, error) {
	cmd := exec.CommandContext(
		ctx,
		"docker", "ps",
		"--filter", fmt.Sprintf("name=%s", containerName),
		"--format", "{{.ID}}",
	)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker ps failed: %w", err)
	}
	containerId, err := parseContainerIdFromOutput(output)
	if err != nil {
		return "", fmt.Errorf("failed to find a running container named %s: %w", containerName, ErrNoSuchContainer)
	}
	return containerId, nil
}

// StartContainer attempts to start a new docker container which will run in the
// background and be auto-removed upon exit, returning the new container ID if
// successful. Values passed via envExports, volumeMounts, and portMappings are expected
// to be pre-formatted strings as passed via -e, -v, and -p respectively: e.g.
// "SOME_ENV_VAR=42", "/home/foo/bar:/etc/bar:ro", "5000:80".
func StartContainer(ctx context.Context, containerName string, image string, envExports []string, volumeMounts []string, portMappings []string) (ContainerId, error) {
	args := []string{
		"run",
		"--rm",
		"-d",
		"--name", containerName,
	}
	for _, e := range envExports {
		args = append(args, "-e", e)
	}
	for _, v := range volumeMounts {
		args = append(args, "-v", v)
	}
	for _, p := range portMappings {
		args = append(args, "-p", p)
	}
	args = append(args, image)

	cmd := exec.CommandContext(ctx, "docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("docker run failed: %w", err)
	}
	containerId, err := parseContainerIdFromOutput(output)
	if err != nil {
		return "", fmt.Errorf("failed to parse container ID from docker run output: %w", err)
	}
	return containerId, nil
}

// TailContainerOutput runs docker logs -f until the provided context is canceled,
// piping line-by-line output from both stdout and stderr to the provided channel
func TailContainerOutput(ctx context.Context, containerId ContainerId, lines chan<- string) error {
	cmd := exec.CommandContext(ctx, "docker", "logs", "-f", string(containerId))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to open pipe for docker logs stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to open pipe for docker logs stderr: %w", err)
	}
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()
	return cmd.Run()
}

// StopContainer terminates the docker container with the given ID
func StopContainer(ctx context.Context, containerId ContainerId) error {
	return exec.CommandContext(ctx, "docker", "stop", string(containerId)).Run()
}

var containerIdRegex = regexp.MustCompile("^[0-9a-f]{12,}$")

func parseContainerId(s string) (ContainerId, error) {
	if containerIdRegex.MatchString(s) {
		return ContainerId(s), nil
	}
	return "", fmt.Errorf("invalid container ID")
}

func parseContainerIdFromOutput(output []byte) (ContainerId, error) {
	lines := strings.Split(string(output), "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("no output")
	}
	return parseContainerId(strings.TrimSpace(lines[0]))
}
