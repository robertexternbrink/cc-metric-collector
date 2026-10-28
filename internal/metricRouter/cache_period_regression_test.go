package metricRouter

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"runtime"
	"sync"
	"testing"
	"time"
)

type cacheTickerDouble struct{ registered chan chan time.Time }

func (*cacheTickerDouble) Init(time.Duration)            {}
func (f *cacheTickerDouble) AddChannel(c chan time.Time) { f.registered <- c }
func (*cacheTickerDouble) Close()                        {}

type aggregationSnapshot struct {
	start, end time.Time
	metrics    []lp.CCMessage
}
type aggregationDouble struct{ samples chan aggregationSnapshot }

func (*aggregationDouble) Init(chan lp.CCMessage) error { return nil }
func (*aggregationDouble) AddAggregation(string, string, string, map[string]string, map[string]string) error {
	return nil
}
func (*aggregationDouble) DeleteAggregation(string) error { return nil }
func (f *aggregationDouble) Eval(start, end time.Time, metrics []lp.CCMessage) {
	f.samples <- aggregationSnapshot{start, end, metrics}
}
func cacheFixtureMessage(t *testing.T, name string, stamp time.Time) lp.CCMessage {
	t.Helper()
	m, e := lp.NewMetric(name, map[string]string{"type": "node"}, nil, 1, stamp)
	if e != nil {
		t.Fatal(e)
	}
	return m
}
func waitCacheStamp(t *testing.T, c *metricCache, stamp time.Time) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c.lock.Lock()
		ready := c.intervals[c.curPeriod].startstamp == stamp
		c.lock.Unlock()
		if ready {
			return
		}
		runtime.Gosched()
	}
	t.Fatal("cache did not finish controlled rotation")
}
func TestMetricCacheRelativePeriodsWrapAndExcludeStaleSlots(t *testing.T) {
	c := &metricCache{numPeriods: 3, curPeriod: 1, intervals: make([]*metricCachePeriod, 4)}
	for i := range c.intervals {
		m := cacheFixtureMessage(t, string(rune('A'+i)), time.Unix(int64(i), 0))
		stale := cacheFixtureMessage(t, "stale", time.Unix(9, 0))
		c.intervals[i] = &metricCachePeriod{metrics: []lp.CCMessage{m, stale}, numMetrics: 1, sizeMetrics: 2}
	}
	for index, want := range []string{"B", "A", "D", "C"} {
		_, _, got := c.GetPeriod(index)
		if len(got) != 1 || got[0].Name() != want {
			t.Errorf("index %d got %v want [%s]", index, got, want)
		}
	}
	for _, index := range []int{-1, 4} {
		_, _, got := c.GetPeriod(index)
		if len(got) != 0 {
			t.Errorf("invalid index%d returned %v", index, got)
		}
	}
	_, _, snapshot := c.GetPeriod(0)
	if len(snapshot) > 0 {
		snapshot[0] = nil
		if c.intervals[1].metrics[0] == nil {
			t.Error("GetPeriod exposes mutable backing array")
		}
	}
}
func TestMetricCacheAggregatesEachCompletedIntervalOnce(t *testing.T) {
	ticker := &cacheTickerDouble{registered: make(chan chan time.Time, 1)}
	wg := new(sync.WaitGroup)
	c := new(metricCache)
	if err := c.Init(make(chan lp.CCMessage, 8), ticker, wg, 3); err != nil {
		t.Fatal(err)
	}
	engine := &aggregationDouble{samples: make(chan aggregationSnapshot, 16)}
	c.aggEngine = engine
	c.Start()
	tick := <-ticker.registered
	defer func() { c.Close(); wg.Wait() }()
	base := time.Unix(1700000000, 0)
	tick <- base
	waitCacheStamp(t, c, base)
	for i, name := range []string{"A", "B", "C", "D", "E"} {
		start := base.Add(time.Duration(i) * 10 * time.Second)
		end := start.Add(10 * time.Second)
		c.Add(cacheFixtureMessage(t, name, start.Add(time.Second)))
		tick <- end
		waitCacheStamp(t, c, end)
		select {
		case result := <-engine.samples:
			if len(result.metrics) != 1 || result.metrics[0].Name() != name || result.start != start || result.end != end {
				t.Errorf("interval%s wrong snapshot %+v", name, result)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("completed interval%s missing", name)
		}
	}
	end := base.Add(60 * time.Second)
	tick <- end
	waitCacheStamp(t, c, end)
	select {
	case result := <-engine.samples:
		t.Errorf("empty interval replayed stale metrics: %+v", result)
	default:
	}
	_, _, last := c.GetPeriod(1)
	if len(last) != 0 {
		t.Errorf("empty completed interval kept stale tail: %v", last)
	}
}
func TestMetricCacheAddAndRotationAreSynchronized(t *testing.T) {
	ticker := &cacheTickerDouble{registered: make(chan chan time.Time, 1)}
	wg := new(sync.WaitGroup)
	c := new(metricCache)
	if err := c.Init(make(chan lp.CCMessage, 8), ticker, wg, 3); err != nil {
		t.Fatal(err)
	}
	c.aggEngine = &aggregationDouble{samples: make(chan aggregationSnapshot, 64)}
	c.Start()
	tick := <-ticker.registered
	defer func() { c.Close(); wg.Wait() }()
	base := time.Unix(1700000000, 0)
	tick <- base
	waitCacheStamp(t, c, base)
	sample := cacheFixtureMessage(t, "concurrent", base)
	start := make(chan struct{})
	finished := make(chan struct{})
	go func() {
		<-start
		for i := 0; i < 5000; i++ {
			c.Add(sample)
			runtime.Gosched()
		}
		close(finished)
	}()
	close(start)
	for i := 1; i <= 30; i++ {
		tick <- base.Add(time.Duration(i) * time.Second)
		c.GetPeriod(0)
	}
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		t.Fatal("Add worker blocked")
	}
}

func TestMetricCacheOnePastIntervalAndStableSnapshot(t *testing.T) {
	ticker := &cacheTickerDouble{registered: make(chan chan time.Time, 1)}
	wg := new(sync.WaitGroup)
	c := new(metricCache)
	if err := c.Init(make(chan lp.CCMessage, 8), ticker, wg, 1); err != nil {
		t.Fatal(err)
	}
	engine := &aggregationDouble{samples: make(chan aggregationSnapshot, 8)}
	c.aggEngine = engine
	c.Start()
	tick := <-ticker.registered
	defer func() { c.Close(); wg.Wait() }()
	base := time.Unix(1700000000, 0)
	tick <- base
	waitCacheStamp(t, c, base)
	for i, name := range []string{"A", "B", "C"} {
		start := base.Add(time.Duration(i) * time.Second)
		c.Add(cacheFixtureMessage(t, name, start))
		if i == 0 {
			c.Add(cacheFixtureMessage(t, "A-second", start))
		}
		end := start.Add(time.Second)
		tick <- end
		waitCacheStamp(t, c, end)
		select {
		case result := <-engine.samples:
			wantLength := 1
			if i == 0 {
				wantLength = 2
			}
			if len(result.metrics) != wantLength || result.metrics[0].Name() != name {
				t.Fatalf("one past period %s snapshot %+v", name, result)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("one-period aggregation missing")
		}
		_, _, previous := c.GetPeriod(1)
		if len(previous) == 0 || previous[0].Name() != name {
			t.Fatalf("last interval missing: %v", previous)
		}
	}
	_, _, previous := c.GetPeriod(1)
	tick <- base.Add(4 * time.Second)
	waitCacheStamp(t, c, base.Add(4*time.Second))
	if previous[0].Name() != "C" {
		t.Fatal("slot reuse mutated earlier snapshot")
	}
	select {
	case extra := <-engine.samples:
		t.Fatalf("empty interval replayed %v", extra)
	default:
	}
}
