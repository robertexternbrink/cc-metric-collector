package collectors

import (
	"encoding/json"
	"testing"
)

func TestNfsioReviewControls(t *testing.T) {
	text := nfsioReviewHeader("srv:/one", "/a") + nfsioReviewBytes("100") + nfsioReviewHeader("srv:/two", "/b") + nfsioReviewBytes("200")
	c := nfsioReviewInit(t, text, `{"exclude_filesystem":["/b"],"exclude_metrics":["nwrite"]}`)
	if !c.Initialized() || !c.Parallel() || c.key != "mntpoint" || !c.config.SendAbsoluteValues || c.config.SendDerivedValues {
		t.Fatal("defaults/lifecycle differs")
	}
	out := nfsioReviewRead(c)
	if len(out) != 7 {
		t.Fatalf("absolute filtered messages%d", len(out))
	}
	m := nfsioReviewValue(t, out, "nfsio_nread:/a", 100)
	if m != nil {
		if unit, _ := m.GetMeta("unit"); unit != "bytes" {
			t.Error("normal byteunit differs")
		}
		if typ, _ := m.GetTag("stype"); typ != "filesystem" {
			t.Error("stype differs")
		}
	}
	if out["nfsio_nwrite:/a"] != nil || out["nfsio_nread:/b"] != nil {
		t.Error("excluded value emitted")
	}
	if err := new(NfsIOStatCollector).Init(json.RawMessage(`{"unknown":true}`)); err == nil {
		t.Error("unknown config accepted")
	}
	c = nfsioReviewInit(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("100"), `{"send_derived_values":true}`)
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("110"))
	first := nfsioReviewRead(c)["nfsio_nread:/a"]
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("120"))
	second := nfsioReviewRead(c)
	nfsioReviewValue(t, second, "nfsio_nread:/a", 120)
	nfsioReviewRate(t, second["nfsio_nread_bw:/a"], 10, first.Time())
	c = nfsioReviewInit(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("0"), `{"send_derived_values":true}`)
	nfsioReviewWrite(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewBytes("10"))
	if nfsioReviewRead(c)["nfsio_nread_bw:/a"] == nil {
		t.Error("true zero was treated as unknown")
	}
	// Generic named groups, absent matches, NFSv3 header, and raw metric selection.
	groups := resolve_regex_fields("device srv:/x mounted on /c with fstype nfs statvers=1.0", deviceRegex)
	if groups["mntpoint"] != "/c" || groups["version"] != "" {
		t.Error("named groups differ")
	}
	if len(resolve_regex_fields("none", bytesRegex)) != 0 {
		t.Error("nonmatch differs")
	}
}
