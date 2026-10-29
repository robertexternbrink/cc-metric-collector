package collectors

import (
	"testing"
)

func TestNfsioReviewCounterReset(t *testing.T) {
	c := nfsioReviewInit(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("100"), `{"send_derived_values":true}`)
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("5"))
	reset := nfsioReviewRead(c)
	absolute := nfsioReviewValue(t, reset, "nfsio_nread:/a", 5)
	if m := reset["nfsio_nread_bw:/a"]; m != nil {
		v, _ := m.GetMetricValue()
		t.Errorf("reset emitted invalid rate%v", v)
	}
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("15"))
	if absolute != nil {
		nfsioReviewRate(t, nfsioReviewRead(c)["nfsio_nread_bw:/a"], 10, absolute.Time())
	}
}
