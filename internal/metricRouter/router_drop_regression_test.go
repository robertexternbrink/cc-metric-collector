package metricRouter

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	mct "github.com/ClusterCockpit/cc-metric-collector/pkg/multiChanTicker"
	"sync"
	"testing"
	"time"
)

type routerTickerDouble struct{}

func (*routerTickerDouble) Init(time.Duration)        {}
func (*routerTickerDouble) AddChannel(chan time.Time) {}
func (*routerTickerDouble) Close()                    {}

type captureCacheDouble struct{ samples chan lp.CCMessage }

func (*captureCacheDouble) Init(chan lp.CCMessage, mct.MultiChanTicker, *sync.WaitGroup, int) error {
	return nil
}
func (*captureCacheDouble) Start()               {}
func (c *captureCacheDouble) Add(m lp.CCMessage) { c.samples <- m }
func (*captureCacheDouble) GetPeriod(int) (time.Time, time.Time, []lp.CCMessage) {
	return time.Time{}, time.Time{}, nil
}
func (*captureCacheDouble) AddAggregation(string, string, string, map[string]string, map[string]string) error {
	return nil
}
func (*captureCacheDouble) DeleteAggregation(string) error { return nil }
func (*captureCacheDouble) Close()                         {}
func TestRouterDroppedCollectorMessageRemainsAvailableForAggregation(t *testing.T) {
	wg := new(sync.WaitGroup)
	r := new(metricRouter)
	if err := r.Init(&routerTickerDouble{}, wg, []byte(`{"num_cache_intervals":1,"drop_metrics":["ignored"]}`)); err != nil {
		t.Fatal(err)
	}
	cache := &captureCacheDouble{samples: make(chan lp.CCMessage, 2)}
	r.cache = cache
	input, output := make(chan lp.CCMessage, 2), make(chan lp.CCMessage, 2)
	r.AddCollectorInput(input)
	r.AddOutput(output)
	r.Start()
	defer func() { r.Close(); wg.Wait() }()
	for _, name := range []string{"ignored", "retained"} {
		message, err := lp.NewMetric(name, map[string]string{"type": "node"}, nil, 3.5, time.Now())
		if err != nil {
			t.Fatal(err)
		}
		input <- message
		select {
		case cached := <-cache.samples:
			if cached == nil {
				t.Errorf("%s became nil before cache.Add (real cache would panic)", name)
			} else if cached.Name() != name {
				t.Errorf("cache name=%s want %s", cached.Name(), name)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("cache did not receive message")
		}
	}
	if len(output) != 1 {
		t.Fatalf("drop/forward outputs=%d want1", len(output))
	}
	if m := <-output; m.Name() != "retained" {
		t.Fatalf("wrong message forwarded: %s", m.Name())
	}
}
func TestActualMetricCacheAcceptsControlMessage(t *testing.T) {
	cache := &metricCache{numPeriods: 1, intervals: []*metricCachePeriod{{}}}
	m, err := lp.NewMetric("control", map[string]string{"type": "node"}, nil, 1, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	cache.Add(m)
	if cache.intervals[0].numMetrics != 1 || cache.intervals[0].metrics[0] != m {
		t.Fatal("valid-message cache control failed")
	}
}
