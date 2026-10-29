package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMemstatMissingValuesRemainMissing(t *testing.T) {
	for _, scenario := range []string{"missing-file", "missing-fields", "real-zero"} {
		t.Run(scenario, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "meminfo")
			if scenario != "missing-file" {
				data := "Node 0 MemTotal: 1024 kB\n"
				if scenario == "real-zero" {
					data += "Node 0 MemFree: 0 kB\n"
				}
				if err := os.WriteFile(path, []byte(data), 0600); err != nil {
					t.Fatal(err)
				}
			}
			collector := MemstatCollector{metricCollector: metricCollector{init: true}, matches: map[string]string{"MemTotal": "mem_total", "MemFree": "mem_free"}, sendMemUsed: true, config: MemstatCollectorConfig{NumaStats: true}, nodefiles: map[int]MemstatCollectorNode{0: {file: path, tags: map[string]string{"type": "memoryDomain", "type-id": "0"}}}}
			out := make(chan lp.CCMessage, 8)
			collector.Read(time.Second, out)
			values := memoryFixtureValues(out)
			switch scenario {
			case "missing-file":
				if len(values) != 0 {
					t.Fatalf("read failure fabricated measurements: %v", values)
				}
			case "missing-fields":
				if len(values) != 1 || values["mem_total"] != 1024 {
					t.Fatalf("missing fields fabricated zeros/derived memory: %v", values)
				}
			case "real-zero":
				if value, exists := values["mem_free"]; !exists || value != 0 {
					t.Fatalf("valid zero dropped: %v", values)
				}
			}
		})
	}
}

func memoryFixtureValues(output chan lp.CCMessage) map[string]float64 {
	values := make(map[string]float64)
	for len(output) > 0 {
		message := <-output
		value, _ := message.GetMetricValue()
		switch number := value.(type) {
		case float64:
			values[message.Name()] = number
		case int64:
			values[message.Name()] = float64(number)
		}
	}
	return values
}
