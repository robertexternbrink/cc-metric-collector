package collectors

import (
	"strings"
	"testing"
	"time"

	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
)

func TestCpustatMembershipReturnsWithNewBaseline(t *testing.T) {
	cpustatRegressionFixture(t, "cpu 0 0 0 0 0 0 0 0 0 0\ncpu0 0 0 0 0 0 0 0 0 0 0\ncpu7 0 0 0 0 0 0 0 0 0 0\n")
	collector := new(CpustatCollector)
	if err := collector.Init(nil); err != nil {
		t.Fatal(err)
	}
	output := make(chan lp.CCMessage, 64)
	cpustatRegressionFixture(t, "cpu 0 0 0 0 0 0 0 0 0 0\ncpu0 0 0 0 0 0 0 0 0 0 0\n")
	collector.Read(time.Second, output)
	cpustatRegressionOutput(output)
	if _, found := collector.olddata["cpu7"]; found {
		t.Error("removed CPU retained a stale baseline")
	}
	cpustatRegressionFixture(t, "cpu 0 0 0 0 0 0 0 0 0 0\ncpu0 0 0 0 0 0 0 0 0 0 0\ncpu7 6000 0 2000 2000 0 0 0 0 0 0\n")
	collector.Read(time.Second, output)
	for len(output) > 0 {
		message := <-output
		if id, _ := message.GetTag("type-id"); id == "7" {
			t.Errorf("returning CPU emitted %s across a missing interval", message.Name())
		}
	}
}

func TestCpustatOptionalCounterReturnsWithNewBaseline(t *testing.T) {
	collector := cpustatRegressionCollector(t, "{}")
	output := make(chan lp.CCMessage, 32)
	caught := cpustatRegressionCatch(func() {
		collector.parseStatLine(strings.Fields("cpu0 60 0 20 20 0 0 0 0"), collector.cputags["cpu0"], output, time.Unix(1700000000, 0), time.Second)
	})
	if caught != nil {
		t.Fatalf("missing optional fields caused a panic: %v", caught)
	}
	cpustatRegressionOutput(output)
	collector.parseStatLine(strings.Fields("cpu0 120 0 40 40 0 0 0 0 60 0"), collector.cputags["cpu0"], output, time.Unix(1700000001, 0), time.Second)
	if _, found := cpustatRegressionOutput(output)["cpu_guest"]; found {
		t.Error("returning optional guest field emitted a rate without an adjacent baseline")
	}
}

func TestCpustatInvalidCounterDiscardsWholeSnapshot(t *testing.T) {
	collector := cpustatRegressionCollector(t, "{}")
	before := collector.lastTimestamp
	cpustatRegressionFixture(t, "cpu0 invalid 0 20 20 0 0 0 0 0 0\n")
	output := make(chan lp.CCMessage, 32)
	collector.Read(time.Second, output)
	if !collector.lastTimestamp.Equal(before) || collector.olddata["cpu0"]["cpu_system"] != 0 || len(output) != 0 {
		t.Errorf("invalid complete snapshot changed timestamp/counters/output: %v/%d/%d", !collector.lastTimestamp.Equal(before), collector.olddata["cpu0"]["cpu_system"], len(output))
	}
}
