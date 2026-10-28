package collectors

import (
	"encoding/json"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cpustatRegressionFixture(t *testing.T, data string) {
	t.Helper()
	if strings.HasPrefix(cpustatInputFile, "/proc/") {
		previous := cpustatInputFile
		cpustatInputFile = filepath.Join(t.TempDir(), "snapshot")
		t.Cleanup(func() { cpustatInputFile = previous })
	}
	if err := os.WriteFile(cpustatInputFile, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(cpustatInputFile); err != nil && !os.IsNotExist(err) {
			t.Error(err)
		}
	})
}

func cpustatRegressionCollector(t *testing.T, config string) *CpustatCollector {
	t.Helper()
	cpustatRegressionFixture(t, "cpu 0 0 0 0 0 0 0 0 0 0\ncpu0 0 0 0 0 0 0 0 0 0 0\n")
	collector := new(CpustatCollector)
	if err := collector.Init(json.RawMessage(config)); err != nil {
		t.Fatal(err)
	}
	return collector
}

func cpustatRegressionOutput(output chan lp.CCMessage) map[string]lp.CCMessage {
	messages := make(map[string]lp.CCMessage)
	for len(output) > 0 {
		message := <-output
		messages[message.Name()] = message
	}
	return messages
}

func cpustatRegressionCatch(call func()) (caught any) {
	defer func() { caught = recover() }()
	call()
	return nil
}
