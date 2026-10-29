package collectors

import (
	"testing"
)

func TestNfsReviewReset(t *testing.T) {
	c := nfsReviewInit(t, "v3", "nfs v3 client read: 100\n", nil)
	nfsReviewWrite(t, "nfs v3 client read: 5\n")
	if m := nfsReviewRead(c)["nfs3_read"]; m != nil {
		v, _ := m.GetMetricValue()
		t.Errorf("reset emitted%v instead of establishing new baseline", v)
	}
	nfsReviewWrite(t, "nfs v3 client read: 15\n")
	nfsReviewValue(t, nfsReviewRead(c), "nfs3_read", 10)
}
