package collectors

import (
	"testing"
)

func TestNfsioReviewUnknownFieldBaseline(t *testing.T) {
	c := nfsioReviewInit(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("invalid"), `{"send_derived_values":true}`)
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("150"))
	first := nfsioReviewRead(c)
	if m := first["nfsio_nread_bw:/a"]; m != nil {
		v, _ := m.GetMetricValue()
		t.Errorf("unknown initial field emitted boot-relative rate%v", v)
	}
	absolute := nfsioReviewValue(t, first, "nfsio_nread:/a", 150)
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("160"))
	second := nfsioReviewRead(c)
	if absolute != nil {
		nfsioReviewRate(t, second["nfsio_nread_bw:/a"], 10, absolute.Time())
	}
}
