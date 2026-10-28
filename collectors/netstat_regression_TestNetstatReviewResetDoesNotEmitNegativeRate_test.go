package collectors

import (
	"math"
	"testing"
)

func TestNetstatReviewResetDoesNotEmitNegativeRate(t *testing.T) {
	collector := netstatReviewCollector(t, netstatReviewRow("ethA", "100"), `{"include_devices":["ethA"],"send_derived_values":true}`)
	netstatReviewRead(collector)
	netstatReviewWrite(t, netstatReviewRow("ethA", "5"))
	messages := netstatReviewRead(collector)
	if value, _ := messages["net_bytes_in"].GetMetricValue(); value != int64(5) {
		t.Errorf("reset absolute=%v want5", value)
	}
	if message := messages["net_bytes_in_bw"]; message != nil {
		value, _ := message.GetMetricValue()
		t.Errorf("reset emitted rate%v instead of establishing a new baseline", value)
	}
	resetTime := messages["net_bytes_in"].Time()
	netstatReviewWrite(t, netstatReviewRow("ethA", "15"))
	next := netstatReviewRead(collector)
	if next["net_bytes_in_bw"] == nil || next["net_bytes_in"] == nil {
		t.Fatal("missing rate after reset baseline")
	}
	want := 10 / next["net_bytes_in"].Time().Sub(resetTime).Seconds()
	value, _ := next["net_bytes_in_bw"].GetMetricValue()
	got := value.(float64)
	if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > math.Max(1e-12, math.Abs(want)*1e-8) {
		t.Errorf("rate after reset=%g want%g", got, want)
	}
}
