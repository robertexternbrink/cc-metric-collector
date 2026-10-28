package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"strings"
	"testing"
	"time"
)

func TestCpustatNewCPUStartsWithBaseline(t *testing.T) {
	collector := cpustatRegressionCollector(t, "{}")
	cpustatRegressionFixture(t, "cpu 6000 0 2000 2000 0 0 0 0 0 0\ncpu0 0 0 0 0 0 0 0 0 0 0\ncpu7 6000 0 2000 2000 0 0 0 0 0 0\n")
	output := make(chan lp.CCMessage, 64)
	if caught := cpustatRegressionCatch(func() { collector.Read(time.Second, output) }); caught != nil {
		t.Fatalf("new CPU caused an uncontrolled panic: %v", caught)
	}
	for len(output) > 0 {
		message := <-output
		if id, _ := message.GetTag("type-id"); id == "7" {
			t.Errorf("first sample of CPU7 emitted a rate without an earlier sample: %s", message.Name())
		}
	}
	collector.parseStatLine(strings.Fields("cpu7 6060 0 2020 2020 0 0 0 0 0 0"), collector.cputags["cpu7"], output, time.Unix(1700000000, 0), time.Second)
	message := cpustatRegressionOutput(output)["cpu_used"]
	if message == nil {
		t.Fatal("second CPU7 sample has no derived rate")
	}
	if value, ok := message.GetMetricValue(); !ok || value != float64(80) {
		t.Errorf("second CPU7 sample=%v, want80", value)
	}
	if id, _ := message.GetTag("type-id"); id != "7" {
		t.Errorf("native hardware ID changed: %q", id)
	}
}
