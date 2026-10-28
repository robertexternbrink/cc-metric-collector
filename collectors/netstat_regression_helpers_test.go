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

func netstatReviewRow(name, received string) string {
	return fmt.Sprintf("%s: %s 20 0 0 0 0 0 0 200 40 0 0 0 0 0 0\n", name, received)
}

func netstatReviewWrite(t *testing.T, data string) {
	t.Helper()
	if strings.HasPrefix(netstatInputFile, "/proc/") {
		previous := netstatInputFile
		netstatInputFile = filepath.Join(t.TempDir(), "snapshot")
		t.Cleanup(func() { netstatInputFile = previous })
	}
	if err := os.WriteFile(netstatInputFile, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(netstatInputFile); err != nil && !os.IsNotExist(err) {
			t.Error(err)
		}
	})
}

func netstatReviewCollector(t *testing.T, data, config string) *NetstatCollector {
	t.Helper()
	netstatReviewWrite(t, data)
	collector := new(NetstatCollector)
	if err := collector.Init(json.RawMessage(config)); err != nil {
		t.Fatal(err)
	}
	return collector
}

func netstatReviewRead(collector *NetstatCollector) map[string]lp.CCMessage {
	output := make(chan lp.CCMessage, 64)
	collector.Read(99*time.Second, output)
	messages := make(map[string]lp.CCMessage)
	for len(output) > 0 {
		message := <-output
		messages[message.Name()] = message
	}
	return messages
}

func netstatReviewCatch(call func()) (caught any) {
	defer func() { caught = recover() }()
	call()
	return nil
}
