package main

import "testing"
import "reflect"

func TestParse(t *testing.T) {
    c := ParseLine(":w00t TEST hello world")

    if (c.Prefix != "w00t") {
        t.Error("Expected w00t, got ", c.Prefix)
    }

    if (c.Command != "TEST") {
        t.Error("Expected TEST, got ", c.Command)
    }

    if (!reflect.DeepEqual(c.Parameters, []string{"hello", "world"})) {
        t.Error("Expected [hello world], got ", c.Parameters)
    }
}

func BenchmarkParseSingleLong(b *testing.B) {
    for i := 0; i < b.N; i++ {
        ParseLine(":w00t TEST :hello world")
    }
}

func BenchmarkParseMultipleShort(b *testing.B) {
    for i := 0; i < b.N; i++ {
        ParseLine(":w00t TEST hello world")
    }
}


func BenchmarkParseMultipleAndLong(b *testing.B) {
    for i := 0; i < b.N; i++ {
        ParseLine(":w00t TEST hello world :how are you today")
    }
}


