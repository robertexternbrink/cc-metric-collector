package collectors

import (
	"encoding/json"
	"fmt"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func iostatReviewCounters(reads, inflight string) []string {
	fields := make([]string, 17)
	for i := range fields {
		fields[i] = "0"
	}
	fields[0], fields[2], fields[8], fields[11] = reads, "200", inflight, "150"
	return fields
}

func iostatReviewRow(name string, fields []string) string {
	return fmt.Sprintf("8 0 %s %s\n", name, strings.Join(fields, " "))
}

func iostatReviewWrite(t *testing.T, data string) {
	t.Helper()
	if strings.HasPrefix(iostatInputFile, "/proc/") {
		previous := iostatInputFile
		iostatInputFile = filepath.Join(t.TempDir(), "snapshot")
		t.Cleanup(func() { iostatInputFile = previous })
	}
	if err := os.WriteFile(iostatInputFile, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(iostatInputFile); err != nil && !os.IsNotExist(err) {
			t.Error(err)
		}
	})
}

func iostatReviewCollector(t *testing.T, data, config string) *IOstatCollector {
	t.Helper()
	iostatReviewWrite(t, data)
	collector := new(IOstatCollector)
	if err := collector.Init(json.RawMessage(config)); err != nil {
		t.Fatal(err)
	}
	return collector
}

func iostatReviewRead(collector *IOstatCollector) map[string]lp.CCMessage {
	output := make(chan lp.CCMessage, 128)
	collector.Read(99*time.Second, output)
	messages := make(map[string]lp.CCMessage)
	for len(output) > 0 {
		message := <-output
		messages[message.Name()] = message
	}
	return messages
}
