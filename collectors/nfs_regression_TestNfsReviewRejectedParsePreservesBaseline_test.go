package collectors

import (
	"testing"
)

func TestNfsReviewRejectedParsePreservesBaseline(t *testing.T) {
	c := nfsReviewInit(t, "v3", "nfs v3 client read: 100\nnfs v3 client write: 100\n", nil)
	nfsReviewWrite(t, "nfs v3 client read: 110\nnfs v3 client write: invalid\n")
	if len(nfsReviewRead(c)) != 0 {
		t.Fatal("parseerror emitted messages")
	}
	nfsReviewWrite(t, "nfs v3 client read: 120\nnfs v3 client write: 130\n")
	out := nfsReviewRead(c)
	nfsReviewValue(t, out, "nfs3_read", 20)
	nfsReviewValue(t, out, "nfs3_write", 30)
}
