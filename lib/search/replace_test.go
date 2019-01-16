package search

import (
	"github.com/stvp/assert"
	"testing"
)

func replaceNoError(t *testing.T, rep []TokenReplacement, in []string) []string {
	return replace(rep, in)
}

func TestReplaceSimple(t *testing.T) {
	rep := []TokenReplacement{
		TokenReplacement{
			Search:  []string{"&", "&"},
			Replace: []string{"&&"},
		},
	}
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "&"}), []string{"&&"})
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "&", "&", "&"}), []string{"&&", "&&"})
	assert.Equal(t, replaceNoError(t, rep, []string{"a", "&", "&", "&"}), []string{"a", "&&", "&"})
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "a", "&", "&"}), []string{"&", "a", "&&"})
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "&", "a", "&"}), []string{"&&", "a", "&"})
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "&", "&", "a"}), []string{"&&", "&", "a"})
}

func TestReplaceMultiple(t *testing.T) {
	rep := []TokenReplacement{
		TokenReplacement{
			Search:  []string{"&", "&"},
			Replace: []string{"&&"},
		},
		TokenReplacement{
			Search:  []string{"|", "|"},
			Replace: []string{"||"},
		},
	}
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "&"}), []string{"&&"})
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "&", "&", "&"}), []string{"&&", "&&"})
	assert.Equal(t, replaceNoError(t, rep, []string{"a", "&", "&", "&"}), []string{"a", "&&", "&"})
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "a", "&", "&"}), []string{"&", "a", "&&"})
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "&", "a", "&"}), []string{"&&", "a", "&"})
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "&", "&", "a"}), []string{"&&", "&", "a"})

	assert.Equal(t, replaceNoError(t, rep, []string{"|", "|"}), []string{"||"})
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "&", "|", "|"}), []string{"&&", "||"})
	assert.Equal(t, replaceNoError(t, rep, []string{"&", "|", "&", "|"}), []string{"&", "|", "&", "|"})
	assert.Equal(t, replaceNoError(t, rep, []string{"a", "&", "&", "|", "|"}), []string{"a", "&&", "||"})
}
