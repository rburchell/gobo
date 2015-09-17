/*
 * Copyright (C) 2015 Robin Burchell <robin+git@viroteck.net>
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

package parser

import "testing"
import "reflect"

type ParserTest struct {
	Input      string
	Prefix     IrcPrefix
	Command    string
	Parameters []string
	BadPrefix  bool
}

func TestParse(t *testing.T) {
	tests := []ParserTest{
		{
			":w00t TEST",
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			false,
		},
		{
			":w00t TEST hello",
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{"hello"},
			false,
		},
		{
			":w00t TEST hello world",
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{"hello", "world"},
			false,
		},
		{
			":w00t TEST :hello world",
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{"hello world"},
			false,
		},
		{
			":w00t TEST hello world :how are you today",
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{"hello", "world", "how are you today"},
			false,
		},

		{
			"TEST",
			IrcPrefix{},
			"TEST",
			[]string{},
			false,
		},
		{
			"TEST hello",
			IrcPrefix{},
			"TEST",
			[]string{"hello"},
			false,
		},
		{
			"TEST hello world",
			IrcPrefix{},
			"TEST",
			[]string{"hello", "world"},
			false,
		},
		{
			"TEST :hello world",
			IrcPrefix{},
			"TEST",
			[]string{"hello world"},
			false,
		},
		{
			"TEST hello world :how are you today",
			IrcPrefix{},
			"TEST",
			[]string{"hello", "world", "how are you today"},
			false,
		},

		// test prefix parsing
		{
			":w00t TEST",
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			false,
		},
		{
			":w00t!toot@moo TEST",
			IrcPrefix{Nick: "w00t", User: "toot", Host: "moo"},
			"TEST",
			[]string{},
			false,
		},
		{
			":w00t!toot@moo.cows TEST",
			IrcPrefix{Nick: "w00t", User: "toot", Host: "moo.cows"},
			"TEST",
			[]string{},
			false,
		},
		{
			":w00t.toot.moo.cows TEST",
			IrcPrefix{Server: "w00t.toot.moo.cows"},
			"TEST",
			[]string{},
			false,
		},
		{
			":w00t!toot TEST",
			IrcPrefix{}, // invalid
			"TEST",
			[]string{},
			true,
		},
		{
			":w00t@toot TEST",
			IrcPrefix{}, // invalid
			"TEST",
			[]string{},
			true,
		},
	}

	for _, test := range tests {
		t.Logf("Testing: %s", test.Input)

		c := ParseLine(test.Input)
		if c.Prefix != test.Prefix {
			t.Errorf("Expected: %#v, got %#v", test.Prefix, c.Prefix)
		}

		if c.Command != test.Command {
			t.Errorf("Expected: %#v, got %#v", test.Command, c.Command)
		}

		if !reflect.DeepEqual(c.Parameters, test.Parameters) {
			t.Errorf("Expected: %#v, got %#v", test.Parameters, c.Parameters)
		}

		// also test that converting back to string works
		// TODO: remove BadPrefix, add a seperate "expected output" for the
		// BadPrefix cases.
		if !test.BadPrefix {
			if c.String() != test.Input {
				t.Errorf("Expected: %#v, got %#v", test.Input, c.String())
			}
		}
	}
}

func BenchmarkString(b *testing.B) {
	c := ParseLine(":w00t TEST :hello world")
	for i := 0; i < b.N; i++ {
		c.String()
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
