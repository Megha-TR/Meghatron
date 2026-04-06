package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	orch_types "github.com/orchestration-lite/core/pkg/types"
)

// ContainerRuntime manages container operations via the Docker CLI.
type ContainerRuntime struct {
	dockerBin string
}

// dockerLocations are common install paths to check when docker is not in PATH.
var dockerLocations = []string{
	"docker",
	"/usr/local/bin/docker",
	"/usr/bin/docker",
	"/Applications/Docker.app/Contents/Resources/bin/docker",
}

// findDocker returns the path to a working docker binary.
func findDocker() (string, error) {
	for _, path := range dockerLocations {
		if err := exec.Command(path, "info").Run(); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("docker binary not found or daemon not running; checked: %v", dockerLocations)
}

// NewContainerRuntime verifies that the Docker daemon is reachable and returns
// a runtime instance.
func NewContainerRuntime() (*ContainerRuntime, error) {
	bin, err := findDocker()
	if err != nil {
		return nil, err
	}
	return &ContainerRuntime{dockerBin: bin}, nil
}

// CreateContainer pulls the image (if not already present), creates a Docker
// container, and returns its full container ID.
func (cr *ContainerRuntime) CreateContainer(ctx context.Context, pod *orch_types.Pod, containerSpec orch_types.Container) (string, error) {
	if err := cr.pullImage(ctx, containerSpec.Image); err != nil {
		return "", fmt.Errorf("failed to pull image %s: %w", containerSpec.Image, err)
	}

	// Container name is scoped to namespace/pod/container to avoid collisions.
	name := containerName(pod, containerSpec)

	args := []string{"create", "--name", name}

	// Environment variables
	for _, e := range containerSpec.Env {
		args = append(args, "-e", fmt.Sprintf("%s=%s", e.Name, e.Value))
	}

	// Port bindings
	for _, p := range containerSpec.Ports {
		proto := p.Protocol
		if proto == "" {
			proto = "tcp"
		}
		args = append(args, "-p",
			fmt.Sprintf("%d:%d/%s", p.ContainerPort, p.ContainerPort, strings.ToLower(proto)))
	}

	// Resource limits
	if limits := containerSpec.Resources.Limits; limits != nil {
		if cpuStr, ok := limits["cpu"]; ok {
			args = append(args, "--cpus", cpuStr)
		}
		if memStr, ok := limits["memory"]; ok {
			// treat as MiB
			if memMiB, err := strconv.ParseInt(memStr, 10, 64); err == nil {
				args = append(args, "--memory", fmt.Sprintf("%dm", memMiB))
			}
		}
	}

	// Restart policy
	args = append(args, "--restart", toDockerRestartPolicy(containerSpec.RestartPolicy))

	args = append(args, containerSpec.Image)

	out, err := cr.runDockerCommand(ctx, args...)
	if err != nil {
		return "", fmt.Errorf("docker create failed for %s: %w", name, err)
	}

	id := strings.TrimSpace(out)
	log.Printf("Created container %s (ID: %.12s) for pod %s", name, id, pod.Metadata.Name)
	return id, nil
}

// StartContainer starts an already-created container by ID or name.
func (cr *ContainerRuntime) StartContainer(ctx context.Context, containerID string) error {
	if _, err := cr.runDockerCommand(ctx, "start", containerID); err != nil {
		return fmt.Errorf("docker start %.12s failed: %w", containerID, err)
	}
	log.Printf("Started container %.12s", containerID)
	return nil
}

// StopContainer gracefully stops a running container (10 s timeout).
func (cr *ContainerRuntime) StopContainer(ctx context.Context, containerID string) error {
	if _, err := cr.runDockerCommand(ctx, "stop", "--time", "10", containerID); err != nil {
		return fmt.Errorf("docker stop %.12s failed: %w", containerID, err)
	}
	log.Printf("Stopped container %.12s", containerID)
	return nil
}

// RemoveContainer force-removes a container.
func (cr *ContainerRuntime) RemoveContainer(ctx context.Context, containerID string) error {
	if _, err := cr.runDockerCommand(ctx, "rm", "-f", containerID); err != nil {
		return fmt.Errorf("docker rm %.12s failed: %w", containerID, err)
	}
	log.Printf("Removed container %.12s", containerID)
	return nil
}

// dockerInspect is the shape we unmarshal from `docker inspect`.
type dockerInspect struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Running bool   `json:"Running"`
		Status  string `json:"Status"`
	} `json:"State"`
}

// GetContainerStatus inspects a container and returns its current status.
func (cr *ContainerRuntime) GetContainerStatus(ctx context.Context, containerID string) (*orch_types.ContainerStatus, error) {
	out, err := cr.runDockerCommand(ctx, "inspect", containerID)
	if err != nil {
		return nil, fmt.Errorf("docker inspect %.12s failed: %w", containerID, err)
	}

	var results []dockerInspect
	if err := json.Unmarshal([]byte(out), &results); err != nil || len(results) == 0 {
		return nil, fmt.Errorf("failed to parse docker inspect output: %w", err)
	}

	info := results[0]
	return &orch_types.ContainerStatus{
		Name:        strings.TrimPrefix(info.Name, "/"),
		ContainerID: info.ID,
		Ready:       info.State.Running,
		Running:     info.State.Running,
		State:       strings.ToLower(info.State.Status),
	}, nil
}

// Close is a no-op for the CLI-based runtime.
func (cr *ContainerRuntime) Close() error {
	return nil
}

// pullImage pulls a Docker image only if it is not already present locally.
func (cr *ContainerRuntime) pullImage(ctx context.Context, image string) error {
	// Check local cache first to avoid unnecessary network calls.
	out, err := cr.runDockerCommand(ctx, "images", "-q", image)
	if err == nil && strings.TrimSpace(out) != "" {
		return nil // already present
	}

	log.Printf("Pulling image: %s", image)
	cmd := exec.CommandContext(ctx, cr.dockerBin, "pull", image)
	cmd.Stdout = log.Writer()
	cmd.Stderr = log.Writer()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker pull %s failed: %w", image, err)
	}
	return nil
}

// --- helpers -----------------------------------------------------------------

// runDockerCommand runs a docker subcommand and returns combined stdout output.
// Stderr is captured into the error on failure.
func (cr *ContainerRuntime) runDockerCommand(ctx context.Context, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, cr.dockerBin, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func containerName(pod *orch_types.Pod, spec orch_types.Container) string {
	return fmt.Sprintf("%s-%s-%s", pod.Metadata.Namespace, pod.Metadata.Name, spec.Name)
}

func toDockerRestartPolicy(policy orch_types.RestartPolicy) string {
	switch policy {
	case orch_types.RestartPolicyAlways:
		return "always"
	case orch_types.RestartPolicyOnFailure:
		return "on-failure:3"
	default:
		return "no"
	}
}
