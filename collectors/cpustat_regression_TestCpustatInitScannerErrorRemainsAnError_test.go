package collectors

import (
	"strings"
	"testing"
)

func TestCpustatInitScannerErrorRemainsAnError(t *testing.T) {
	cpustatRegressionFixture(t, strings.Repeat("x", 70*1024)+"\n")
	if err := new(CpustatCollector).Init(nil); err == nil {
		t.Error("Init accepted a Scanner failure")
	}
}
