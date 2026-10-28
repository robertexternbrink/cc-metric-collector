// Copyright (C) NHR@FAU, University Erlangen-Nuremberg.
// All rights reserved. This file is part of cc-lib.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.
// additional authors:
// Holger Obermaier (NHR@KIT)

package metricRouter

import (
	"fmt"
	"slices"
	"sync"
	"time"

	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"

	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	agg "github.com/ClusterCockpit/cc-metric-collector/internal/metricAggregator"
	mct "github.com/ClusterCockpit/cc-metric-collector/pkg/multiChanTicker"
)

type metricCachePeriod struct {
	startstamp  time.Time
	stopstamp   time.Time
	numMetrics  int
	sizeMetrics int
	metrics     []lp.CCMessage
}

// Metric cache data structure
type metricCache struct {
	numPeriods int
	curPeriod  int
	lock       sync.Mutex
	intervals  []*metricCachePeriod
	wg         *sync.WaitGroup
	ticker     mct.MultiChanTicker
	tickchan   chan time.Time
	done       chan bool
	output     chan lp.CCMessage
	aggEngine  agg.MetricAggregator
}

type MetricCache interface {
	Init(output chan lp.CCMessage, ticker mct.MultiChanTicker, wg *sync.WaitGroup, numPeriods int) error
	Start()
	Add(metric lp.CCMessage)
	GetPeriod(index int) (time.Time, time.Time, []lp.CCMessage)
	AddAggregation(name, function, condition string, tags, meta map[string]string) error
	DeleteAggregation(name string) error
	Close()
}

func (c *metricCache) Init(output chan lp.CCMessage, ticker mct.MultiChanTicker, wg *sync.WaitGroup, numPeriods int) error {
	var err error
	c.done = make(chan bool)
	c.wg = wg
	c.ticker = ticker
	c.numPeriods = numPeriods
	c.output = output
	c.intervals = make([]*metricCachePeriod, 0)
	for i := 0; i < c.numPeriods+1; i++ {
		p := new(metricCachePeriod)
		p.numMetrics = 0
		p.sizeMetrics = 0
		p.metrics = make([]lp.CCMessage, 0)
		c.intervals = append(c.intervals, p)
	}

	// Create a new aggregation engine. No separate goroutine at the moment
	// The code is executed by the MetricCache goroutine
	c.aggEngine, err = agg.NewAggregator(c.output)
	if err != nil {
		return fmt.Errorf("MetricCache: failed to create aggregator: %w", err)
	}

	return nil
}

// Start starts the metric cache
func (c *metricCache) Start() {
	c.tickchan = make(chan time.Time)
	c.ticker.AddChannel(c.tickchan)
	// Router cache is done
	done := func() {
		cclog.ComponentDebug("MetricCache", "DONE")
		close(c.done)
	}

	// Capture the completed interval before advancing or reusing its slot.
	rotate := func(timestamp time.Time) (time.Time, time.Time, []lp.CCMessage) {
		c.intervals[c.curPeriod].stopstamp = timestamp
		start, stop, metrics := c.getPeriodLocked(0)
		c.curPeriod = (c.curPeriod + 1) % len(c.intervals)
		next := c.intervals[c.curPeriod]
		next.numMetrics = 0
		next.startstamp = timestamp
		next.stopstamp = timestamp
		return start, stop, metrics
	}

	c.wg.Go(func() {
		for {
			select {
			case <-c.done:
				done()
				return
			case tick := <-c.tickchan:
				c.lock.Lock()
				starttime, endtime, metrics := rotate(tick)
				c.lock.Unlock()
				if len(metrics) > 0 {
					c.aggEngine.Eval(starttime, endtime, metrics)
				} else {
					// This message is also printed in the first interval after startup
					cclog.ComponentDebug("MetricCache", "EMPTY INTERVAL?")
				}
			}
		}
	})
	cclog.ComponentDebug("MetricCache", "START")
}

// Add a metric to the cache. The interval is defined by the global timer (rotate() in Start())
// The intervals list is used as round-robin buffer and the metric list grows dynamically and
// to avoid reallocations
func (c *metricCache) Add(metric lp.CCMessage) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.curPeriod >= 0 && c.curPeriod < len(c.intervals) {
		p := c.intervals[c.curPeriod]
		if p.numMetrics < p.sizeMetrics {
			p.metrics[p.numMetrics] = metric
		} else {
			p.metrics = append(p.metrics, metric)
			p.sizeMetrics++
		}
		p.numMetrics++
		p.stopstamp = metric.Time()
	}
}

func (c *metricCache) AddAggregation(name, function, condition string, tags, meta map[string]string) error {
	return c.aggEngine.AddAggregation(name, function, condition, tags, meta)
}

func (c *metricCache) DeleteAggregation(name string) error {
	return c.aggEngine.DeleteAggregation(name)
}

// Get all metrics of a interval. The index is the difference to the current interval, so index=0
// is the current one, index=1 the last interval and so on. Returns and empty array if a wrong index
// is given (negative index, index larger than configured number of total intervals, ...)
func (c *metricCache) GetPeriod(index int) (time.Time, time.Time, []lp.CCMessage) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.getPeriodLocked(index)
}

// The ring has one current slot and numPeriods completed slots. The caller
// holds c.lock; return an independent slice so slot reuse cannot alter it.
func (c *metricCache) getPeriodLocked(index int) (time.Time, time.Time, []lp.CCMessage) {
	if index < 0 || index > c.numPeriods || len(c.intervals) == 0 {
		now := time.Now()
		return now, now, []lp.CCMessage{}
	}
	pindex := (c.curPeriod - index + len(c.intervals)) % len(c.intervals)
	p := c.intervals[pindex]
	return p.startstamp, p.stopstamp, slices.Clone(p.metrics[:p.numMetrics])
}

// Close finishes / stops the metric cache
func (c *metricCache) Close() {
	cclog.ComponentDebug("MetricCache", "CLOSE")
	c.done <- true
}

func NewCache(output chan lp.CCMessage, ticker mct.MultiChanTicker, wg *sync.WaitGroup, numPeriods int) (MetricCache, error) {
	c := new(metricCache)
	err := c.Init(output, ticker, wg, numPeriods)
	if err != nil {
		return nil, err
	}
	return c, err
}
