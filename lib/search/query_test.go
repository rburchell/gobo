package search

import (
	"github.com/stvp/assert"
	"testing"
)

type resultSet []ResultIdentifier

func queryNoError(t *testing.T, q string) resultSet {
	out, err := CreateQuery(q)
	if err != nil {
		t.Errorf("Parse of %s failed: %s", q, err)
	}
	rchan, err := Evaluate(out, &testIndex{})
	if err != nil {
		t.Errorf("Evaluate of %s failed: %s", q, err)
	}

	var rs resultSet = make(resultSet, 0)
	for r := range rchan {
		rs = append(rs, r)
	}

	return rs
}

type testIndex struct {
	filteredItems map[ResultIdentifier]ResultIdentifier
}

func (this *testIndex) shouldFilter(id ResultIdentifier) bool {
	if this.filteredItems == nil {
		return false
	}

	_, ok := this.filteredItems[id]
	if !ok {
		return true // filter this
	}
	return false
}

func (this *testIndex) sendIfUnfiltered(id ResultIdentifier, ch chan ResultIdentifier) {
	if !this.shouldFilter(id) {
		ch <- id
	}
}

func (this *testIndex) QueryAll() chan ResultIdentifier {
	ch := make(chan ResultIdentifier)
	go func() {
		this.sendIfUnfiltered(0, ch)
		this.sendIfUnfiltered(1, ch)
		this.sendIfUnfiltered(2, ch)
		this.sendIfUnfiltered(3, ch)
		this.sendIfUnfiltered(4, ch)
		close(ch)
	}()
	return ch
}

func (this *testIndex) QueryTagExact(tag string) chan ResultIdentifier {
	ch := make(chan ResultIdentifier)
	go func() {
		switch {
		case tag == "0":
			this.sendIfUnfiltered(0, ch)
		case tag == "1":
			this.sendIfUnfiltered(1, ch)
		case tag == "2":
			this.sendIfUnfiltered(2, ch)
		case tag == "3":
			this.sendIfUnfiltered(3, ch)
		case tag == "4":
			this.sendIfUnfiltered(4, ch)
		case tag == "undertwo":
			this.sendIfUnfiltered(0, ch)
			this.sendIfUnfiltered(1, ch)
		case tag == "abovetwo":
			this.sendIfUnfiltered(3, ch)
			this.sendIfUnfiltered(4, ch)
		case tag == "all":
			this.sendIfUnfiltered(0, ch)
			this.sendIfUnfiltered(1, ch)
			this.sendIfUnfiltered(2, ch)
			this.sendIfUnfiltered(3, ch)
			this.sendIfUnfiltered(4, ch)
		}
		close(ch)
	}()
	return ch
}

func (this *testIndex) QueryTagFuzzy(tag string) chan ResultIdentifier {
	return this.QueryTagExact(tag)
}

func (this *testIndex) QueryTypedTags(tagType string) chan TypedResult {
	return nil
}

func (this *testIndex) CostTagExact(tag string) int64 { return 0 }

func (this *testIndex) CostTagFuzzy(tag string) int64 { return 0 }

func (this *testIndex) CostTypedTags(tagType string) int64 { return 0 }

func (this *testIndex) CostAll() int64 { return 0 }

func (this *testIndex) CreateFilteredIndex(filteredResults map[ResultIdentifier]ResultIdentifier) Index {
	return &testIndex{
		filteredItems: filteredResults,
	}
}

func TestQueryExact(t *testing.T) {
	type parserTest struct {
		q      string
		result resultSet
	}

	tests := []parserTest{
		parserTest{
			q:      "^all$",
			result: resultSet{0, 1, 2, 3, 4},
		},
		parserTest{
			q:      "^0$",
			result: resultSet{0},
		},
		parserTest{
			q:      "^1$",
			result: resultSet{1},
		},
		parserTest{
			q:      "^2$",
			result: resultSet{2},
		},
		parserTest{
			q:      "^3$",
			result: resultSet{3},
		},
		parserTest{
			q:      "^4$",
			result: resultSet{4},
		},
		parserTest{
			q:      "^nonexist$",
			result: resultSet{},
		},
		parserTest{
			q:      "^undertwo$",
			result: resultSet{0, 1},
		},
		parserTest{
			q:      "^abovetwo$",
			result: resultSet{3, 4},
		},
	}

	for _, test := range tests {
		v := queryNoError(t, test.q)
		assert.Equal(t, v, test.result)
	}
}

func TestQueryFuzzy(t *testing.T) {
	type parserTest struct {
		q      string
		result resultSet
	}

	tests := []parserTest{
		parserTest{
			q:      "all",
			result: resultSet{0, 1, 2, 3, 4},
		},
		parserTest{
			q:      "0",
			result: resultSet{0},
		},
		parserTest{
			q:      "1",
			result: resultSet{1},
		},
		parserTest{
			q:      "2",
			result: resultSet{2},
		},
		parserTest{
			q:      "3",
			result: resultSet{3},
		},
		parserTest{
			q:      "4",
			result: resultSet{4},
		},
		parserTest{
			q:      "nonexist",
			result: resultSet{},
		},
		parserTest{
			q:      "undertwo",
			result: resultSet{0, 1},
		},
		parserTest{
			q:      "abovetwo",
			result: resultSet{3, 4},
		},
	}

	for _, test := range tests {
		v := queryNoError(t, test.q)
		assert.Equal(t, v, test.result)
	}
}

func TestQueryNot(t *testing.T) {
	type parserTest struct {
		q      string
		result resultSet
	}

	tests := []parserTest{
		parserTest{
			q:      "undertwo && !0",
			result: resultSet{1},
		},
		parserTest{
			q:      "undertwo && !1",
			result: resultSet{0},
		},
		parserTest{
			q:      "all && !undertwo",
			result: resultSet{2, 3, 4},
		},
		parserTest{
			q:      "all && !abovetwo",
			result: resultSet{0, 1, 2},
		},
		parserTest{
			q:      "all && !(2 || 3)",
			result: resultSet{0, 1, 4},
		},
	}

	for _, test := range tests {
		v := queryNoError(t, test.q)
		assert.Equal(t, v, test.result)
	}
}
