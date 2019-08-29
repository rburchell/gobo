package proto_parser

import (
	"github.com/stvp/assert"
	"testing"
)

func TestParseMessageSimple(t *testing.T) {
	msg := `message PlaybackHeader {
	uint32 magic = 1;
	float testfloat = 2;
	double testdouble = 3;
}`

	types := ParseTypes([]byte(msg))
	assert.Equal(t, types, []Message{
		Message{
			Type: "PlaybackHeader",
			Fields: []MessageField{
				MessageField{
					Type:        "uint32",
					RawName:     "magic",
					DisplayName: "magic",
					FieldNumber: 1,
				},
				MessageField{
					Type:        "float",
					RawName:     "testfloat",
					DisplayName: "testfloat",
					FieldNumber: 2,
				},
				MessageField{
					Type:        "double",
					RawName:     "testdouble",
					DisplayName: "testdouble",
					FieldNumber: 3,
				},
			},
		},
	})
}

func TestParseMessageNested(t *testing.T) {
	msg := `message PlaybackHeader {
	uint32 magic = 1;
	float testfloat = 2;
	double testdouble = 3;
}

message PlaybackFile {
	PlaybackHeader header = 1;
	bytes body = 2;
}`

	types := ParseTypes([]byte(msg))
	assert.Equal(t, types, []Message{
		Message{
			Type: "PlaybackHeader",
			Fields: []MessageField{
				MessageField{
					Type:        "uint32",
					RawName:     "magic",
					DisplayName: "magic",
					FieldNumber: 1,
				},
				MessageField{
					Type:        "float",
					RawName:     "testfloat",
					DisplayName: "testfloat",
					FieldNumber: 2,
				},
				MessageField{
					Type:        "double",
					RawName:     "testdouble",
					DisplayName: "testdouble",
					FieldNumber: 3,
				},
			},
		},
		Message{
			Type: "PlaybackFile",
			Fields: []MessageField{
				MessageField{
					Type:        "PlaybackHeader",
					RawName:     "header",
					DisplayName: "header",
					FieldNumber: 1,
				},
				MessageField{
					Type:        "bytes",
					RawName:     "body",
					DisplayName: "body",
					FieldNumber: 2,
				},
			},
		},
	})
}

func TestParseMessageWithSyntax(t *testing.T) {
	msg := `syntax = "proto3";

message PlaybackHeader {
	uint32 magic = 1;
	float testfloat = 2;
	double testdouble = 3;
}`

	types := ParseTypes([]byte(msg))
	assert.Equal(t, types, []Message{
		Message{
			Type: "PlaybackHeader",
			Fields: []MessageField{
				MessageField{
					Type:        "uint32",
					RawName:     "magic",
					DisplayName: "magic",
					FieldNumber: 1,
				},
				MessageField{
					Type:        "float",
					RawName:     "testfloat",
					DisplayName: "testfloat",
					FieldNumber: 2,
				},
				MessageField{
					Type:        "double",
					RawName:     "testdouble",
					DisplayName: "testdouble",
					FieldNumber: 3,
				},
			},
		},
	})
}
