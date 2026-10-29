package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"github.com/ClusterCockpit/go-rocm-smi/pkg/rocm_smi"
	"strings"
	"testing"
	"time"
)

func mockRocmLifecycle(t *testing.T) (*int, *int) {
	t.Helper()
	init, shutdown, count, handle, pci, serial, metrics, status := rocmSmiInit, rocmSmiShutdown, rocmSmiDeviceCount, rocmSmiDeviceHandle, rocmSmiPciInfo, rocmSmiSerial, rocmSmiMetrics, rocmSmiStatusString
	stops, reads := new(int), new(int)
	rocmSmiInit = func() rocm_smi.RSMI_status { return rocm_smi.STATUS_SUCCESS }
	rocmSmiShutdown = func() rocm_smi.RSMI_status { *stops++; return rocm_smi.STATUS_SUCCESS }
	rocmSmiDeviceCount = func() (int, rocm_smi.RSMI_status) { return 1, rocm_smi.STATUS_SUCCESS }
	rocmSmiDeviceHandle = func(int) (rocm_smi.DeviceHandle, rocm_smi.RSMI_status) {
		return rocm_smi.DeviceHandle{}, rocm_smi.STATUS_SUCCESS
	}
	rocmSmiPciInfo = func(rocm_smi.DeviceHandle) (rocm_smi.Pci_info, rocm_smi.RSMI_status) {
		return rocm_smi.Pci_info{}, rocm_smi.STATUS_SUCCESS
	}
	rocmSmiSerial = func(rocm_smi.DeviceHandle) (string, rocm_smi.RSMI_status) { return "", rocm_smi.STATUS_NOT_SUPPORTED }
	rocmSmiMetrics = func(rocm_smi.DeviceHandle) (rocm_smi.RSMI_gpu_metrics, rocm_smi.RSMI_status) {
		*reads++
		return rocm_smi.RSMI_gpu_metrics{}, rocm_smi.STATUS_SUCCESS
	}
	rocmSmiStatusString = func(rocm_smi.RSMI_status) string { return "fake ROCm error" }
	t.Cleanup(func() {
		rocmSmiInit = init
		rocmSmiShutdown = shutdown
		rocmSmiDeviceCount = count
		rocmSmiDeviceHandle = handle
		rocmSmiPciInfo = pci
		rocmSmiSerial = serial
		rocmSmiMetrics = metrics
		rocmSmiStatusString = status
	})
	return stops, reads
}
func TestRocmInitRollbackControls(t *testing.T) {
	for _, scenario := range []string{"init-error", "init-panic", "count-error", "count-panic", "handle-error", "pci-error", "cleanup-error", "cleanup-panic", "success"} {
		t.Run(scenario, func(t *testing.T) {
			stops, reads := mockRocmLifecycle(t)
			switch scenario {
			case "init-error":
				rocmSmiInit = func() rocm_smi.RSMI_status { return rocm_smi.STATUS_NOT_SUPPORTED }
			case "init-panic":
				rocmSmiInit = func() rocm_smi.RSMI_status { panic("mock init panic") }
			case "count-error", "cleanup-error", "cleanup-panic":
				rocmSmiDeviceCount = func() (int, rocm_smi.RSMI_status) { return 0, rocm_smi.STATUS_NOT_SUPPORTED }
			case "count-panic":
				rocmSmiDeviceCount = func() (int, rocm_smi.RSMI_status) { panic("mock count panic") }
			case "handle-error":
				rocmSmiDeviceHandle = func(int) (rocm_smi.DeviceHandle, rocm_smi.RSMI_status) {
					return rocm_smi.DeviceHandle{}, rocm_smi.STATUS_NOT_SUPPORTED
				}
			case "pci-error":
				rocmSmiPciInfo = func(rocm_smi.DeviceHandle) (rocm_smi.Pci_info, rocm_smi.RSMI_status) {
					return rocm_smi.Pci_info{}, rocm_smi.STATUS_NOT_SUPPORTED
				}
			}
			if scenario == "cleanup-error" {
				rocmSmiShutdown = func() rocm_smi.RSMI_status { *stops++; return rocm_smi.STATUS_NOT_SUPPORTED }
			}
			if scenario == "cleanup-panic" {
				rocmSmiShutdown = func() rocm_smi.RSMI_status { *stops++; panic("mock cleanup panic") }
			}
			collector := RocmSmiCollector{}
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Init escaped panic instead of error: %v", r)
				}
			}()
			err := collector.Init([]byte(`{"add_serial_meta":true}`))
			if scenario == "success" {
				if err != nil || !collector.Initialized() || *stops != 0 {
					t.Fatalf("success err=%v initialized=%v shutdown=%d", err, collector.Initialized(), *stops)
				}
				out := make(chan lp.CCMessage, 64)
				collector.Read(time.Second, out)
				if *reads != 1 {
					t.Errorf("successful Read calls %d", *reads)
				}
				collector.Close()
				collector.Close()
				collector.Read(time.Second, out)
				if *stops != 1 || *reads != 1 || collector.Initialized() {
					t.Errorf("Close/repeatedClose/Read shutdown=%d read=%d init=%v", *stops, *reads, collector.Initialized())
				}
			} else {
				want := 1
				if scenario == "init-error" || scenario == "init-panic" {
					want = 0
				}
				if err == nil || collector.Initialized() || *stops != want {
					t.Errorf("failed init err=%v init=%v shutdown=%d want%d", err, collector.Initialized(), *stops, want)
				}
				if (scenario == "cleanup-error" || scenario == "cleanup-panic") && err != nil && !strings.Contains(err.Error(), "cleanup failed") {
					t.Errorf("cleanup error lost: %v", err)
				}
				collector.Close()
				if *stops != want {
					t.Errorf("failed init Close released unowned reference %d", *stops)
				}
			}
		})
	}
}
func TestRocmNeverInitializedDoesNotReadOrShutdown(t *testing.T) {
	stops, reads := mockRocmLifecycle(t)
	collector := RocmSmiCollector{devices: []RocmSmiCollectorDevice{{}}}
	collector.Read(time.Second, make(chan lp.CCMessage, 64))
	collector.Close()
	if *stops != 0 || *reads != 0 {
		t.Errorf("uninitialized Read/Close hardware calls reads=%d shutdown=%d", *reads, *stops)
	}
}
