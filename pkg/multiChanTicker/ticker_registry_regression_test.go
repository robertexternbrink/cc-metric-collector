package multiChanTicker

import (
	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	"testing"
	"time"
)

func init() { cclog.Init("error", false) }

func TestTickerConcurrentRegistrationAndDispatch(t *testing.T) {
	ticker := new(multiChanTicker)
	ticker.Init(time.Millisecond)
	first := make(chan time.Time)
	ticker.AddChannel(first)
	defer ticker.Close()
	for i := 0; i < 80; i++ {
		select {
		case <-first:
		case <-time.After(2 * time.Second):
			t.Fatal("tick not delivered")
		}
		ticker.AddChannel(make(chan time.Time, 128))
	}
}
func TestTickerCloseStopsNativeTimer(t *testing.T) {
	ticker := new(multiChanTicker)
	ticker.Init(time.Millisecond)
	ticker.Close()
	select {
	case <-ticker.ticker.C:
		t.Error("native timer still ticks after Close")
	case <-time.After(5 * time.Millisecond):
	}
}
func TestTickerCloseInterruptsUndrainedConsumer(t *testing.T) {
	ticker := new(multiChanTicker)
	ticker.Init(time.Millisecond)
	ticker.AddChannel(make(chan time.Time))
	done := make(chan struct{})
	go func() { ticker.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close blocked on undrained consumer")
	}
}
