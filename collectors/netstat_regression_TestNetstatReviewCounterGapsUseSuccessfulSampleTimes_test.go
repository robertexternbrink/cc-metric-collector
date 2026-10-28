package collectors

import (
	"math"
	"os"
	"testing"
)

func TestNetstatReviewCounterGapsUseSuccessfulSampleTimes(t *testing.T) {
	for _, gap := range []string{"missing-file", "missing-interface", "invalid-counter"} {
		t.Run(gap, func(t *testing.T) {
			collector := netstatReviewCollector(t, netstatReviewRow("ethA", "100"), `{"include_devices":["ethA"],"send_derived_values":true}`)
			first := netstatReviewRead(collector)["net_bytes_in"]
			switch gap {
			case "missing-file":
				if err := os.Remove(netstatInputFile); err != nil {
					t.Fatal(err)
				}
			case "missing-interface":
				netstatReviewWrite(t, netstatReviewRow("other", "110"))
			case "invalid-counter":
				netstatReviewWrite(t, netstatReviewRow("ethA", "invalid"))
			}
			netstatReviewRead(collector)
			netstatReviewWrite(t, netstatReviewRow("ethA", "120"))
			messages := netstatReviewRead(collector)
			last, rateMessage := messages["net_bytes_in"], messages["net_bytes_in_bw"]
			if last == nil || rateMessage == nil {
				t.Fatal("missing final absolute or rate sample")
			}
			want := 20 / last.Time().Sub(first.Time()).Seconds()
			value, _ := rateMessage.GetMetricValue()
			got := value.(float64)
			if math.IsNaN(got) || math.IsInf(got, 0) || math.Abs(got-want) > math.Max(1e-12, math.Abs(want)*1e-8) {
				t.Errorf("20 bytes over observed successful-sample times: rate%g want%g", got, want)
			}
		})
	}
}
