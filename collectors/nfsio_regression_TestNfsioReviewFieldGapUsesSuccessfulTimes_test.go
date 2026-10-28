package collectors

import (
	"testing"
)

func TestNfsioReviewFieldGapUsesSuccessfulTimes(t *testing.T) {
	c := nfsioReviewInit(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("100"), `{"send_derived_values":true}`)
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("110"))
	first := nfsioReviewRead(c)["nfsio_nread:/a"]
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("invalid"))
	if nfsioReviewRead(c)["nfsio_nread_bw:/a"] != nil {
		t.Error("invalid field emitted")
	}
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("130"))
	nfsioReviewRate(t, nfsioReviewRead(c)["nfsio_nread_bw:/a"], 20, first.Time())
}
