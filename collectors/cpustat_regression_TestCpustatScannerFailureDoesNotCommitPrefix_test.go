package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"strings"
	"testing"
	"time"
)

func TestCpustatScannerFailureDoesNotCommitPrefix(t *testing.T) {
	collector := cpustatRegressionCollector(t, "{}")
	before := collector.lastTimestamp
	cpustatRegressionFixture(t, "cpu0 60 0 20 20 0 0 0 0 0 0\n"+strings.Repeat("x", 70*1024)+"\n")
	output := make(chan lp.CCMessage, 64)
	collector.Read(time.Second, output)
	if !collector.lastTimestamp.Equal(before) || collector.olddata["cpu0"]["cpu_user"] != 0 || len(output) != 0 {
		t.Errorf("partial snapshot changed timestamp/counters/output: %v/%d/%d", !collector.lastTimestamp.Equal(before), collector.olddata["cpu0"]["cpu_user"], len(output))
	}
}
