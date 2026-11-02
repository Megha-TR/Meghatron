// Package resources uses millicores for CPU and bytes for memory internally.
package resources

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Megha-TR/Meghatron/pkg/types"
)

type Quantities struct{ CPU, Memory int64 }

func ParseCPU(value string) (int64, error) {
	multiplier := float64(1000)
	if strings.HasSuffix(value, "m") {
		value = strings.TrimSuffix(value, "m")
		multiplier = 1
	}
	return parse(value, multiplier)
}
func ParseMemory(value string) (int64, error) {
	// Bare memory numbers retain the original project's MiB convention.
	units := []struct {
		suffix     string
		multiplier float64
	}{{"Gi", 1 << 30}, {"Mi", 1 << 20}, {"Ki", 1 << 10}, {"G", 1e9}, {"M", 1e6}, {"K", 1e3}, {"B", 1}}
	for _, u := range units {
		if strings.HasSuffix(value, u.suffix) {
			return parse(strings.TrimSuffix(value, u.suffix), u.multiplier)
		}
	}
	return parse(value, 1<<20)
}
func parse(value string, multiplier float64) (int64, error) {
	n, err := strconv.ParseFloat(value, 64)
	total := n * multiplier
	if err != nil || math.IsNaN(total) || math.IsInf(total, 0) || total <= 0 || total >= math.MaxInt64 {
		return 0, fmt.Errorf("invalid resource quantity %q", value)
	}
	return int64(math.Ceil(total)), nil
}
func Parse(list types.ResourceList) (Quantities, error) {
	var q Quantities
	for k, v := range list {
		var err error
		switch k {
		case "cpu":
			q.CPU, err = ParseCPU(v)
		case "memory":
			q.Memory, err = ParseMemory(v)
		default:
			return q, fmt.Errorf("unsupported resource %q", k)
		}
		if err != nil {
			return q, fmt.Errorf("%s: %w", k, err)
		}
	}
	return q, nil
}
func Requests(pod *types.Pod) (Quantities, error) {
	var total Quantities
	for _, c := range pod.Spec.Containers {
		q, err := Parse(c.Resources.Requests)
		if err != nil {
			return total, err
		}
		if total.CPU > math.MaxInt64-q.CPU || total.Memory > math.MaxInt64-q.Memory {
			return total, fmt.Errorf("pod resource requests overflow")
		}
		total.CPU += q.CPU
		total.Memory += q.Memory
	}
	return total, nil
}
func (q Quantities) Fits(need Quantities) bool { return q.CPU >= need.CPU && q.Memory >= need.Memory }
func (q Quantities) Sub(used Quantities) Quantities {
	return Quantities{q.CPU - used.CPU, q.Memory - used.Memory}
}
