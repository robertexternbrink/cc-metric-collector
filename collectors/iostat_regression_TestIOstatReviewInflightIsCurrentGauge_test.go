package collectors

import (
	"testing"
)

func TestIOstatReviewInflightIsCurrentGauge(t *testing.T) {
	collector := iostatReviewCollector(t, iostatReviewRow("sda", iostatReviewCounters("100", "2")), `{}`)
	for _, current := range []string{"2", "0"} {
		iostatReviewWrite(t, iostatReviewRow("sda", iostatReviewCounters("100", current)))
		message := iostatReviewRead(collector)["io_ioops"]
		value, _ := message.GetMetricValue()
		want := int64(2)
		if current == "0" {
			want = 0
		}
		if value != want {
			t.Errorf("inflight current%s emitted%v want%d", current, value, want)
		}
	}
	unknownGauge := iostatReviewCollector(t, iostatReviewRow("sda", iostatReviewCounters("100", "2")[:8]), `{}`)
	iostatReviewWrite(t, iostatReviewRow("sda", iostatReviewCounters("100", "7")))
	if message := iostatReviewRead(unknownGauge)["io_ioops"]; message == nil {
		t.Error("first current gauge missing")
	} else if value, _ := message.GetMetricValue(); value != int64(7) {
		t.Errorf("first current gauge=%v want7", value)
	}
}
