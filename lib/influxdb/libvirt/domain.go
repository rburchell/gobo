package libvirt

import (
	"bytes"
	"fmt"
	"github.com/rburchell/gobo/lib/influxdb"
	"regexp"
	"strings"
)

var domRegex = regexp.MustCompile(`^Domain: '(.+)'`)
var kvRegex = regexp.MustCompile(`\b(.+)=(.+)`)

func checkErr(err error, msg string) {
	if err != nil {
		fmt.Printf("error: %s: %s", msg, err.Error())
		panic("boom")
	}
}

// representing all the stats for a given domain (VM).
type Domain struct {
	Stats []*influxdb.Stat

	// The libvirt domain name
	Domain string
}

func (this *Domain) newStat(table string) *influxdb.Stat {
	p := influxdb.NewStat(table, "_libvirt")
	p.AppendTag("domain", this.Domain)
	this.Stats = append(this.Stats, p)
	return p
}

func ParseIntoStats(buf []byte) map[string]*Domain {
	m := make(map[string]*Domain)
	d := &Domain{}

	var p *influxdb.Stat

	lines := bytes.Split(buf, []byte{'\n'})
	for _, line := range lines {
		dparts := domRegex.FindStringSubmatch(string(line))
		if len(dparts) > 0 {
			if len(d.Domain) > 0 {
				d = &Domain{}
				p = nil
			}
			d.Domain = dparts[1]
			m[d.Domain] = d
			continue
		}

		// turn: foo.bar.0.blah=value into a key/value pair.
		kvs := kvRegex.FindStringSubmatch(string(line))
		if len(kvs) == 0 {
			continue
		}

		parts := strings.Split(kvs[1], ".")

		complexTypes := map[string]map[string]bool{
			"block": {
				"name": true,
				"path": true,
			},
			"net": {
				"name": true,
			},
			"vcpu": {},
		}

		if p == nil || p.Table() != parts[0] {
			p = d.newStat(parts[0])
		}
		startIndex := 1

		if specials, ok := complexTypes[p.Table()]; ok {
			potentialKey := parts[startIndex]
			startIndex += 1

			if startIndex >= len(parts) {
				continue
			}

			if p.Key() != potentialKey {
				p = d.newStat(parts[0])
			}
			p.SetKey(potentialKey)

			if len(specials) == 0 {
				p.AppendTag("id", potentialKey)
			}

		}

		if _, ok := complexTypes[p.Table()][parts[startIndex]]; ok {
			p.AppendTag(strings.Join(parts[startIndex:], "."), kvs[2])
		} else {
			p.AppendValue(strings.Join(parts[startIndex:], "."), kvs[2])
		}
	}

	return m
}
