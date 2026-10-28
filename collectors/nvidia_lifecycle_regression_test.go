package collectors

import (
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"strings"
	"testing"
)

func mockNvidiaLifecycle(t *testing.T) (*int, *int) {
	t.Helper()
	init, shutdown, count, errorString := nvml.Init, nvml.Shutdown, nvml.DeviceGetCount, nvml.ErrorString
	starts, stops := new(int), new(int)
	nvml.Init = func() nvml.Return { *starts++; return nvml.SUCCESS }
	nvml.Shutdown = func() nvml.Return { *stops++; return nvml.SUCCESS }
	nvml.DeviceGetCount = func() (int, nvml.Return) { return 0, nvml.SUCCESS }
	nvml.ErrorString = func(r nvml.Return) string { return "mock NVML error" }
	t.Cleanup(func() {
		nvml.Init = init
		nvml.Shutdown = shutdown
		nvml.DeviceGetCount = count
		nvml.ErrorString = errorString
	})
	return starts, stops
}
func TestNvidiaFailedInitReleasesOnlyOwnedReference(t *testing.T) {
	for _, scenario := range []string{"init-error", "count-error", "count-panic", "success"} {
		t.Run(scenario, func(t *testing.T) {
			starts, stops := mockNvidiaLifecycle(t)
			switch scenario {
			case "init-error":
				nvml.Init = func() nvml.Return { *starts++; return nvml.ERROR_LIBRARY_NOT_FOUND }
			case "count-error":
				nvml.DeviceGetCount = func() (int, nvml.Return) { return 0, nvml.ERROR_UNKNOWN }
			case "count-panic":
				nvml.DeviceGetCount = func() (int, nvml.Return) { panic("mock count panic") }
			}
			collector := NvidiaCollector{}
			err := collector.Init(nil)
			if scenario == "success" {
				if err != nil || !collector.Initialized() || *stops != 0 {
					t.Fatalf("success err=%v initialized=%v shutdown=%d", err, collector.Initialized(), *stops)
				}
				collector.Close()
				collector.Close()
				if *stops != 1 || collector.Initialized() {
					t.Fatalf("Close ownership/count %d", *stops)
				}
			} else {
				wantStops := 1
				if scenario == "init-error" {
					wantStops = 0
				}
				if err == nil || collector.Initialized() || *stops != wantStops {
					t.Errorf("failed init err=%v initialized=%v shutdown=%d want %d", err, collector.Initialized(), *stops, wantStops)
				}
				collector.Close()
				if *stops != wantStops {
					t.Errorf("failed init Close double-release: %d", *stops)
				}
			}
			if *starts != 1 {
				t.Errorf("Init calls %d", *starts)
			}
		})
	}
}

func TestNvidiaInitPanicBeforeOwnershipDoesNotShutdown(t *testing.T) {
	_, stops := mockNvidiaLifecycle(t)
	nvml.Init = func() nvml.Return { panic("mock before successful Init") }
	collector := NvidiaCollector{}
	if err := collector.Init(nil); err == nil || collector.Initialized() || *stops != 0 {
		t.Fatalf("panic before ownership err=%v init=%v shutdown=%d", err, collector.Initialized(), *stops)
	}
}
func TestNvidiaFailedInitReportsCleanupFailure(t *testing.T) {
	_, stops := mockNvidiaLifecycle(t)
	nvml.DeviceGetCount = func() (int, nvml.Return) { return 0, nvml.ERROR_UNKNOWN }
	nvml.Shutdown = func() nvml.Return { *stops++; return nvml.ERROR_UNKNOWN }
	collector := NvidiaCollector{}
	err := collector.Init(nil)
	if err == nil || !strings.Contains(err.Error(), "cleanup failed") || *stops != 1 || collector.Initialized() {
		t.Fatalf("cleanup error lost err=%v stops=%d init=%v", err, *stops, collector.Initialized())
	}
}
