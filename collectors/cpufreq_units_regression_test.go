package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCPUFreqPublishesHertz(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scaling_cur_freq")
	if err := os.WriteFile(path, []byte("2000000\n"), 0600); err != nil {
		t.Fatal(err)
	}
	c := CPUFreqCollector{metricCollector: metricCollector{init: true, meta: map[string]string{"unit": "Hz"}}, topology: []CPUFreqCollectorTopology{{scalingCurFreqFile: path, tagSet: map[string]string{"type": "hwthread", "type-id": "0"}}}}
	ch := make(chan lp.CCMessage, 2)
	c.Read(time.Second, ch)
	if len(ch) != 1 {
		t.Fatalf("output count=%d", len(ch))
	}
	m := <-ch
	v, _ := m.GetMetricValue()
	if v != int64(2000000000) {
		t.Fatalf("frequency=%v, want 2 GHz in Hz", v)
	}
	if unit, _ := m.GetMeta("unit"); unit != "Hz" {
		t.Fatalf("unit=%q", unit)
	}
}
