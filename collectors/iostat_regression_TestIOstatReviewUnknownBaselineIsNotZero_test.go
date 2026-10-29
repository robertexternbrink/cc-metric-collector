package collectors

import (
	"testing"
)

func TestIOstatReviewUnknownBaselineIsNotZero(t *testing.T) {
	for _, fixture := range []string{"optional-missing", "invalid-value"} {
		t.Run(fixture, func(t *testing.T) {
			fields := iostatReviewCounters("100", "2")
			key := "io_discards"
			if fixture == "optional-missing" {
				fields = fields[:11]
			} else {
				fields[0] = "invalid"
				key = "io_reads"
			}
			collector := iostatReviewCollector(t, iostatReviewRow("sda", fields), `{}`)
			iostatReviewWrite(t, iostatReviewRow("sda", iostatReviewCounters("150", "2")))
			if message := iostatReviewRead(collector)[key]; message != nil {
				value, _ := message.GetMetricValue()
				t.Errorf("first valid unknown%s emitted boot-relative delta%v", key, value)
			}
			fields = iostatReviewCounters("160", "2")
			fields[11] = "160"
			iostatReviewWrite(t, iostatReviewRow("sda", fields))
			if value, _ := iostatReviewRead(collector)[key].GetMetricValue(); value != int64(10) {
				t.Errorf("second known delta%s=%v want10", key, value)
			}
		})
	}
}
