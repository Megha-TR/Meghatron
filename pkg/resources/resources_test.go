package resources

import "testing"

func TestQuantityUnits(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  int64
	}{{"100m", 100}, {"0.5", 500}, {"2", 2000}} {
		got, err := ParseCPU(tc.value)
		if err != nil || got != tc.want {
			t.Errorf("CPU %s: %d %v", tc.value, got, err)
		}
	}
	for _, tc := range []struct {
		value string
		want  int64
	}{{"256Mi", 256 << 20}, {"1Gi", 1 << 30}, {"256", 256 << 20}, {"1024B", 1024}} {
		got, err := ParseMemory(tc.value)
		if err != nil || got != tc.want {
			t.Errorf("memory %s: %d %v", tc.value, got, err)
		}
	}
	for _, bad := range []string{"-1", "NaN", "Inf", "100oops", "0", "1e99"} {
		if _, err := ParseCPU(bad); err == nil {
			t.Errorf("accepted invalid CPU %q", bad)
		}
	}
}
