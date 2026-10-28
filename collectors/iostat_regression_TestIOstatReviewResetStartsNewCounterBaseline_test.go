package collectors

import (
	"testing"
)

func TestIOstatReviewResetStartsNewCounterBaseline(t *testing.T) {
	collector := iostatReviewCollector(t, iostatReviewRow("sda", iostatReviewCounters("100", "2")), `{}`)
	iostatReviewWrite(t, iostatReviewRow("sda", iostatReviewCounters("5", "2")))
	if message := iostatReviewRead(collector)["io_reads"]; message != nil {
		value, _ := message.GetMetricValue()
		t.Errorf("counter reset emitted delta%v instead of rebasing", value)
	}
	iostatReviewWrite(t, iostatReviewRow("sda", iostatReviewCounters("15", "2")))
	if value, _ := iostatReviewRead(collector)["io_reads"].GetMetricValue(); value != int64(10) {
		t.Errorf("post-reset delta=%v want10", value)
	}
}
