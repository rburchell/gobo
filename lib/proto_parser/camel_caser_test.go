package proto_parser

import (
	"github.com/stvp/assert"
	"testing"
)

func TestCamelcasing(t *testing.T) {
	assert.Equal(t, CamelCaseName("test"), "test")
	assert.Equal(t, CamelCaseName("test_name"), "testName")
}
