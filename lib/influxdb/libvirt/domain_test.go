package libvirt

import (
	"github.com/rburchell/gobo/lib/influxdb"
	"github.com/stvp/assert"
	"testing"
)

func TestSimple(t *testing.T) {
	testStr := `Domain: 'test'
  state.state=1234
  state.reason=4321`
	domains := ParseIntoStats([]byte(testStr))
	assert.Equal(t, len(domains), 1)

	dom := domains["test"]
	assert.Equal(t, dom.Domain, "test")
	assert.Equal(t, dom.Stats, []*influxdb.Stat{
		influxdb.NewStatWithTagsAndValues("state", "_libvirt",
			map[string]string{"domain": "test"},
			map[string]string{"state": "1234", "reason": "4321"},
		),
	})
}

func TestMulti(t *testing.T) {
	testStr := `Domain: 'test'
  state.state=1234
  state.reason=4321
  
Domain: 'test2'
  state.state=4321
  state.reason=1234`
	domains := ParseIntoStats([]byte(testStr))
	assert.Equal(t, len(domains), 2)

	dom := domains["test"]
	assert.Equal(t, dom.Domain, "test")
	assert.Equal(t, dom.Stats, []*influxdb.Stat{
		influxdb.NewStatWithTagsAndValues("state", "_libvirt",
			map[string]string{"domain": "test"},
			map[string]string{"state": "1234", "reason": "4321"},
		),
	})

	dom = domains["test2"]
	assert.Equal(t, dom.Domain, "test2")
	assert.Equal(t, dom.Stats, []*influxdb.Stat{
		influxdb.NewStatWithTagsAndValues("state", "_libvirt",
			map[string]string{"domain": "test2"},
			map[string]string{"state": "4321", "reason": "1234"},
		),
	})
}
