package metricRouter

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"sync"
	"testing"
	"time"
)

// The standard aggregator evaluates this ordinary CCMessage condition. Its
// controlled gate pauses the real evaluator, without replacing the evaluator
// or its real nonblocking output send.
type shutdownGatedMetric struct {
	lp.CCMessage
	entered, release chan struct{}
}

func (m *shutdownGatedMetric) AuditWait() bool { close(m.entered); <-m.release; return true }
func waitShutdownSignal(t *testing.T, c <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(2 * time.Second):
		t.Fatalf("%s did not complete", name)
	}
}
func TestRouterCloseKeepsConsumerAliveForActiveCacheEvaluation(t *testing.T) {
	ticker := &cacheTickerDouble{registered: make(chan chan time.Time, 1)}
	wg := new(sync.WaitGroup)
	r := new(metricRouter)
	if err := r.Init(ticker, wg, []byte(`{"num_cache_intervals":1}`)); err != nil {
		t.Fatal(err)
	}
	c := r.cache.(*metricCache)
	base := time.Unix(1700000000, 0)
	// This is the real factory-provided engine with a documented expression,
	// and its normal sum(), NewMessage(), and select/default output code.
	if err := c.AddAggregation("final-aggregate", "sum(values)", "metric.AuditWait()", map[string]string{"type": "node"}, nil); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	input, output := make(chan lp.CCMessage, 1), make(chan lp.CCMessage, 1)
	r.AddCollectorInput(input)
	r.AddOutput(output)
	r.Start()
	tick := <-ticker.registered
	tick <- base
	waitCacheStamp(t, c, base)
	c.Add(&shutdownGatedMetric{CCMessage: cacheFixtureMessage(t, "raw", base), entered: entered, release: release})
	tick <- base.Add(time.Second)
	waitShutdownSignal(t, entered, "real evaluator condition entry")
	routerStopped := make(chan struct{})
	go func() { wg.Wait(); close(routerStopped) }()
	closeComplete := make(chan struct{})
	go func() { r.Close(); close(closeComplete) }()
	select {
	case <-routerStopped:
		t.Error("router consumer stopped while standard cache evaluation still active")
	case <-time.After(60 * time.Millisecond):
	}
	close(release)
	waitShutdownSignal(t, closeComplete, "Router.Close")
	waitShutdownSignal(t, routerStopped, "router waitgroup")
	// The old implementation finishes too, but the actual engine drops the
	// final result because its nonblocking send no longer has a consumer.
	if len(output) != 1 {
		t.Fatalf("final standard-engine result lost, outputcount=%d", len(output))
	}
	result := <-output
	if result.Name() != "final-aggregate" {
		t.Errorf("wrong final metric %s", result.Name())
	}
	value, _ := result.GetMetricValue()
	if value != int64(1) {
		t.Errorf("final sum=%v (%T) want1", value, value)
	}
}
