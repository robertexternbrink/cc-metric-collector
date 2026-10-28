package collectors

import (
	"errors"
	"os"
	"strings"
	"testing"
)

func TestCpustatScannerFailureClosesFile(t *testing.T) {
	cpustatRegressionFixture(t, strings.Repeat("x", 70*1024)+"\n")
	original := cpustatOpenFile
	var opened *os.File
	cpustatOpenFile = func(path string) (*os.File, error) { f, err := os.Open(path); opened = f; return f, err }
	t.Cleanup(func() {
		cpustatOpenFile = original
		if opened != nil {
			opened.Close()
		}
	})
	if err := new(CpustatCollector).Init(nil); err == nil {
		t.Fatal("scanner error not returned")
	}
	if opened == nil {
		t.Fatal("fixture was not opened")
	}
	_, err := opened.Read(make([]byte, 1))
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("Init leaked its opened file: %v", err)
	}
}
