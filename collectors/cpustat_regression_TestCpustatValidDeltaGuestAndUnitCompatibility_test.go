package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"math"
	"strings"
	"testing"
	"time"
)

func TestCpustatValidDeltaGuestAndUnitCompatibility(t *testing.T) {
	collector := cpustatRegressionCollector(t, "{}")
	output := make(chan lp.CCMessage, 32)
	now := time.Unix(1700000000, 0)
	collector.parseStatLine(strings.Fields("cpu0 60 0 20 20 0 0 0 0 0 0"), collector.cputags["cpu0"], output, now, time.Second)
	messages := cpustatRegressionOutput(output)
	for name, want := range map[string]float64{"cpu_user": 60, "cpu_system": 20, "cpu_idle": 20, "cpu_used": 80} {
		message := messages[name]
		if message == nil {
			t.Fatalf("missing control metric%s", name)
		}
		value, ok := message.GetMetricValue()
		if !ok || math.Abs(value.(float64)-want) > 1e-12 || !message.Time().Equal(now) {
			t.Errorf("%s=%v at%v, want%v at%v", name, value, message.Time(), want, now)
		}
		if unit, _ := message.GetTag("unit"); unit != "Percent" {
			t.Errorf("unit tag compatibility changed: %q", unit)
		}
	}
	collector.parseStatLine(strings.Fields("cpu0 160 0 20 20 0 0 0 0 60 0"), collector.cputags["cpu0"], output, now.Add(time.Second), time.Second)
	value, _ := cpustatRegressionOutput(output)["cpu_used"].GetMetricValue()
	if value != float64(160) {
		t.Errorf("unresolved legacy Guest formula changed: %v, want160", value)
	}
}
