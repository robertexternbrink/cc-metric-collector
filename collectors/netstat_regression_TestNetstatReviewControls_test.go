package collectors

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNetstatReviewControls(t *testing.T) {
	collector := netstatReviewCollector(t, "ignored header\n"+netstatReviewRow("ethA", "100"), `{"include_devices":["ethA"],"send_derived_values":true}`)
	if getCanonicalName("ethA", nil) != "ethA" || getCanonicalName("alias", map[string]string{"alias": "ethA"}) != "ethA" {
		t.Error("pure alias lookup changed")
	}
	messages := netstatReviewRead(collector)
	if len(messages) != 4 {
		t.Fatalf("first sample count=%d, want4 absolute only", len(messages))
	}
	for name, want := range map[string]int64{"net_bytes_in": 100, "net_bytes_out": 200, "net_pkts_in": 20, "net_pkts_out": 40} {
		message := messages[name]
		if message == nil {
			t.Fatalf("missing control%s", name)
		}
		value, _ := message.GetMetricValue()
		if value != want {
			t.Errorf("%s=%v want%d", name, value, want)
		}
		if id, _ := message.GetTag("stype-id"); id != "ethA" {
			t.Errorf("raw ID=%q", id)
		}
		unit, _ := message.GetMeta("unit")
		if strings.Contains(name, "bytes") && unit != "bytes" || strings.Contains(name, "pkts") && unit != "packets" {
			t.Errorf("unit=%q", unit)
		}
	}
	collector.Close()
	if messages := netstatReviewRead(collector); len(messages) != 0 {
		t.Error("closed collector emitted metrics")
	}
	if err := new(NetstatCollector).Init(json.RawMessage(`{"unknown":true}`)); err == nil {
		t.Error("unknown config accepted")
	}
}
