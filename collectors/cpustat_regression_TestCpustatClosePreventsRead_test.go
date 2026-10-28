package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"testing"
	"time"
)

func TestCpustatClosePreventsRead(t *testing.T) {
	collector := CpustatCollector{metricCollector: metricCollector{init: true}}
	collector.Close()
	output := make(chan lp.CCMessage, 16)
	collector.Read(time.Second, output)
	if len(output) != 0 {
		t.Error("closed collector emitted metrics")
	}
}
