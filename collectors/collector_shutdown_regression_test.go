package collectors

import (
	"encoding/json"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"sync"
	"testing"
	"time"
)

type managerTickerDouble struct{ registered chan chan time.Time }

func (*managerTickerDouble) Init(time.Duration)            {}
func (f *managerTickerDouble) AddChannel(c chan time.Time) { f.registered <- c }
func (*managerTickerDouble) Close()                        {}

type managerCollectorDouble struct {
	metricCollector
	closed  chan struct{}
	entered chan struct{}
	release chan struct{}
}

func (f *managerCollectorDouble) Init(json.RawMessage) error { f.init = true; return nil }
func (f *managerCollectorDouble) Read(time.Duration, chan lp.CCMessage) {
	if f.entered != nil {
		close(f.entered)
		<-f.release
	}
}
func (f *managerCollectorDouble) Close() { close(f.closed); f.init = false }
func waitManagerSignal(t *testing.T, c <-chan struct{}) {
	t.Helper()
	select {
	case <-c:
	case <-time.After(2 * time.Second):
		t.Fatal("manager signal did not arrive")
	}
}
func TestCollectorManagerClosesSerialAndParallelAfterReads(t *testing.T) {
	ticker := &managerTickerDouble{registered: make(chan chan time.Time, 1)}
	wg := new(sync.WaitGroup)
	parallel := &managerCollectorDouble{metricCollector: metricCollector{name: "fake parallel", init: true, parallel: true}, closed: make(chan struct{}), entered: make(chan struct{}), release: make(chan struct{})}
	serial := &managerCollectorDouble{metricCollector: metricCollector{name: "fake serial", init: true}, closed: make(chan struct{})}
	cm := &collectorManager{collectors: []MetricCollector{parallel}, serial: []MetricCollector{serial}, ticker: ticker, wg: wg, done: make(chan bool), output: make(chan lp.CCMessage, 2)}
	cm.Start()
	tick := <-ticker.registered
	tick <- time.Now()
	waitManagerSignal(t, parallel.entered)
	complete := make(chan struct{})
	go func() { cm.Close(); close(complete) }()
	select {
	case <-parallel.closed:
		t.Fatal("parallel closed before Read finished")
	case <-serial.closed:
		t.Fatal("serial closed before active phase finished")
	default:
	}
	close(parallel.release)
	waitManagerSignal(t, complete)
	wg.Wait()
	select {
	case <-parallel.closed:
	default:
		t.Error("parallel not closed")
	}
	select {
	case <-serial.closed:
	default:
		t.Error("serial collector resources leaked")
	}
}
