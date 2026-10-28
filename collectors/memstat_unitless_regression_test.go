package collectors

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemstatUnitlessHugePageFields(t *testing.T) {
	for _, prefix := range []string{"", "Node 0 "} {
		path := filepath.Join(t.TempDir(), "meminfo")
		data := prefix + "HugePages_Total: 8\n" + prefix + "HugePages_Free: 2\n" + prefix + "Hugepagesize: 2048 kB\n"
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		stats := getStats(path)
		if value, ok := stats["HugePages_Total"]; !ok || value.value != 8 || value.unit != "" {
			t.Fatalf("unitless hugepage count lost: %v", stats)
		}
		if stats["Hugepagesize"].value != 2048 || stats["Hugepagesize"].unit != "kB" {
			t.Fatal("unit-bearing field changed")
		}
	}
}
