package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/orchestration-lite/core/pkg/storage"
	"github.com/orchestration-lite/core/pkg/types"
	"github.com/google/uuid"
	"github.com/urfave/cli/v2"
)

var etcdEndpoint string
var serverEndpoint string

func main() {
	app := &cli.App{
		Name:  "orc",
		Usage: "Orchestration CLI",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "etcd",
				Usage:       "etcd endpoint",
				Value:       "localhost:2379",
				Destination: &etcdEndpoint,
			},
			&cli.StringFlag{
				Name:        "server",
				Usage:       "orchestration server address",
				Value:       "http://localhost:8080",
				Destination: &serverEndpoint,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "pod",
				Usage: "Manage pods",
				Subcommands: []*cli.Command{
					{
						Name:   "create",
						Usage:  "Create a pod",
						Action: createPod,
					},
					{
						Name:   "get",
						Usage:  "Get a pod",
						Action: getPod,
					},
					{
						Name:   "list",
						Usage:  "List pods",
						Action: listPods,
					},
					{
						Name:   "delete",
						Usage:  "Delete a pod",
						Action: deletePod,
					},
				},
			},
			{
				Name:  "node",
				Usage: "Manage nodes",
				Subcommands: []*cli.Command{
					{
						Name:   "get",
						Usage:  "Get a node",
						Action: getNode,
					},
					{
						Name:   "list",
						Usage:  "List nodes",
						Action: listNodes,
					},
				},
			},
			{
				Name:  "deployment",
				Usage: "Manage deployments",
				Subcommands: []*cli.Command{
					{
						Name:   "create",
						Usage:  "Create a deployment",
						Action: createDeployment,
					},
					{
						Name:   "get",
						Usage:  "Get a deployment",
						Action: getDeployment,
					},
					{
						Name:   "list",
						Usage:  "List deployments",
						Action: listDeployments,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

// Pod commands
func createPod(c *cli.Context) error {
	if c.NArg() < 2 {
		return fmt.Errorf("usage: orc pod create <name> <image>")
	}

	name := c.Args().Get(0)
	image := c.Args().Get(1)

	pod := &types.Pod{
		Metadata: types.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       uuid.New().String(),
		},
		Spec: types.PodSpec{
			RestartPolicy: types.RestartPolicyAlways,
			Containers: []types.Container{
				{
					Name:          name,
					Image:         image,
					RestartPolicy: types.RestartPolicyAlways,
				},
			},
		},
	}

	body, _ := json.Marshal(pod)
	resp, err := http.Post(serverEndpoint+"/api/v1/pods", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to reach server at %s: %w", serverEndpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		var errResp map[string]string
		json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("server error: %s", errResp["error"])
	}

	fmt.Printf("Pod %s created\n", name)
	return nil
}

func getPod(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("usage: orc pod get <name>")
	}

	name := c.Args().Get(0)

	store, err := storage.NewEtcdStorage([]string{etcdEndpoint})
	if err != nil {
		return err
	}
	defer store.Close()

	pod, err := store.GetPod(context.Background(), "default", name)
	if err != nil {
		return err
	}

	data, _ := json.MarshalIndent(pod, "", "  ")
	fmt.Println(string(data))
	return nil
}

func listPods(c *cli.Context) error {
	store, err := storage.NewEtcdStorage([]string{etcdEndpoint})
	if err != nil {
		return err
	}
	defer store.Close()

	pods, err := store.ListPods(context.Background(), "default")
	if err != nil {
		return err
	}

	fmt.Printf("NAME\tNAMESPACE\tPHASE\tNODE\n")
	for _, pod := range pods {
		fmt.Printf("%s\t%s\t%s\t%s\n", pod.Metadata.Name, pod.Metadata.Namespace, pod.Status.Phase, pod.Spec.NodeName)
	}
	return nil
}

func deletePod(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("usage: orc pod delete <name>")
	}

	name := c.Args().Get(0)

	store, err := storage.NewEtcdStorage([]string{etcdEndpoint})
	if err != nil {
		return err
	}
	defer store.Close()

	if err := store.DeletePod(context.Background(), "default", name); err != nil {
		return err
	}

	fmt.Printf("Pod %s deleted\n", name)
	return nil
}

// Node commands
func getNode(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("usage: orc node get <name>")
	}

	name := c.Args().Get(0)

	store, err := storage.NewEtcdStorage([]string{etcdEndpoint})
	if err != nil {
		return err
	}
	defer store.Close()

	node, err := store.GetNode(context.Background(), name)
	if err != nil {
		return err
	}

	data, _ := json.MarshalIndent(node, "", "  ")
	fmt.Println(string(data))
	return nil
}

func listNodes(c *cli.Context) error {
	store, err := storage.NewEtcdStorage([]string{etcdEndpoint})
	if err != nil {
		return err
	}
	defer store.Close()

	nodes, err := store.ListNodes(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("NAME\tREADY\tCPU\tMEMORY\n")
	for _, node := range nodes {
		ready := "False"
		if node.Status.Ready {
			ready = "True"
		}
		fmt.Printf("%s\t%s\t%s\t%s\n", node.Metadata.Name, ready, node.Capacity["cpu"], node.Capacity["memory"])
	}
	return nil
}

// Deployment commands
func createDeployment(c *cli.Context) error {
	if c.NArg() < 3 {
		return fmt.Errorf("usage: orc deployment create <name> <image> <replicas>")
	}

	name := c.Args().Get(0)
	image := c.Args().Get(1)
	var replicas int32
	fmt.Sscanf(c.Args().Get(2), "%d", &replicas)

	store, err := storage.NewEtcdStorage([]string{etcdEndpoint})
	if err != nil {
		return err
	}
	defer store.Close()

	deployment := &types.Deployment{
		Metadata: types.ObjectMeta{
			Name:      name,
			Namespace: "default",
			UID:       uuid.New().String(),
		},
		Spec: types.DeploymentSpec{
			Replicas: replicas,
			Selector: map[string]string{
				"app": name,
			},
			Template: types.PodTemplateSpec{
				Metadata: types.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: types.PodSpec{
					RestartPolicy: types.RestartPolicyAlways,
					Containers: []types.Container{
						{
							Name:            name,
							Image:           image,
							RestartPolicy:   types.RestartPolicyAlways,
						},
					},
				},
			},
		},
	}

	if err := store.SaveDeployment(context.Background(), deployment); err != nil {
		return err
	}

	fmt.Printf("Deployment %s created with %d replicas\n", name, replicas)
	return nil
}

func getDeployment(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("usage: orc deployment get <name>")
	}

	name := c.Args().Get(0)

	store, err := storage.NewEtcdStorage([]string{etcdEndpoint})
	if err != nil {
		return err
	}
	defer store.Close()

	deployment, err := store.GetDeployment(context.Background(), "default", name)
	if err != nil {
		return err
	}

	data, _ := json.MarshalIndent(deployment, "", "  ")
	fmt.Println(string(data))
	return nil
}

func listDeployments(c *cli.Context) error {
	store, err := storage.NewEtcdStorage([]string{etcdEndpoint})
	if err != nil {
		return err
	}
	defer store.Close()

	deployments, err := store.ListDeployments(context.Background(), "default")
	if err != nil {
		return err
	}

	fmt.Printf("NAME\tREPLICAS\tUPDATED\tAVAILABLE\n")
	for _, dep := range deployments {
		fmt.Printf("%s\t%d\t%d\t%d\n", dep.Metadata.Name, dep.Spec.Replicas, dep.Status.UpdatedReplicas, dep.Status.AvailableReplicas)
	}
	return nil
}
