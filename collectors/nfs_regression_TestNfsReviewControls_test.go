package collectors

import (
	"encoding/json"
	"os"
	"testing"
)

func TestNfsReviewControls(t *testing.T) {
	for _, version := range []string{"v3", "v4"} {
		t.Run(version, func(t *testing.T) {
			prefix := "nfs3"
			if version == "v4" {
				prefix = "nfs4"
			}
			c := nfsReviewInit(t, version, "nfs "+version+" client read: 100\nnfs "+version+" client write: 20\nnfs v2 client read: 999\n", nil)
			if !c.Initialized() || !c.Parallel() {
				t.Fatal("lifecycle differs")
			}
			nfsReviewWrite(t, "nfs "+version+" client read: 120\nnfs "+version+" client write: 25\n")
			out := nfsReviewRead(c)
			if len(out) != 2 {
				t.Fatalf("messagecount%d", len(out))
			}
			nfsReviewValue(t, out, prefix+"_read", 20)
			nfsReviewValue(t, out, prefix+"_write", 5)
			m := out[prefix+"_read"]
			if tag, _ := m.GetTag("type"); tag != "node" {
				t.Error("type differs")
			}
			if v, _ := m.GetMeta("version"); v != version {
				t.Error("version differs")
			}
			if g, _ := m.GetMeta("group"); g != "NFS" {
				t.Error("group differs")
			}
			c.Close()
			if len(nfsReviewRead(c)) != 0 {
				t.Error("closed collector emitted")
			}
		})
	}
	c := nfsReviewInit(t, "v3", "nfs v3 client read: 100\n", []string{"read"})
	nfsReviewWrite(t, "nfs v3 client read: 110\n")
	if len(nfsReviewRead(c)) != 0 {
		t.Error("raw exclude differs")
	}
	if err := new(Nfs3Collector).Init(json.RawMessage(`{"unknown":true}`)); err == nil {
		t.Error("unknown config accepted")
	}
	// Failed command returns before updateStats mutates known counters.
	c = nfsReviewInit(t, "v3", "nfs v3 client read: 100\n", nil)
	if err := os.WriteFile(c.config.Nfsstats, []byte("#!/bin/sh\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if len(nfsReviewRead(c)) != 0 || c.data["read"].current != 100 {
		t.Error("failed command consumed baseline")
	}
}
