package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"strings"
	"testing"
	"time"
)

func TestCpustatExclusionsOnlyFilterOutput(t *testing.T) {
	for _, excluded := range []string{"cpu_user", "cpu_idle", "cpu_used"} {
		t.Run(excluded, func(t *testing.T) {
			collector := cpustatRegressionCollector(t, `{"exclude_metrics":["`+excluded+`"]}`)
			output := make(chan lp.CCMessage, 32)
			collector.parseStatLine(strings.Fields("cpu0 60 0 20 20 0 0 0 0 0 0"), collector.cputags["cpu0"], output, time.Unix(1700000000, 0), time.Second)
			messages := cpustatRegressionOutput(output)
			if _, found := messages[excluded]; found {
				t.Errorf("excluded %s was forwarded", excluded)
			}
			if excluded == "cpu_used" {
				return
			}
			message := messages["cpu_used"]
			if message == nil {
				t.Fatal("raw exclusion suppressed derived cpu_used")
			}
			if value, ok := message.GetMetricValue(); !ok || value != float64(80) {
				t.Errorf("raw exclusion changed cpu_used to%v, want80", value)
			}
		})
	}
}
