package api

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Megha-TR/Meghatron/pkg/resources"
	"github.com/Megha-TR/Meghatron/pkg/types"
)

var envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func defaultAndValidateSpec(spec *types.PodSpec) error {
	if len(spec.Containers) == 0 || len(spec.Containers) > 16 {
		return fmt.Errorf("spec requires 1–16 containers")
	}
	if spec.NodeName != "" {
		return fmt.Errorf("nodeName is assigned by the scheduler")
	}
	if len(spec.Resources.Requests) > 0 || len(spec.Resources.Limits) > 0 {
		return fmt.Errorf("set resources on each container")
	}
	if spec.RestartPolicy == "" {
		spec.RestartPolicy = types.RestartPolicyAlways
	}
	if !validPolicy(spec.RestartPolicy) {
		return fmt.Errorf("invalid restartPolicy")
	}
	names := map[string]bool{}
	ports := map[string]bool{}
	for i := range spec.Containers {
		c := &spec.Containers[i]
		if !types.ValidName(c.Name) || names[c.Name] {
			return fmt.Errorf("container names must be valid and unique")
		}
		names[c.Name] = true
		if c.Image == "" || strings.HasPrefix(c.Image, "-") || strings.ContainsAny(c.Image, " \t\r\n") {
			return fmt.Errorf("container image is required and must be a valid image reference")
		}
		if c.RestartPolicy == "" {
			c.RestartPolicy = spec.RestartPolicy
		}
		if !validPolicy(c.RestartPolicy) {
			return fmt.Errorf("invalid container restartPolicy")
		}
		for _, e := range c.Env {
			if !envName.MatchString(e.Name) {
				return fmt.Errorf("invalid environment variable name")
			}
		}
		for j := range c.Ports {
			p := &c.Ports[j]
			p.Protocol = strings.ToUpper(p.Protocol)
			if p.Protocol == "" {
				p.Protocol = "TCP"
			}
			if p.ContainerPort < 1 || p.ContainerPort > 65535 || p.HostPort < 0 || p.HostPort > 65535 || (p.Protocol != "TCP" && p.Protocol != "UDP") {
				return fmt.Errorf("invalid port or protocol")
			}
			if p.HostPort > 0 {
				key := fmt.Sprintf("%d/%s", p.HostPort, p.Protocol)
				if ports[key] {
					return fmt.Errorf("duplicate host port")
				}
				ports[key] = true
			}
		}
		if c.Resources.Limits == nil {
			c.Resources.Limits = types.ResourceList{}
		}
		if c.Resources.Requests == nil {
			c.Resources.Requests = types.ResourceList{}
		}
		if _, ok := c.Resources.Limits["cpu"]; !ok {
			c.Resources.Limits["cpu"] = "1"
		}
		if _, ok := c.Resources.Limits["memory"]; !ok {
			c.Resources.Limits["memory"] = "256Mi"
		}
		limits, err := resources.Parse(c.Resources.Limits)
		if err != nil {
			return err
		}
		if _, ok := c.Resources.Requests["cpu"]; !ok {
			c.Resources.Requests["cpu"] = "100m"
			if limits.CPU < 100 {
				c.Resources.Requests["cpu"] = c.Resources.Limits["cpu"]
			}
		}
		if _, ok := c.Resources.Requests["memory"]; !ok {
			c.Resources.Requests["memory"] = "64Mi"
			if limits.Memory < 64<<20 {
				c.Resources.Requests["memory"] = c.Resources.Limits["memory"]
			}
		}
		requests, err := resources.Parse(c.Resources.Requests)
		if err != nil {
			return err
		}
		if !limits.Fits(requests) {
			return fmt.Errorf("container resource requests must not exceed limits")
		}
	}
	return nil
}
func validPolicy(policy types.RestartPolicy) bool {
	return policy == types.RestartPolicyAlways || policy == types.RestartPolicyOnFailure || policy == types.RestartPolicyNever
}
func validateDeployment(d *types.Deployment) error {
	if err := types.ValidateMeta(d.Metadata); err != nil {
		return err
	}
	if d.Spec.Replicas < 0 || d.Spec.Replicas > 100 {
		return fmt.Errorf("replicas must be between 0 and 100")
	}
	if len(d.Spec.Selector) == 0 {
		return fmt.Errorf("deployment selector must not be empty")
	}
	for k, v := range d.Spec.Selector {
		if d.Spec.Template.Metadata.Labels[k] != v || k == "" || v == "" {
			return fmt.Errorf("selector must match template labels")
		}
	}
	return defaultAndValidateSpec(&d.Spec.Template.Spec)
}
