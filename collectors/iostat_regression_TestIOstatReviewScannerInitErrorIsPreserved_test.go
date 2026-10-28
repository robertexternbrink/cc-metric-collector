package collectors

import (
	"strings"
	"testing"
)

func TestIOstatReviewScannerInitErrorIsPreserved(t *testing.T) {
	iostatReviewWrite(t, strings.Repeat("x", 70*1024)+"\n")
	if err := new(IOstatCollector).Init(nil); err == nil {
		t.Error("scanner error accepted")
	} else {
		t.Logf("real Init error return exercised:%v; no FD/resource-lifetime measurement", err)
	}
}
