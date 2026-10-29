package collectors

import (
	"fmt"
	lp "github.com/ClusterCockpit/cc-lib/v2/ccMessage"
	"path/filepath"
	"testing"
	"time"
)

type ibDeliveryFixture struct {
	now    time.Time
	driver string
	values map[string]string
}

func newIBDeliveryFixture(driver string) (*InfinibandCollector, *ibDeliveryFixture) {
	f := &ibDeliveryFixture{now: time.Unix(100, 0), driver: driver, values: map[string]string{"port_rcv_data": "10", "port_xmit_data": "20", "port_rcv_packets": "30", "port_xmit_packets": "40"}}
	c := &InfinibandCollector{sources: infinibandSources{
		glob: func(string) ([]string, error) { return []string{"/sys/class/infiniband/fake_0/ports/1"}, nil },
		readFile: func(path string) ([]byte, error) {
			if filepath.Base(path) == "lid" {
				return []byte("0x1"), nil
			}
			v, ok := f.values[filepath.Base(path)]
			if !ok {
				return nil, fmt.Errorf("unavailable fixture counter")
			}
			return []byte(v), nil
		},
		access: func(string, uint32) error { return nil }, now: func() time.Time { return f.now }, evalSymlinks: func(string) (string, error) {
			if f.driver == "missing" {
				return "", fmt.Errorf("no driver symlink")
			}
			return "/drivers/" + f.driver, nil
		},
	}}
	return c, f
}
func ibDeliveryRead(t *testing.T, c *InfinibandCollector) map[string][]any {
	t.Helper()
	ch := make(chan lp.CCMessage, 64)
	c.Read(time.Second, ch)
	out := map[string][]any{}
	for len(ch) > 0 {
		m := <-ch
		v, ok := m.GetMetricValue()
		if !ok {
			t.Fatal("missing metric value")
		}
		out[m.Name()] = append(out[m.Name()], v)
	}
	return out
}
