package collectors

import (
	"testing"
)

func TestNfsioReviewDeferredDirectIOContract(t *testing.T) {
	c := nfsioReviewInit(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("100"), `{"exclude_metrics":["oread"]}`)
	out := nfsioReviewRead(c)
	if out["nfsio_dread:/a"] == nil || out["nfsio_oread:/a"] != nil {
		t.Error("deferred DirectIO naming/exclusion behavior changed")
	}
}
