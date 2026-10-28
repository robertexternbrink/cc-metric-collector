package collectors

import (
	"testing"
)

func TestNfsReviewDocumentedExclusion(t *testing.T) {
	for _, version := range []string{"v3", "v4"} {
		t.Run(version, func(t *testing.T) {
			prefix := "nfs3"
			if version == "v4" {
				prefix = "nfs4"
			}
			c := nfsReviewInit(t, version, "nfs "+version+" client total: 100\n", []string{prefix + "_total"})
			nfsReviewWrite(t, "nfs "+version+" client total: 110\n")
			if m := nfsReviewRead(c)[prefix+"_total"]; m != nil {
				t.Errorf("documented exclude %s_total still forwarded", prefix)
			}
		})
	}
}
