/*
 * Copyright (C) 2017 Robin Burchell <robin+git@viroteck.net>
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *  - Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *  - Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND ANY
 * EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 * WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 * DISCLAIMED. IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE FOR ANY
 * DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 * (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 * LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 * ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF
 * THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

package influxdb

import (
	"fmt"
	"strconv"
	"strings"
)

// a Stat represents a single measurement about a single statistic for a domain.
// For instance, when measuring CPU use, you would want to create a Stat every
// time you wanted to write a measurement to InfluxDB, and if you wanted to
// measure per core, you would want an instance per measurement per core.
type Stat struct {
	// the type of stat (e.g. vcpu, cpu, etc.)
	table       string
	tableSuffix string
	key         string

	// tags for influx
	tags map[string]string

	// values for influx
	values map[string]string
}

// Create an empty stat representing a given table, and tableSuffix. The table and
// tableSuffix are combined to give the InfluxDB measurement name.
func NewStat(table string, tableSuffix string) *Stat {
	p := Stat{}
	p.table = table
	p.tableSuffix = tableSuffix
	p.tags = make(map[string]string)
	p.values = make(map[string]string)
	return &p
}

// Set the key representing multiple entries for the same table. For instance,
// if you are measuring per-core CPU use, you likely have multiple entries with
// the same table name.
//
// The key is used only as an identifier, and does not get used in any way in
// itself.
func (this *Stat) SetKey(key string) {
	this.key = key
}

// Get the key representing multiple entries for the same table. See SetKey().
func (this *Stat) Key() string {
	return this.key
}

// Get the table name (e.g. "cpu" for CPU measurements). This is used to create
// the Influx measurement name.
func (this *Stat) Table() string {
	return this.table
}

// Append a tag for this stat, e.g. AppendTag("domain", "my.host.here")
func (this *Stat) AppendTag(key string, value string) {
	this.tags[key] = value
}

// Get the tags for this stat.
func (this *Stat) Tags() map[string]string {
	return this.tags
}

// Append a value for this stat, e.g. AppendValue("system_time", "12345").
func (this *Stat) AppendValue(key string, value string) {
	this.values[key] = value
}

// Get the values for this stat.
func (this *Stat) Values() map[string]string {
	return this.values
}

// Turn val into something that influxdb's line protocol understands.
func quoteIfNecessary(val string) string {
	_, err := strconv.ParseUint(val, 10, 64)
	if err == nil {
		// Number
		return val
	}
	_, err = strconv.ParseFloat(val, 64)
	if err == nil {
		// Float
		return val
	}

	return fmt.Sprintf("\"%s\"", val)
}

// Get this stat as a string following InfluxDB's line protocol.
func (this *Stat) InfluxString() string {
	if len(this.values) == 0 {
		return ""
	}

	tstr := this.table + this.tableSuffix

	if len(this.tags) > 0 {
		tstr += ","
		for key, val := range this.tags {
			if key == "time" {
				key = "time_field"
			}
			tstr += key
			tstr += "="
			tstr += strings.Replace(val, " ", "\\ ", -1)
			tstr += ","
		}
		tstr = tstr[:len(tstr)-1]
	}

	vstr := ""
	for key, val := range this.values {
		if key == "time" {
			key = "time_field"
		}
		vstr += key
		vstr += "="
		vstr += quoteIfNecessary(val)
		vstr += ","
	}
	vstr = vstr[:len(vstr)-1]

	return fmt.Sprintf("%s %s\n", tstr, vstr)
}
