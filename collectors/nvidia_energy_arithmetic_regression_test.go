package collectors

import (
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"math"
	"testing"
	"time"
)

func TestNvidiaEnergyArithmeticControls(t *testing.T) {
	base := time.Unix(1700000000, 0)
	scenarios := []struct {
		name              string
		previous, current uint64
		elapsed           time.Duration
		want              map[string]float64
	}{
		{"fractional-joules", 1000, 1250, 2 * time.Second, map[string]float64{"nv_energy": .25, "nv_energy_abs": 1.25, "nv_average_power": .125}},
		{"fractional-seconds", 1000, 4000, 1500 * time.Millisecond, map[string]float64{"nv_energy": 3, "nv_energy_abs": 4, "nv_average_power": 2}},
		{"subsecond", 1000, 1250, 250 * time.Millisecond, map[string]float64{"nv_energy": .25, "nv_energy_abs": 1.25, "nv_average_power": 1}},
		{"zero-baseline", 0, 250, 250 * time.Millisecond, map[string]float64{"nv_energy": .25, "nv_energy_abs": .25, "nv_average_power": 1}},
		{"counter-reset", 5000, 500, time.Second, map[string]float64{"nv_energy_abs": .5}},
		{"zero-time", 1000, 2000, 0, map[string]float64{"nv_energy": 1, "nv_energy_abs": 2}},
		{"clock-backwards", 1000, 2000, -time.Second, map[string]float64{"nv_energy": 1, "nv_energy_abs": 2}},
	}
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			instant := base.Add(s.elapsed)
			value := s.current
			mockEnergy(t, &value, &instant)
			device := NvidiaCollectorDevice{excludeMetrics: map[string]bool{}, lastEnergyReading: s.previous, lastEnergyTimestamp: base}
			out := make(chan lp.CCMessage, 8)
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("sample panicked: %v", r)
				}
			}()
			readEnergyConsumption(&device, out)
			got := nvidiaValues(t, out)
			if len(got) != len(s.want) {
				t.Fatalf("got %v want %v", got, s.want)
			}
			for name, want := range s.want {
				v, exists := got[name]
				if !exists || math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v-want) > 1e-12 {
					t.Errorf("%s=%g exists=%v want %g", name, v, exists, want)
				}
			}
			if device.lastEnergyReading != value || device.lastEnergyTimestamp != instant {
				t.Fatal("baseline did not use same sample timestamp")
			}
		})
	}
}
func TestNvidiaEnergyFirstZeroAndResetRecovery(t *testing.T) {
	now := time.Unix(1700000000, 0)
	energy := uint64(0)
	mockEnergy(t, &energy, &now)
	device := NvidiaCollectorDevice{excludeMetrics: map[string]bool{}}
	out := make(chan lp.CCMessage, 8)
	readEnergyConsumption(&device, out)
	first := nvidiaValues(t, out)
	if len(first) != 1 || first["nv_energy_abs"] != 0 {
		t.Fatalf("first sample fabricated delta: %v", first)
	}
	now = now.Add(250 * time.Millisecond)
	energy = 250
	readEnergyConsumption(&device, out)
	got := nvidiaValues(t, out)
	if len(got) != 3 || got["nv_energy"] != .25 || got["nv_average_power"] != 1 {
		t.Fatalf("valid zero baseline lost: %v", got)
	}
	now = now.Add(time.Second)
	energy = 100
	readEnergyConsumption(&device, out)
	got = nvidiaValues(t, out)
	if len(got) != 1 || got["nv_energy_abs"] != .1 {
		t.Fatalf("reset underflow: %v", got)
	}
	now = now.Add(time.Second)
	energy = 600
	readEnergyConsumption(&device, out)
	got = nvidiaValues(t, out)
	if got["nv_energy"] != .5 || got["nv_average_power"] != .5 {
		t.Fatalf("reset did not rebaseline: %v", got)
	}
}
