package collectors

import (
	"testing"
)

func TestNfsioReviewExcludedBlockCannotReusePreviousMount(t *testing.T) {
	c := nfsioReviewInit(t, nfsioReviewHeader("srv:/one", "/a")+nfsioReviewHeader("srv:/two", "/b")+nfsioReviewBytes("999"), `{"exclude_filesystem":["/b"]}`)
	if values := c.data["/a"]; values != nil {
		t.Errorf("excluded /b bytes wrongly assigned to /a:%v", values)
	}
}
