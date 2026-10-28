package collectors

import (
	"fmt"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func cpuInfoRecord(id int, mhz int) string {
	return fmt.Sprintf("processor : %d\ncore id : %d\nphysical id : 0\ncpu MHz : %d\n\n", id, id, mhz)
}
func cpuInfoFixture(t *testing.T) (*CPUFreqCpuInfoCollector, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cpuinfo")
	if err := os.WriteFile(path, []byte(cpuInfoRecord(0, 1000)+cpuInfoRecord(2, 2000)), 0600); err != nil {
		t.Fatal(err)
	}
	c := &CPUFreqCpuInfoCollector{cpuInfoFile: path}
	if err := c.Init(nil); err != nil {
		t.Fatal(err)
	}
	return c, path
}
func cpuInfoReadFixture(t *testing.T, c *CPUFreqCpuInfoCollector, path, data string) map[string]float64 {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	ch := make(chan lp.CCMessage, 16)
	c.Read(time.Second, ch)
	out := make(map[string]float64)
	for len(ch) > 0 {
		m := <-ch
		id, _ := m.GetTag("type-id")
		v, _ := m.GetMetricValue()
		out[id] = v.(float64)
	}
	return out
}
