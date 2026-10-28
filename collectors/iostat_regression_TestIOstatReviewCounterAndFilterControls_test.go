package collectors

import (
	"encoding/json"
	"os"
	"testing"
)

func TestIOstatReviewCounterAndFilterControls(t *testing.T) {
	collector := iostatReviewCollector(t, iostatReviewRow("sda", iostatReviewCounters("100", "2"))+iostatReviewRow("sdb", iostatReviewCounters("1000", "3"))+iostatReviewRow("loop0", iostatReviewCounters("1000", "3")), `{"exclude_devices":["sdb"],"exclude_metrics":["io_read_ms","io_ioops"]}`)
	if len(collector.devices) != 1 || !collector.Initialized() || !collector.Parallel() {
		t.Fatal("device filter/lifecycle differs")
	}
	fields := iostatReviewCounters("110", "2")
	fields[2] = "230"
	iostatReviewWrite(t, iostatReviewRow("sda", fields))
	messages := iostatReviewRead(collector)
	for name, want := range map[string]int64{"io_reads": 10, "io_read_sectors": 30} {
		value, _ := messages[name].GetMetricValue()
		if value != want {
			t.Errorf("counter delta%s=%v want%d", name, value, want)
		}
		if device, _ := messages[name].GetTag("device"); device != "sda" {
			t.Errorf("raw device=%q", device)
		}
	}
	if messages["io_read_ms"] != nil || messages["io_ioops"] != nil {
		t.Error("excluded metric forwarded")
	}
	if err := os.Remove(iostatInputFile); err != nil {
		t.Fatal(err)
	}
	if messages := iostatReviewRead(collector); len(messages) != 0 {
		t.Error("missing own fixture emitted metrics")
	}
	iostatReviewWrite(t, iostatReviewRow("sda", iostatReviewCounters("130", "2")))
	if value, _ := iostatReviewRead(collector)["io_reads"].GetMetricValue(); value != int64(20) {
		t.Errorf("cumulative delta across missing read=%v want20", value)
	}
	collector.Close()
	if messages := iostatReviewRead(collector); len(messages) != 0 {
		t.Error("closed collector emitted metrics")
	}
	if err := new(IOstatCollector).Init(json.RawMessage(`{"unknown":true}`)); err == nil {
		t.Error("unknown config accepted")
	}
	zero := iostatReviewCollector(t, iostatReviewRow("sda", iostatReviewCounters("0", "0")), `{}`)
	iostatReviewWrite(t, iostatReviewRow("sda", iostatReviewCounters("10", "0")))
	if message := iostatReviewRead(zero)["io_reads"]; message == nil {
		t.Error("known zero baseline suppressed")
	} else if value, _ := message.GetMetricValue(); value != int64(10) {
		t.Errorf("known zero→10 delta=%v", value)
	}
}
