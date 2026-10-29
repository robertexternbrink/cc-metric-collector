package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"math"
	"strings"
	"testing"
	"time"
)

func TestNetstatReviewAliasesKeepRawIdentities(t *testing.T) {
	t.Run("simultaneous", func(t *testing.T) {
		collector := netstatReviewCollector(t, netstatReviewRow("ethA", "100")+netstatReviewRow("ethB", "1000"), `{"include_devices":["canonical"],"interface_aliases":{"canonical":["ethA","ethB"]},"send_derived_values":true}`)
		output := make(chan lp.CCMessage, 64)
		collector.Read(time.Second, output)
		ids := make(map[string]int)
		for len(output) > 0 {
			message := <-output
			if strings.HasSuffix(message.Name(), "_bw") {
				t.Errorf("first sample mixed different interfaces into rate%s", message.Name())
				continue
			}
			id, _ := message.GetTag("stype-id")
			ids[id]++
		}
		if ids["ethA"] != 4 || ids["ethB"] != 4 || len(ids) != 2 {
			t.Errorf("raw identities/counts=%v, wantA4/B4", ids)
		}
	})
	t.Run("renamed-alias", func(t *testing.T) {
		collector := netstatReviewCollector(t, netstatReviewRow("ethA", "100"), `{"include_devices":["canonical"],"interface_aliases":{"canonical":["ethA","ethB"]}}`)
		netstatReviewWrite(t, netstatReviewRow("ethB", "1000"))
		messages := netstatReviewRead(collector)
		if len(messages) != 4 {
			t.Fatalf("renamed alias produced%d messages, want4 absolute metrics", len(messages))
		}
		for _, message := range messages {
			if id, _ := message.GetTag("stype-id"); id != "ethB" {
				t.Errorf("documented actual raw name=%q wantethB", id)
			}
		}
	})
	collector := netstatReviewCollector(t, netstatReviewRow("ethA", "100")+netstatReviewRow("ethB", "1000"), `{"include_devices":["canonical","unseen"],"interface_aliases":{"canonical":["ethA","ethB"],"unseen":["ethC"]},"send_derived_values":true}`)
	previousTimes := make(map[string]time.Time)
	previousValues := make(map[string]int64)
	phases := []struct {
		data   string
		values map[string]int64
	}{
		{netstatReviewRow("ethA", "100") + netstatReviewRow("ethB", "1000"), map[string]int64{"ethA": 100, "ethB": 1000}},
		{netstatReviewRow("ethA", "120") + netstatReviewRow("ethB", "1050") + netstatReviewRow("ethC", "500"), map[string]int64{"ethA": 120, "ethB": 1050}},
		{netstatReviewRow("ethB", "1070"), map[string]int64{"ethB": 1070}},
		{netstatReviewRow("ethA", "140") + netstatReviewRow("ethB", "1100"), map[string]int64{"ethA": 140, "ethB": 1100}},
	}
	for phase, sample := range phases {
		netstatReviewWrite(t, sample.data)
		output := make(chan lp.CCMessage, 64)
		collector.Read(time.Second, output)
		absolute := make(map[string]lp.CCMessage)
		rates := make(map[string]lp.CCMessage)
		for len(output) > 0 {
			message := <-output
			id, _ := message.GetTag("stype-id")
			if _, ok := sample.values[id]; !ok {
				t.Errorf("phase%d unexpected raw/group %q", phase, id)
			}
			if message.Name() == "net_bytes_in" {
				absolute[id] = message
			}
			if message.Name() == "net_bytes_in_bw" {
				rates[id] = message
			}
		}
		for id, wantValue := range sample.values {
			message := absolute[id]
			if message == nil {
				t.Errorf("phase%d missing raw%s", phase, id)
				continue
			}
			value, _ := message.GetMetricValue()
			if value != wantValue {
				t.Errorf("phase%d raw%s absolute=%v want%d", phase, id, value, wantValue)
			}
			if last, ok := previousTimes[id]; ok {
				if rates[id] == nil {
					t.Errorf("phase%d missing raw%s rate", phase, id)
				} else {
					value, _ := rates[id].GetMetricValue()
					got := value.(float64)
					want := float64(wantValue-previousValues[id]) / message.Time().Sub(last).Seconds()
					if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > math.Max(1e-12, math.Abs(want)*1e-8) {
						t.Errorf("phase%d raw%s rate%g want%g", phase, id, got, want)
					}
				}
			} else if rates[id] != nil {
				t.Errorf("phase%d raw%s first rate", phase, id)
			}
			previousTimes[id] = message.Time()
			previousValues[id] = wantValue
		}
	}
}
