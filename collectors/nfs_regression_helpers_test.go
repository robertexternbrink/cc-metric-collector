package collectors

import (
	"encoding/json"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var nfsReviewRoot string

func nfsReviewWrite(t *testing.T, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(nfsReviewRoot, "output"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
}

func nfsReviewInit(t *testing.T, version, text string, exclude []string) *nfsCollector {
	t.Helper()
	previous := nfsReviewRoot
	nfsReviewRoot = t.TempDir()
	t.Cleanup(func() { nfsReviewRoot = previous })
	if err := os.MkdirAll(nfsReviewRoot, 0700); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(nfsReviewRoot, "nfsstat-stub")
	// Only an explicitly configured local output stub, never real nfsstat.
	content := "#!/bin/sh\n[ \"$1\" = \"-l\" ] && [ \"$2\" = \"--all\" ] || exit 7\nexec /bin/cat '" + filepath.Join(nfsReviewRoot, "output") + "'\n"
	if err := os.WriteFile(script, []byte(content), 0700); err != nil {
		t.Fatal(err)
	}
	nfsReviewWrite(t, text)
	cfg, err := json.Marshal(map[string]any{"nfsstat": script, "exclude_metrics": exclude})
	if err != nil {
		t.Fatal(err)
	}
	var c *nfsCollector
	if version == "v3" {
		v := new(Nfs3Collector)
		if err := v.Init(cfg); err != nil {
			t.Fatal(err)
		}
		c = &v.nfsCollector
	} else {
		v := new(Nfs4Collector)
		if err := v.Init(cfg); err != nil {
			t.Fatal(err)
		}
		c = &v.nfsCollector
	}
	t.Cleanup(func() { os.Remove(script); os.Remove(filepath.Join(nfsReviewRoot, "output")) })
	return c
}

func nfsReviewRead(c *nfsCollector) map[string]lp.CCMessage {
	ch := make(chan lp.CCMessage, 16)
	c.Read(time.Second, ch)
	out := make(map[string]lp.CCMessage)
	for len(ch) > 0 {
		m := <-ch
		out[m.Name()] = m
	}
	return out
}

func nfsReviewValue(t *testing.T, out map[string]lp.CCMessage, key string, want int64) {
	t.Helper()
	m := out[key]
	if m == nil {
		t.Errorf("missing %s want%d", key, want)
		return
	}
	v, ok := m.GetMetricValue()
	if !ok || v != want {
		t.Errorf("%s=%v,error%v want%d", key, v, ok, want)
	}
}
