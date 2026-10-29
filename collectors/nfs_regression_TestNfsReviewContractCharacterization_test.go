package collectors

import (
	"testing"
)

func TestNfsReviewContractCharacterization(t *testing.T) {
	c := nfsReviewInit(t, "v3", "nfs v3 server read: 100\nnfs v3 client read: 1000\n", nil)
	nfsReviewWrite(t, "nfs v3 server read: 120\nnfs v3 client read: 1050\n")
	out := nfsReviewRead(c)
	v, _ := out["nfs3_read"].GetMetricValue()
	if v != int64(50) {
		t.Errorf("existing client/server characterization changed:%v", v)
	}
	t.Logf("combined server+client output read=%v; no aggregation policy asserted", v)
	nfsReviewWrite(t, "")
	if len(nfsReviewRead(c)) != 0 {
		t.Fatal("empty output emitted")
	}
	nfsReviewWrite(t, "nfs v3 client read: 1070\n")
	if out := nfsReviewRead(c); len(out) != 0 {
		t.Errorf("existing missing-counter characterization changed:%v", out)
	}
}
