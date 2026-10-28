package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"testing"
)

func TestNvidiaRemappedRowsKeepIndependentNames(t *testing.T) {
	original := nvml.DeviceGetRemappedRows
	t.Cleanup(func() { nvml.DeviceGetRemappedRows = original })
	nvml.DeviceGetRemappedRows = func(nvml.Device) (int, int, bool, bool, nvml.Return) { return 3, 7, true, false, nvml.SUCCESS }
	for _, uncorrectedOnly := range []bool{false, true} {
		device := NvidiaCollectorDevice{excludeMetrics: map[string]bool{}}
		if uncorrectedOnly {
			device.excludeMetrics = map[string]bool{"nv_remapped_rows_corrected": true, "nv_remapped_rows_pending": true, "nv_remapped_rows_failure": true}
		}
		out := make(chan lp.CCMessage, 4)
		readRemappedRows(&device, out)
		got := nvidiaValues(t, out)
		if got["nv_remapped_rows_uncorrected"] != 7 {
			t.Errorf("uncorrected metric lost/renamed: %v", got)
		}
		if uncorrectedOnly {
			if len(got) != 1 {
				t.Errorf("excluded metric emitted: %v", got)
			}
		} else if got["nv_remapped_rows_corrected"] != 3 || len(got) != 4 {
			t.Errorf("corrected overwritten: %v", got)
		}
	}
}
