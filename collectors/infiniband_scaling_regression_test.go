package collectors

import (
	"testing"
	"time"
)

func TestIBDriverSpecificDataCounterUnits(t *testing.T) {
	for _, driver := range []string{"mlx5_core", "hfi1", "missing"} {
		t.Run(driver, func(t *testing.T) {
			c, f := newIBDeliveryFixture(driver)
			if err := c.Init([]byte(`{"send_derived_values":true,"send_total_values":true}`)); err != nil {
				t.Fatal(err)
			}
			scale := uint64(4)
			if driver == "hfi1" {
				scale = 8
			}
			first := ibDeliveryRead(t, c)
			if len(first["ib_recv"]) != 1 || first["ib_recv"][0] != 10*scale {
				t.Fatalf("first receive=%v, want %d", first["ib_recv"], 10*scale)
			}
			if first["ib_recv_pkts"][0] != uint64(30) {
				t.Fatalf("packet unit changed: %v", first["ib_recv_pkts"])
			}
			f.now = f.now.Add(2 * time.Second)
			f.values["port_rcv_data"] = "14"
			f.values["port_xmit_data"] = "26"
			second := ibDeliveryRead(t, c)
			if len(second["ib_recv_bw"]) != 1 || second["ib_recv_bw"][0] != float64(2*scale) {
				t.Fatalf("receive rate=%v", second["ib_recv_bw"])
			}
			if second["ib_total"][0] != uint64(40)*scale || second["ib_total_bw"][0] != float64(5*scale) {
				t.Fatalf("totals=%v", second)
			}
		})
	}
}
