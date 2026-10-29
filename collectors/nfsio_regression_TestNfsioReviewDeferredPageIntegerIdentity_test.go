package collectors

import (
	"testing"
)

func TestNfsioReviewDeferredPageIntegerIdentity(t *testing.T) {
	text := nfsioReviewHeader("srv:/one", "/a") + "\tbytes:\t1 2 3 4 5 6 9223372036854775807 0\n"
	c := nfsioReviewInit(t, text, `{}`)
	out := nfsioReviewRead(c)
	nfsioReviewValue(t, out, "nfsio_pageread:/a", int64(9223372036854775807))
	nfsioReviewValue(t, out, "nfsio_pagewrite:/a", 0)
}
