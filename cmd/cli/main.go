package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

var client = &http.Client{Timeout: 15 * time.Second}

func request(c *cli.Context, method, path string, body any) error {
	var data []byte
	var err error
	if body != nil {
		data, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(c.Context, method, strings.TrimRight(c.String("server"), "/")+"/api/v1/"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	result, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("API %d: %s", resp.StatusCode, strings.TrimSpace(string(result)))
	}
	var pretty bytes.Buffer
	if json.Indent(&pretty, result, "", "  ") == nil {
		fmt.Println(pretty.String())
	} else {
		fmt.Println(string(result))
	}
	return nil
}
func path(c *cli.Context, kind string, withName bool) string {
	p := kind
	if kind != "nodes" {
		p += "/" + url.PathEscape(c.String("namespace"))
	}
	if withName {
		p += "/" + url.PathEscape(c.Args().First())
	}
	return p
}
func app() *cli.App {
	a := &cli.App{Name: "orc", Usage: "Manage Meghatron through its REST API", Flags: []cli.Flag{&cli.StringFlag{Name: "server", Value: "http://localhost:8080", EnvVars: []string{"MEGHATRON_SERVER"}}, &cli.StringFlag{Name: "namespace", Aliases: []string{"n"}, Value: "default"}}}
	for _, kind := range []string{"pod", "node", "deployment"} {
		kind := kind
		plural := kind + "s"
		command := &cli.Command{Name: kind, Subcommands: []*cli.Command{
			{Name: "list", Action: func(c *cli.Context) error { return request(c, "GET", path(c, plural, false), nil) }},
			{Name: "get", ArgsUsage: "<name>", Action: func(c *cli.Context) error {
				if c.NArg() != 1 {
					return fmt.Errorf("provide one name")
				}
				return request(c, "GET", path(c, plural, true), nil)
			}},
		}}
		if kind != "node" {
			command.Subcommands = append(command.Subcommands, &cli.Command{Name: "delete", ArgsUsage: "<name>", Action: func(c *cli.Context) error {
				if c.NArg() != 1 {
					return fmt.Errorf("provide one name")
				}
				return request(c, "DELETE", path(c, plural, true), nil)
			}})
			command.Subcommands = append(command.Subcommands, &cli.Command{Name: "create", ArgsUsage: "<name> <image> [replicas]", Action: func(c *cli.Context) error {
				if c.NArg() < 2 || c.NArg() > 3 || (kind == "pod" && c.NArg() != 2) {
					return fmt.Errorf("usage: orc %s create <name> <image> [replicas]", kind)
				}
				name, image := c.Args().Get(0), c.Args().Get(1)
				meta := map[string]any{"name": name, "namespace": c.String("namespace")}
				spec := map[string]any{"containers": []any{map[string]any{"name": "app", "image": image}}}
				if kind == "deployment" {
					replicas := 1
					if c.NArg() == 3 {
						var err error
						replicas, err = strconv.Atoi(c.Args().Get(2))
						if err != nil {
							return err
						}
					}
					spec = map[string]any{"replicas": replicas, "selector": map[string]string{"app": name}, "template": map[string]any{"metadata": map[string]any{"labels": map[string]string{"app": name}}, "spec": spec}}
				}
				return request(c, "POST", plural, map[string]any{"metadata": meta, "spec": spec})
			}})
		}
		if kind == "deployment" {
			command.Subcommands = append(command.Subcommands, &cli.Command{Name: "scale", ArgsUsage: "<name> <replicas>", Action: func(c *cli.Context) error {
				if c.NArg() != 2 {
					return fmt.Errorf("provide name and replica count")
				}
				n, err := strconv.Atoi(c.Args().Get(1))
				if err != nil {
					return err
				}
				return request(c, "PATCH", path(c, plural, true)+"/scale", map[string]int{"replicas": n})
			}})
		}
		a.Commands = append(a.Commands, command)
	}
	return a
}
func main() {
	if err := app().Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
