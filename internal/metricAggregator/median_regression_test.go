package metricAggregator

import (
	"fmt"
	"testing"
)

func auditMedianCase[T float64 | float32 | int | int32 | int64](t *testing.T, values []T, want T) {
	t.Helper()
	defer func() {
		if value := recover(); value != nil {
			t.Errorf("median(%v) panicked: %v", values, value)
		}
	}()
	got, err := medianAnyType(values)
	if err != nil || got != want {
		t.Errorf("median(%v) = %v, %v; want %v", values, got, err, want)
	}
}

func TestAuditMedianMathematicalContract(t *testing.T) {
	// Integer cases have integral medians, so the test imposes no new
	// fractional-result or integer-return contract.
	for _, item := range []struct {
		values []int
		want   int
	}{
		{[]int{7}, 7},
		{[]int{3, 1}, 2},
		{[]int{3, 1, 2}, 2},
		{[]int{5, 1, 4, 2, 3}, 3},
		{[]int{8, 1, 7, 2, 6, 3}, 4},
	} {
		t.Run(fmt.Sprintf("int-length-%d", len(item.values)), func(t *testing.T) {
			auditMedianCase(t, item.values, item.want)
		})
	}
	t.Run("int32", func(t *testing.T) { auditMedianCase(t, []int32{1, 2, 3, 4, 5}, int32(3)) })
	t.Run("int64", func(t *testing.T) { auditMedianCase(t, []int64{1, 3}, int64(2)) })
	t.Run("float32", func(t *testing.T) { auditMedianCase(t, []float32{1.5, 2.5}, float32(2)) })
	t.Run("float64", func(t *testing.T) { auditMedianCase(t, []float64{1, 2, 3, 4, 5}, float64(3)) })
}
