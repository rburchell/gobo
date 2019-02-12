package search

import (
	"github.com/stvp/assert"
	"testing"
)

func parseNoError(t *testing.T, q string) queryToken {
	out, err := CreateQuery(q)
	if err != nil {
		t.Errorf("Parse of %s failed: %s", q, err)
	}
	_, err = Evaluate(out, &emptyTestIndex{})
	if err != nil {
		t.Errorf("Evaluate of %s failed: %s", q, err)
	}
	return out.queryRoot
}

type emptyTestIndex struct {
}

func (this *emptyTestIndex) QueryAll() []ResultIdentifier { return nil }

func (this *emptyTestIndex) QueryTagExact(tag string) []ResultIdentifier { return nil }

func (this *emptyTestIndex) QueryTagFuzzy(tag string) []ResultIdentifier { return nil }

func (this *emptyTestIndex) QueryTypedTags(tagType string) []TypedResult { return nil }

func (this *emptyTestIndex) CostTagExact(tag string) int64 { return 0 }

func (this *emptyTestIndex) CostTagFuzzy(tag string) int64 { return 0 }

func (this *emptyTestIndex) CostTypedTags(tagType string) int64 { return 0 }

func (this *emptyTestIndex) CostAll() int64 { return 0 }

func (this *emptyTestIndex) CreateFilteredIndex(filteredResults map[ResultIdentifier]ResultIdentifier) Index {
	return &emptyTestIndex{}
}

func TestParseSimple(t *testing.T) {
	type parserTest struct {
		q    string
		root queryToken
	}

	tests := []parserTest{
		parserTest{
			q:    "a",
			root: tagQueryToken{tag: "a"},
		},
		parserTest{
			q:    "\"a\"",
			root: tagQueryToken{tag: "a"},
		},
		parserTest{
			q:    "(a)",
			root: tagQueryToken{tag: "a"},
		},
		parserTest{
			q:    "!a",
			root: notToken{right: tagQueryToken{tag: "a"}},
		},
		parserTest{
			q:    "a&&b",
			root: andQueryToken{left: tagQueryToken{tag: "a"}, right: tagQueryToken{tag: "b"}},
		},
		parserTest{
			q:    "a||b",
			root: orQueryToken{left: tagQueryToken{tag: "a"}, right: tagQueryToken{tag: "b"}},
		},
		parserTest{
			q:    "a>5",
			root: greaterThanToken{left: tagQueryToken{tag: "a"}, right: tagQueryToken{tag: "5"}},
		},
		parserTest{
			q:    "a>=5",
			root: greaterThanEqualToken{left: tagQueryToken{tag: "a"}, right: tagQueryToken{tag: "5"}},
		},
		parserTest{
			q:    "a<5",
			root: lessThanToken{left: tagQueryToken{tag: "a"}, right: tagQueryToken{tag: "5"}},
		},
		parserTest{
			q:    "a<=5",
			root: lessThanEqualToken{left: tagQueryToken{tag: "a"}, right: tagQueryToken{tag: "5"}},
		},
		parserTest{
			q:    "a==5",
			root: equalToToken{left: tagQueryToken{tag: "a"}, right: tagQueryToken{tag: "5"}},
		},
		parserTest{
			q:    "a:5",
			root: virtualToken{printable: "a:5", realToken: equalsQueryToken{equals: "a:5"}},
		},
	}

	for _, test := range tests {
		v := parseNoError(t, test.q)
		assert.Equal(t, v, test.root)
	}
}

func TestParseExactTag(t *testing.T) {
	type parserTest struct {
		q    string
		root queryToken
	}

	tests := []parserTest{
		parserTest{
			q:    "^ab$",
			root: equalsQueryToken{equals: "ab"},
		},
		parserTest{
			q:    "(^a$ || ^b$)",
			root: orQueryToken{left: equalsQueryToken{equals: "a"}, right: equalsQueryToken{equals: "b"}},
		},
	}

	for _, test := range tests {
		v := parseNoError(t, test.q)
		assert.Equal(t, v, test.root)
	}
}
