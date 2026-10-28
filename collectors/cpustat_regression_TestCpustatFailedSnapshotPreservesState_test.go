package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"os"
	"testing"
	"time"
)

func TestCpustatFailedSnapshotPreservesState(t *testing.T) {
	collector := cpustatRegressionCollector(t, "{}")
	before := collector.lastTimestamp
	if err := os.Remove(cpustatInputFile); err != nil {
		t.Fatal(err)
	}
	output := make(chan lp.CCMessage, 64)
	collector.Read(time.Second, output)
	if !collector.lastTimestamp.Equal(before) || collector.olddata["cpu0"]["cpu_user"] != 0 || len(output) != 0 {
		t.Errorf("missing snapshot changed timestamp/counters/output: %v/%d/%d", !collector.lastTimestamp.Equal(before), collector.olddata["cpu0"]["cpu_user"], len(output))
	}
}
