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
	Input          string
	Tags           []IrcTag
	Prefix         IrcPrefix
	Command        string
	Parameters     []string
	ExpectedOutput string
}

func TestParse(t *testing.T) {
	tests := []ParserTest{
		{
			":w00t TEST",
			[]IrcTag{},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			":w00t TEST",
		},
		{
			":w00t TEST hello",
			[]IrcTag{},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{"hello"},
			":w00t TEST hello",
		},
		{
			":w00t TEST hello world",
			[]IrcTag{},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{"hello", "world"},
			":w00t TEST hello world",
		},
		{
			":w00t TEST :hello world",
			[]IrcTag{},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{"hello world"},
			":w00t TEST :hello world",
		},
		{
			":w00t TEST hello world :how are you today",
			[]IrcTag{},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{"hello", "world", "how are you today"},
			":w00t TEST hello world :how are you today",
		},

		{
			"TEST",
			[]IrcTag{},
			IrcPrefix{},
			"TEST",
			[]string{},
			"TEST",
		},
		{
			"TEST hello",
			[]IrcTag{},
			IrcPrefix{},
			"TEST",
			[]string{"hello"},
			"TEST hello",
		},
		{
			"TEST hello world",
			[]IrcTag{},
			IrcPrefix{},
			"TEST",
			[]string{"hello", "world"},
			"TEST hello world",
		},
		{
			"TEST :hello world",
			[]IrcTag{},
			IrcPrefix{},
			"TEST",
			[]string{"hello world"},
			"TEST :hello world",
		},
		{
			"TEST hello world :how are you today",
			[]IrcTag{},
			IrcPrefix{},
			"TEST",
			[]string{"hello", "world", "how are you today"},
			"TEST hello world :how are you today",
		},

		// test prefix parsing
		{
			":w00t TEST",
			[]IrcTag{},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			":w00t TEST",
		},
		{
			":w00t!toot@moo TEST",
			[]IrcTag{},
			IrcPrefix{Nick: "w00t", User: "toot", Host: "moo"},
			"TEST",
			[]string{},
			":w00t!toot@moo TEST",
		},
		{
			":w00t!toot@moo.cows TEST",
			[]IrcTag{},
			IrcPrefix{Nick: "w00t", User: "toot", Host: "moo.cows"},
			"TEST",
			[]string{},
			":w00t!toot@moo.cows TEST",
		},
		{
			":w00t.toot.moo.cows TEST",
			[]IrcTag{},
			IrcPrefix{Server: "w00t.toot.moo.cows"},
			"TEST",
			[]string{},
			":w00t.toot.moo.cows TEST",
		},
		{
			":w00t!toot TEST",
			[]IrcTag{},
			IrcPrefix{}, // invalid
			"TEST",
			[]string{},
			"TEST",
		},
		{
			":w00t@toot TEST",
			[]IrcTag{},
			IrcPrefix{}, // invalid
			"TEST",
			[]string{},
			"TEST",
		},
		{
			":@! TEST",
			[]IrcTag{},
			IrcPrefix{}, // invalid
			"TEST",
			[]string{},
			"TEST",
		},

		// ircv3 message-tags
		{
			"@aaaa :w00t TEST",
			[]IrcTag{IrcTag{Key: "aaaa"}},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			":w00t TEST",
		},
		{
			"@aaaa;bbb;cccc :w00t TEST",
			[]IrcTag{IrcTag{Key: "aaaa"}, IrcTag{Key: "bbb"}, IrcTag{Key: "cccc"}},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			":w00t TEST",
		},
		{
			"@aaaa=test;bbb :w00t TEST",
			[]IrcTag{IrcTag{Key: "aaaa", Value: "test"}, IrcTag{Key: "bbb"}},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			":w00t TEST",
		},
		{
			"@example.org/aaaa=test;bbb :w00t TEST",
			[]IrcTag{IrcTag{VendorPrefix: "example.org", Key: "aaaa", Value: "test"}, IrcTag{Key: "bbb"}},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			":w00t TEST",
		},
		{
			"@example.org/aaaa=test;another.example.org/bbb :w00t TEST",
			[]IrcTag{IrcTag{VendorPrefix: "example.org", Key: "aaaa", Value: "test"}, IrcTag{VendorPrefix: "another.example.org", Key: "bbb"}},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			":w00t TEST",
		},
		{
			"@aaaa=test;bbb=another :w00t TEST",
			[]IrcTag{IrcTag{Key: "aaaa", Value: "test"}, IrcTag{Key: "bbb", Value: "another"}},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			":w00t TEST",
		},
		{
			// test escaping of tag values
			"@aaaa=magic\\:things\\s\\\\happen\\rhere\\nsometimes :w00t TEST",
			[]IrcTag{IrcTag{Key: "aaaa", Value: "magic;things \\happen\rhere\nsometimes"}},
			IrcPrefix{Nick: "w00t"},
			"TEST",
			[]string{},
			":w00t TEST",
		},
	}

	for _, test := range tests {
		t.Logf("Testing: %s", test.Input)

		c := ParseLine(test.Input)

		if len(test.Tags) > 0 {
			for tidx, tag := range test.Tags {
				actualtag := c.Tags[tidx]

				if tag.Key != actualtag.Key {
					t.Errorf("Expected: tag with key %#v, got %#v", tag.Key, actualtag.Key)
				}

				if tag.VendorPrefix != actualtag.VendorPrefix {
					t.Errorf("Expected: tag with key %#v from vendor %#v, got vendor %#v", tag.Key, tag.VendorPrefix, actualtag.VendorPrefix)
				}

				if tag.Value != actualtag.Value {
					t.Errorf("Expected: tag with key %#v with value %#v, got value %#v", tag.Key, tag.Value, actualtag.Value)
				}
			}
		}

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
		if c.String() != test.ExpectedOutput {
			t.Errorf("Expected: %#v, got %#v", test.ExpectedOutput, c.String())
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

func BenchmarkServerPrefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseLine(":server.name.goes.here TEST")
	}
}

func BenchmarkNickPrefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseLine(":nick TEST")
	}
}

func BenchmarkNickUserHostPrefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseLine(":nick!user@host TEST")
	}
}

func BenchmarkTagKeyOnly(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseLine("@aaaa :nick TEST")
	}
}

func BenchmarkTagKeysOnly(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseLine("@aaaa;bbbb :nick TEST")
	}
}

func BenchmarkTagKeysAndValues(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseLine("@aaaa=onetwoonetwo;bbbb=onetwoonetwo :nick TEST")
	}
}

func BenchmarkTagKeyWithVendorPrefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseLine("@example.org/aaaa :nick TEST")
	}
}

func BenchmarkTagKeyWithVendorPrefixAndLotsOfParameters(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseLine("@example.org/aaaa :nick!user@host TEST this is a command with rather a :large number of parameters included")
	}
}
