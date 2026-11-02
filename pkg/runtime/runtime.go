package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Megha-TR/Meghatron/pkg/resources"
	"log"
	"os/exec"
	"strings"
	"time"

	orch_types "github.com/Megha-TR/Meghatron/pkg/types"
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := exec.CommandContext(ctx, path, "info").Run()
		cancel()
		if err == nil {
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

	args := []string{"create", "--name", name,
		"--label", "meghatron.managed=true", "--label", "meghatron.node=" + pod.Spec.NodeName,
		"--label", "meghatron.podUID=" + pod.Metadata.UID, "--label", "meghatron.container=" + containerSpec.Name,
		"--label", "meghatron.namespace=" + pod.Metadata.Namespace,
		"--label", "meghatron.pod=" + pod.Metadata.Name}

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
		args = append(args, "--expose", fmt.Sprintf("%d/%s", p.ContainerPort, strings.ToLower(proto)))
		if p.HostPort != 0 {
			args = append(args, "-p", fmt.Sprintf("127.0.0.1:%d:%d/%s", p.HostPort, p.ContainerPort, strings.ToLower(proto)))
		}
	}

	// Resource limits are normalized into Docker's cores and bytes.
	limits, err := resources.Parse(containerSpec.Resources.Limits)
	if err != nil {
		return "", err
	}
	if limits.CPU > 0 {
		args = append(args, "--cpus", fmt.Sprintf("%.3f", float64(limits.CPU)/1000))
	}
	if limits.Memory > 0 {
		args = append(args, "--memory", fmt.Sprint(limits.Memory))
	}
	// The agent owns restart decisions and observes exits on the next pass.
	args = append(args, "--restart", "no")

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
	return fmt.Sprintf("meghatron-%s-%s", pod.Metadata.UID, spec.Name)
}

// ManagedContainer carries stable ownership across agent process restarts.
type ManagedContainer struct {
	orch_types.ContainerStatus
	PodUID        string
	ContainerName string
	ExitCode      int
}

func (cr *ContainerRuntime) ListContainers(ctx context.Context, nodeName string) ([]ManagedContainer, error) {
	output, err := cr.runDockerCommand(ctx, "ps", "-aq", "--filter", "label=meghatron.managed=true", "--filter", "label=meghatron.node="+nodeName)
	if err != nil {
		return nil, err
	}
	ids := strings.Fields(output)
	if len(ids) == 0 {
		return []ManagedContainer{}, nil
	}
	args := append([]string{"inspect"}, ids...)
	output, err = cr.runDockerCommand(ctx, args...)
	if err != nil {
		return nil, err
	}
	var inspected []struct {
		ID     string `json:"Id"`
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
		State struct {
			Running  bool   `json:"Running"`
			Status   string `json:"Status"`
			ExitCode int    `json:"ExitCode"`
		} `json:"State"`
	}
	if err := json.Unmarshal([]byte(output), &inspected); err != nil {
		return nil, err
	}
	result := make([]ManagedContainer, 0, len(inspected))
	for _, c := range inspected {
		result = append(result, ManagedContainer{ContainerStatus: orch_types.ContainerStatus{ContainerID: c.ID, Running: c.State.Running, Ready: c.State.Running, State: c.State.Status}, PodUID: c.Config.Labels["meghatron.podUID"], ContainerName: c.Config.Labels["meghatron.container"], ExitCode: c.State.ExitCode})
	}
	return result, nil
}

// Capacity reports the Docker daemon's capacity, which may differ from its host.
func (cr *ContainerRuntime) Capacity(ctx context.Context) (orch_types.ResourceList, error) {
	output, err := cr.runDockerCommand(ctx, "info", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	var info struct {
		NCPU     int64
		MemTotal int64
	}
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		return nil, err
	}
	if info.NCPU <= 0 || info.MemTotal <= 0 {
		return nil, fmt.Errorf("invalid Docker capacity")
	}
	return orch_types.ResourceList{"cpu": fmt.Sprintf("%dm", info.NCPU*1000), "memory": fmt.Sprintf("%dB", info.MemTotal)}, nil
}
