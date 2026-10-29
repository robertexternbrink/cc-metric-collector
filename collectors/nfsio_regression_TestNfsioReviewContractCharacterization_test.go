package collectors

import (
	"os"
	"testing"
)

func TestNfsioReviewContractCharacterization(t *testing.T) {
	text := nfsioReviewHeader("srv:/same", "/a") + nfsioReviewBytes("100") + nfsioReviewHeader("srv:/same", "/b") + nfsioReviewBytes("200")
	c := nfsioReviewInit(t, text, `{"use_server_as_stype":true}`)
	if c.data["srv:/same"]["nread"] != 200 || len(c.data) != 1 {
		t.Error("deferred export behavior changed")
	}
	t.Logf("server-keyed export contract unchanged:%v", c.data)
	c = nfsioReviewInit(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("100"), `{}`)
	c.Close()
	if len(nfsioReviewRead(c)) != 8 {
		t.Error("deferred Read-after-Close contract changed")
	}
	if err := os.Remove(nfsioReviewFile); err != nil {
		t.Fatal(err)
	}
	nfsioReviewRead(c)
	if len(c.data) != 0 {
		t.Error("deferred unavailable policy changed")
	}
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("130"))
	if len(nfsioReviewRead(c)) != 0 {
		t.Error("deferred first-mount absolute policy changed")
	}
}
