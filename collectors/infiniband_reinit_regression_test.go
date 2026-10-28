package collectors

import "testing"

func TestIBReinitializationUsesFreshPortsAndDefaults(t *testing.T) {
	c, _ := newIBDeliveryFixture("mlx5_core")
	if err := c.Init([]byte(`{"send_total_values":true}`)); err != nil {
		t.Fatal(err)
	}
	c.Close()
	if err := c.Init(nil); err != nil {
		t.Fatal(err)
	}
	out := ibDeliveryRead(t, c)
	if len(out["ib_recv"]) != 1 {
		t.Errorf("port duplicated after reinit: %v", out["ib_recv"])
	}
	if c.config.SendTotalValues || len(out["ib_total"]) != 0 {
		t.Errorf("old total configuration retained")
	}
	c.Close()
	if err := c.Init([]byte(`{"send_total_values":true,"unknown":true}`)); err == nil {
		t.Fatal("invalid configuration accepted")
	}
	if err := c.Init(nil); err != nil {
		t.Fatal(err)
	}
	if c.config.SendTotalValues || len(c.info) != 1 {
		t.Fatalf("failed-init retry retained state: total=%v ports=%d", c.config.SendTotalValues, len(c.info))
	}
}
