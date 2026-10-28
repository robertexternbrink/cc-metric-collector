package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"testing"
	"time"
)

func mockEnergy(t *testing.T, value *uint64, instant *time.Time) *int {
	t.Helper()
	originalMode, originalEnergy, originalNow := nvml.DeviceGetPowerManagementMode, nvml.DeviceGetTotalEnergyConsumption, nvidiaNow
	calls := new(int)
	nvml.DeviceGetPowerManagementMode = func(nvml.Device) (nvml.EnableState, nvml.Return) { return nvml.FEATURE_ENABLED, nvml.SUCCESS }
	nvml.DeviceGetTotalEnergyConsumption = func(nvml.Device) (uint64, nvml.Return) { *calls++; return *value, nvml.SUCCESS }
	nvidiaNow = func() time.Time { return *instant }
	t.Cleanup(func() {
		nvml.DeviceGetPowerManagementMode = originalMode
		nvml.DeviceGetTotalEnergyConsumption = originalEnergy
		nvidiaNow = originalNow
	})
	return calls
}
func nvidiaValues(t *testing.T, out chan lp.CCMessage) map[string]float64 {
	t.Helper()
	result := map[string]float64{}
	for len(out) > 0 {
		m := <-out
		v, _ := m.GetMetricValue()
		switch n := v.(type) {
		case float64:
			result[m.Name()] = n
		case uint64:
			result[m.Name()] = float64(n)
		case int64:
			result[m.Name()] = float64(n)
		default:
			t.Fatalf("unexpected metric %s value %T", m.Name(), v)
		}
	}
	return result
}
func TestNvidiaEnergyIndependentExcludes(t *testing.T) {
	now := time.Unix(1700000000, 0)
	value := uint64(4000)
	calls := mockEnergy(t, &value, &now)
	names := []string{"nv_energy", "nv_energy_abs", "nv_average_power"}
	for _, selected := range names {
		t.Run(selected, func(t *testing.T) {
			excluded := map[string]bool{}
			for _, name := range names {
				excluded[name] = name != selected
			}
			device := NvidiaCollectorDevice{excludeMetrics: excluded, lastEnergyReading: 2000, lastEnergyTimestamp: now.Add(-2 * time.Second)}
			out := make(chan lp.CCMessage, 4)
			readEnergyConsumption(&device, out)
			got := nvidiaValues(t, out)
			expected := map[string]float64{"nv_energy": 2, "nv_energy_abs": 4, "nv_average_power": 1}[selected]
			if len(got) != 1 || got[selected] != expected {
				t.Fatalf("selected %s output %v, want %g", selected, got, expected)
			}
		})
	}
	before := *calls
	device := NvidiaCollectorDevice{excludeMetrics: map[string]bool{"nv_energy": true, "nv_energy_abs": true, "nv_average_power": true}}
	out := make(chan lp.CCMessage, 4)
	readEnergyConsumption(&device, out)
	if len(out) != 0 || *calls != before {
		t.Fatal("all excluded still sampled")
	}
}
