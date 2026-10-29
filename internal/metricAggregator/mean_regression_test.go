package metricAggregator

import (
	"math"
	"testing"
)

func TestAuditMeanRepresentableResult(t *testing.T) {
	// Each mean is representable in the helper's documented float64 result.
	// All inputs are supported; wrapped intermediate sums are not their mean.
	t.Run("int32-max", func(t *testing.T) {
		got, err := avgAnyType([]int32{math.MaxInt32, math.MaxInt32})
		if err != nil || got != float64(math.MaxInt32) {
			t.Fatalf("mean = %.17g, %v; want %.17g", got, err, float64(math.MaxInt32))
		}
	})
	t.Run("int32-min", func(t *testing.T) {
		got, err := avgAnyType([]int32{math.MinInt32, math.MinInt32})
		if err != nil || got != float64(math.MinInt32) {
			t.Fatalf("mean = %.17g, %v; want %.17g", got, err, float64(math.MinInt32))
		}
	})
	t.Run("int64-max", func(t *testing.T) {
		got, err := avgAnyType([]int64{math.MaxInt64, math.MaxInt64})
		if err != nil || got != float64(math.MaxInt64) {
			t.Fatalf("mean = %.17g, %v; want %.17g", got, err, float64(math.MaxInt64))
		}
	})
	t.Run("float32-max", func(t *testing.T) {
		got, err := avgAnyType([]float32{math.MaxFloat32, math.MaxFloat32})
		if err != nil || got != float64(float32(math.MaxFloat32)) {
			t.Fatalf("mean = %.17g, %v; want %.17g", got, err, float64(float32(math.MaxFloat32)))
		}
	})
}

func TestMeanRetainsIntegerCancellation(t *testing.T) {
	got, err := avgAnyType([]int64{1<<53 + 1, -(1 << 53)})
	if err != nil || got != .5 {
		t.Fatalf("mean=%v,%v; want .5", got, err)
	}
}
func TestMeanAvoidsFiniteFloatSumOverflow(t *testing.T) {
	got, err := avgAnyType([]float64{math.MaxFloat64, math.MaxFloat64})
	if err != nil || math.IsNaN(got) || math.IsInf(got, 0) || got != math.MaxFloat64 {
		t.Fatalf("mean=%v,%v", got, err)
	}
}
