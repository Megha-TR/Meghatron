package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Megha-TR/Meghatron/pkg/types"
)

func fakeDocker(t *testing.T, script string) *ContainerRuntime {
	t.Helper()
	path := filepath.Join(t.TempDir(), "docker")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	return &ContainerRuntime{dockerBin: path}
}
func TestCreateUsesNormalizedLimitsAndExplicitHostPorts(t *testing.T) {
	log := filepath.Join(t.TempDir(), "args")
	t.Setenv("DOCKER_ARGS", log)
	cr := fakeDocker(t, `case "$1" in
 images) echo cached ;;
 create) printf '%s\n' "$@" >> "$DOCKER_ARGS"; echo container-id ;;
 esac
`)
	pod := &types.Pod{Metadata: types.ObjectMeta{UID: "unique", Name: "web", Namespace: "default"}, Spec: types.PodSpec{NodeName: "worker"}}
	spec := types.Container{Name: "app", Image: "nginx", Ports: []types.ContainerPort{{ContainerPort: 80}}, Resources: types.ResourceRequirements{Limits: types.ResourceList{"cpu": "500m", "memory": "128Mi"}}}
	if _, err := cr.CreateContainer(context.Background(), pod, spec); err != nil {
		t.Fatal(err)
	}
	args, _ := os.ReadFile(log)
	text := string(args)
	for _, expected := range []string{"--cpus\n0.500", "--memory\n134217728", "--label\nmeghatron.podUID=unique", "--label\nmeghatron.node=worker", "--expose\n80/tcp"} {
		if !strings.Contains(text, expected) {
			t.Errorf("missing %q in %s", expected, text)
		}
	}
	if strings.Contains(text, "\n-p\n") {
		t.Fatal("container port was published to host implicitly")
	}
	spec.Ports[0].HostPort = 8080
	if _, err := cr.CreateContainer(context.Background(), pod, spec); err != nil {
		t.Fatal(err)
	}
	args, _ = os.ReadFile(log)
	if !strings.Contains(string(args), "127.0.0.1:8080:80/tcp") {
		t.Fatal("explicit host port missing loopback binding")
	}
}
func TestDockerStateAndCapacity(t *testing.T) {
	cr := fakeDocker(t, `case "$1" in
 ps) echo id ;;
 inspect) echo '[{"Id":"id","Config":{"Labels":{"meghatron.podUID":"uid","meghatron.container":"app"}},"State":{"Running":false,"Status":"exited","ExitCode":7}}]' ;;
 info) echo '{"NCPU":4,"MemTotal":2147483648}' ;;
 esac
`)
	cs, err := cr.ListContainers(context.Background(), "worker")
	if err != nil || len(cs) != 1 || cs[0].ExitCode != 7 || cs[0].PodUID != "uid" {
		t.Fatalf("inspect: %+v %v", cs, err)
	}
	q, err := cr.Capacity(context.Background())
	if err != nil || q["cpu"] != "4000m" || q["memory"] != "2147483648B" {
		t.Fatalf("capacity: %+v %v", q, err)
	}
}
func TestDockerFailureIncludesStderr(t *testing.T) {
	cr := fakeDocker(t, "echo 'daemon unavailable' >&2\nexit 1\n")
	if err := cr.RemoveContainer(context.Background(), "id"); err == nil || !strings.Contains(err.Error(), "daemon unavailable") {
		t.Fatalf("lost Docker error: %v", err)
	}
}
