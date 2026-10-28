package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNUMARatesResetAndFailedReadGap(t *testing.T) {
	old := numaStatsNow
	t.Cleanup(func() { numaStatsNow = old })
	path := filepath.Join(t.TempDir(), "numastat")
	clock := time.Unix(1700000000, 0)
	numaStatsNow = func() time.Time { return clock }
	collector := NUMAStatsCollector{metricCollector: metricCollector{init: true}, config: NUMAStatsCollectorConfig{SendAbsoluteValues: true, SendDerivedValues: true}, topology: []NUMAStatsCollectorTopolgy{{file: path, tagSet: map[string]string{"type": "memoryDomain", "type-id": "0"}, previousValues: make(map[string]int64)}}}
	read := func(text string) map[string]float64 {
		if text == "" {
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
		} else if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		out := make(chan lp.CCMessage, 16)
		collector.Read(time.Second, out)
		return numaCounterFixtureValues(out)
	}
	if values := read("numa_hit 100\nnuma_miss 10\n"); len(values) != 2 {
		t.Fatalf("first sample emitted rates: %v", values)
	}
	clock = clock.Add(10 * time.Second)
	if values := read(""); len(values) != 0 {
		t.Fatalf("failed read emitted metrics: %v", values)
	}
	clock = clock.Add(10 * time.Second)
	if values := read("numa_hit 120\nnuma_miss 20\n"); values["numastats_numa_hit_rate"] != 1 {
		t.Errorf("rate used scheduled interval instead of successful-sample interval: %v", values)
	}
	clock = clock.Add(10 * time.Second)
	if values := read("numa_hit 5\nnuma_miss 25\n"); values["numastats_numa_hit"] != 5 {
		t.Fatal("reset absolute counter lost")
	} else if _, exists := values["numastats_numa_hit_rate"]; exists {
		t.Errorf("counter reset emitted false rate: %v", values)
	}
	clock = clock.Add(10 * time.Second)
	if values := read("numa_hit invalid\nnuma_miss 30\n"); values["numastats_numa_miss_rate"] != 0.5 {
		t.Errorf("valid companion counter affected by parse error: %v", values)
	}
	clock = clock.Add(10 * time.Second)
	if values := read("numa_hit 25\nnuma_miss 35\n"); values["numastats_numa_hit_rate"] != 1 {
		t.Errorf("individual counter parse gap lost interval: %v", values)
	}
	if values := read("numa_hit 25\nnuma_miss 35\n"); len(values) != 2 {
		t.Errorf("nonpositive elapsed time emitted rates: %v", values)
	}
}

func numaCounterFixtureValues(out chan lp.CCMessage) map[string]float64 {
	values := make(map[string]float64)
	for len(out) > 0 {
		m := <-out
		v, _ := m.GetMetricValue()
		switch n := v.(type) {
		case int64:
			values[m.Name()] = float64(n)
		case float64:
			values[m.Name()] = n
		}
	}
	return values
}
