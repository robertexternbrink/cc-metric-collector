package collectors

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCPUInfoScannerFailureClosesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpuinfo")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 70*1024)), 0600); err != nil {
		t.Fatal(err)
	}
	original := cpuInfoOpenFile
	var opened *os.File
	cpuInfoOpenFile = func(path string) (*os.File, error) { f, err := os.Open(path); opened = f; return f, err }
	t.Cleanup(func() {
		cpuInfoOpenFile = original
		if opened != nil {
			opened.Close()
		}
	})
	if err := (&CPUFreqCpuInfoCollector{cpuInfoFile: path}).Init(nil); err == nil {
		t.Fatal("scanner error not returned")
	}
	if opened == nil {
		t.Fatal("fixture not opened")
	}
	_, err := opened.Read(make([]byte, 1))
	if !errors.Is(err, os.ErrClosed) {
		t.Fatalf("Init leaked file: %v", err)
	}
}
