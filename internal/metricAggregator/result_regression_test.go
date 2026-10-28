package metricAggregator

import (
	cclog "github.com/ClusterCockpit/cc-lib/v2/ccLogger"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"math"
	"testing"
	"time"
)

func init() { cclog.Init("err", false) }
func auditEngine(t *testing.T, function string) (*metricAggregator, chan lp.CCMessage, []lp.CCMessage) {
	t.Helper()
	out := make(chan lp.CCMessage, 10)
	a := &metricAggregator{}
	if err := a.Init(out); err != nil {
		t.Fatal(err)
	}
	if err := a.AddAggregation("first", function, "true", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := a.AddAggregation("following", "sum(values)", "true", nil, nil); err != nil {
		t.Fatal(err)
	}
	metric, err := lp.NewMetric("input", map[string]string{"type": "node"}, nil, float64(2), time.Unix(100, 0))
	if err != nil {
		t.Fatal(err)
	}
	return a, out, []lp.CCMessage{metric}
}

func TestAuditEngineInvalidResultDoesNotPanic(t *testing.T) {
	for _, function := range []string{"sum(values) * 1e308", "len(values) > 0"} {
		t.Run(function, func(t *testing.T) {
			a, out, metrics := auditEngine(t, function)
			defer func() {
				if value := recover(); value != nil {
					t.Errorf("real Eval panicked on accepted expression %q: %v", function, value)
				}
			}()
			a.Eval(time.Unix(100, 0), time.Unix(101, 0), metrics)
			for len(out) > 0 {
				if metric := <-out; metric == nil {
					t.Fatal("nil message was emitted")
				}
			}
		})
	}
}

func TestAuditArithmeticPositiveControls(t *testing.T) {
	// Empty inputs remain errors; length-three median currently works.
	if _, err := medianAnyType([]float64{}); err == nil {
		t.Fatal("empty median accepted")
	}
	if _, err := avgAnyType([]int{}); err == nil {
		t.Fatal("empty mean accepted")
	}
	auditMedianCase(t, []int{9, 1, 3}, 3)
	if got, err := avgAnyType([]int32{1, 3}); err != nil || got != 2 {
		t.Fatalf("ordinary mean = %v, %v", got, err)
	}
	a, out, metrics := auditEngine(t, "sum(values)")
	a.Eval(time.Unix(100, 0), time.Unix(101, 0), metrics)
	if len(out) != 2 {
		t.Fatalf("valid rule count = %d; want 2", len(out))
	}
	for len(out) > 0 {
		metric := <-out
		if metric == nil {
			t.Fatal("valid rule emitted nil")
		}
		value, ok := metric.GetField("value")
		if !ok || value != float64(2) {
			t.Fatalf("valid result = %v, %v", value, ok)
		}
	}
}

func TestAuditEngineMathWithRealMessages(t *testing.T) {
	for _, item := range []struct {
		name, function string
		values         []any
		want           float64
	}{
		{"median-five", "median(values)", []any{float64(1), float64(2), float64(3), float64(4), float64(5)}, 3},
		{"mean-large-integers", "avg(values)", []any{int64(math.MaxInt64), int64(math.MaxInt64)}, float64(math.MaxInt64)},
	} {
		t.Run(item.name, func(t *testing.T) {
			out := make(chan lp.CCMessage, 1)
			a := &metricAggregator{}
			if err := a.Init(out); err != nil {
				t.Fatal(err)
			}
			if err := a.AddAggregation("result", item.function, "true", nil, nil); err != nil {
				t.Fatal(err)
			}
			metrics := make([]lp.CCMessage, 0, len(item.values))
			for _, value := range item.values {
				metric, err := lp.NewMetric("input", map[string]string{"type": "node"}, nil, value, time.Unix(100, 0))
				if err != nil {
					t.Fatal(err)
				}
				metrics = append(metrics, metric)
			}
			a.Eval(time.Unix(100, 0), time.Unix(101, 0), metrics)
			if len(out) != 1 {
				t.Fatalf("real-message result count = %d; want 1", len(out))
			}
			metric := <-out
			if metric == nil {
				t.Fatal("nil result")
			}
			value, ok := metric.GetField("value")
			if !ok || value != item.want {
				t.Fatalf("actual factory/LP result = %v, %v; want %.17g", value, ok, item.want)
			}
		})
	}
}
