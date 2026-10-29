package collectors

import (
	"encoding/json"
	"fmt"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var nfsioReviewFile string

func nfsioReviewBytes(nread string) string {
	return "\tbytes:\t" + strings.Join([]string{nread, "20", "30", "40", "50", "60", "4", "5"}, " ") + "\n"
}
func nfsioReviewHeader(server, mount string) string {
	return fmt.Sprintf("device %s mounted on %s with fstype nfs4 statvers=1.1\n", server, mount)
}
func nfsioReviewWrite(t *testing.T, text string) {
	t.Helper()
	if nfsioReviewFile == "" {
		nfsioReviewFile = filepath.Join(t.TempDir(), "mountstats")
		t.Cleanup(func() { nfsioReviewFile = "" })
	}
	if err := os.WriteFile(nfsioReviewFile, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(nfsioReviewFile); err != nil && !os.IsNotExist(err) {
			t.Error(err)
		}
	})
}
func nfsioReviewInit(t *testing.T, text, cfg string) *NfsIOStatCollector {
	t.Helper()
	nfsioReviewWrite(t, text)
	c := &NfsIOStatCollector{mountstatsFile: nfsioReviewFile}
	if err := c.Init(json.RawMessage(cfg)); err != nil {
		t.Fatal(err)
	}
	return c
}
func nfsioReviewRead(c *NfsIOStatCollector) map[string]lp.CCMessage {
	ch := make(chan lp.CCMessage, 64)
	c.Read(time.Second, ch)
	out := make(map[string]lp.CCMessage)
	for len(ch) > 0 {
		m := <-ch
		id, _ := m.GetTag("stype-id")
		out[m.Name()+":"+id] = m
	}
	return out
}
func nfsioReviewValue(t *testing.T, out map[string]lp.CCMessage, key string, want int64) lp.CCMessage {
	t.Helper()
	m := out[key]
	if m == nil {
		t.Errorf("missing%s want%d", key, want)
		return nil
	}
	v, ok := m.GetMetricValue()
	if !ok || v != want {
		t.Errorf("%s=%v success%v want%d", key, v, ok, want)
	}
	return m
}
func nfsioReviewRate(t *testing.T, m lp.CCMessage, delta float64, last time.Time) {
	t.Helper()
	if m == nil {
		t.Error("missing expected rate")
		return
	}
	v, ok := m.GetMetricValue()
	rate, isFloat := v.(float64)
	if !ok || !isFloat {
		t.Errorf("rate type%T", v)
		return
	}
	dt := m.Time().Sub(last).Seconds()
	if dt <= 0 {
		t.Fatalf("nonpositive observed interval%v", dt)
	}
	want := delta / dt
	if math.IsNaN(rate) || math.IsInf(rate, 0) || math.Abs(rate-want) > 1e-12+math.Abs(want)*1e-8 {
		t.Errorf("rate=%v want%v over observed dt%v", rate, want, dt)
	}
}
