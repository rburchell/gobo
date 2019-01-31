package search

import (
	"errors"
	"github.com/stvp/assert"
	"testing"
)

func tokenizeNoError(t *testing.T, in string) []string {
	out, err := tokenize(in)
	if err != nil {
		t.Errorf("Tokenize of %s failed: %s", in, err)
	}
	return out
}

func TestTokenizeBasic(t *testing.T) {
	assert.Equal(t, tokenizeNoError(t, " "), []string{})
	assert.Equal(t, tokenizeNoError(t, ""), []string{})
	assert.Equal(t, tokenizeNoError(t, "a"), []string{"a"})
	assert.Equal(t, tokenizeNoError(t, "!"), []string{"!"})
	assert.Equal(t, tokenizeNoError(t, "="), []string{"="})
	assert.Equal(t, tokenizeNoError(t, ">"), []string{">"})
	assert.Equal(t, tokenizeNoError(t, "<"), []string{"<"})
	assert.Equal(t, tokenizeNoError(t, "("), []string{"("})
	assert.Equal(t, tokenizeNoError(t, ")"), []string{")"})
	assert.Equal(t, tokenizeNoError(t, "&"), []string{"&"})
	assert.Equal(t, tokenizeNoError(t, "|"), []string{"|"})
	assert.Equal(t, tokenizeNoError(t, ":"), []string{":"})
	assert.Equal(t, tokenizeNoError(t, "^"), []string{"^"})
	assert.Equal(t, tokenizeNoError(t, "$"), []string{"$"})
}

func TestTokenizeCompound(t *testing.T) {
	assert.Equal(t, tokenizeNoError(t, "abcd"), []string{"abcd"})
}

func TestStringLiteral(t *testing.T) {
	assert.Equal(t, tokenizeNoError(t, "\"abcd\""), []string{"abcd"})
	assert.Equal(t, tokenizeNoError(t, "\"ab  cd\""), []string{"ab  cd"})
	_, err := tokenize("\"ab")
	assert.Equal(t, err, errors.New("Unterminated string literal \""))
}

func TestTokenizeExpression(t *testing.T) {
	assert.Equal(t, tokenizeNoError(t, "a&&b"), []string{"a", "&", "&", "b"})

	// now chuck in whitespace: it should not affect the end result
	assert.Equal(t, tokenizeNoError(t, " a&&b"), []string{"a", "&", "&", "b"})
	assert.Equal(t, tokenizeNoError(t, "a &&b"), []string{"a", "&", "&", "b"})
	assert.Equal(t, tokenizeNoError(t, "a & &b"), []string{"a", "&", "&", "b"})
	assert.Equal(t, tokenizeNoError(t, "a && b"), []string{"a", "&", "&", "b"})
	assert.Equal(t, tokenizeNoError(t, "a &&b "), []string{"a", "&", "&", "b"})

	assert.Equal(t, tokenizeNoError(t, "^ab$"), []string{"^", "ab", "$"})
}

func TestTokenizeComplexExpression(t *testing.T) {
	assert.Equal(t, tokenizeNoError(t, "a&&b || (c && !d)"), []string{"a", "&", "&", "b", "|", "|", "(", "c", "&", "&", "!", "d", ")"})
}
