package collectors

import "testing"

func TestCPUInfoFrequenciesFollowProcessorIDs(t *testing.T) {
	c, path := cpuInfoFixture(t)
	for _, data := range []string{cpuInfoRecord(2, 2200), cpuInfoRecord(2, 2200) + cpuInfoRecord(0, 1100)} {
		out := cpuInfoReadFixture(t, c, path, data)
		if out["2"] != 2200 {
			t.Errorf("CPU2 frequency assigned incorrectly: %v", out)
		}
	}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("new processor panicked: %v", r)
		}
	}()
	out := cpuInfoReadFixture(t, c, path, cpuInfoRecord(0, 1100)+cpuInfoRecord(2, 2200)+cpuInfoRecord(4, 4400))
	if len(out) != 2 || out["0"] != 1100 || out["2"] != 2200 {
		t.Errorf("known CPUs changed after new CPU: %v", out)
	}
}
