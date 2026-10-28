package collectors

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNetstatReviewInitScannerErrorIsRejected(t *testing.T) {
	netstatReviewWrite(t, netstatReviewRow("ethA", "100")+strings.Repeat("x", 70*1024)+"\n"+netstatReviewRow("ethB", "1000"))
	collector := new(NetstatCollector)
	if err := collector.Init(json.RawMessage(`{"include_devices":["ethA","ethB"]}`)); err == nil {
		t.Errorf("Init accepted scanner failure and incomplete device set%v", collector.matches)
	}
}
