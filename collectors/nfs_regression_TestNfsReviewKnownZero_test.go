package collectors

import (
	"testing"
)

func TestNfsReviewKnownZero(t *testing.T) {
	c := nfsReviewInit(t, "v3", "nfs v3 client read: 0\n", nil)
	nfsReviewWrite(t, "nfs v3 client read: 10\n")
	nfsReviewValue(t, nfsReviewRead(c), "nfs3_read", 10)
}
