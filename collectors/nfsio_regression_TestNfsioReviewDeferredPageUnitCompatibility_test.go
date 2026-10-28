package collectors

import (
	"testing"
)

func TestNfsioReviewDeferredPageUnitCompatibility(t *testing.T) {
	c := nfsioReviewInit(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("100"), `{}`)
	m := nfsioReviewValue(t, nfsioReviewRead(c), "nfsio_pageread:/a", 4)
	if m != nil {
		unit, _ := m.GetMeta("unit")
		if unit != "bytes" {
			t.Errorf("deferred absolute pageunit unexpectedly changed:%q", unit)
		}
	}
}
