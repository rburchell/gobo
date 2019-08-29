package proto_parser

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// message Foo {
//   type foo = 1;
// }
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t'
}

func isAlphaNum(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z':
		fallthrough
	case c >= 'A' && c <= 'Z':
		fallthrough
	case c >= '0' && c <= '9':
		return true
	}
	return false
}

func parseLine(line string) []string {
	scanner := bufio.NewScanner(strings.NewReader(line))

	scanWord := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		var i int
		done := false
		for i = 0; !done && i < len(data); {
			c := data[i]
			switch {
			case isAlphaNum(c):
				i++
			default:
				done = true
			}
		}

		if i == 0 {
			panic("didn't find anything word-like")
		}

		return i, data[0:i], nil
	}

	scanWhitespace := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		var i int
		done := false
		for i = 0; !done && i < len(data); {
			c := data[i]
			if !isWhitespace(c) {
				break
			} else {
				i++
			}
		}

		return i, []byte{' '}, nil
	}

	scanStringLiteral := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		var i int
		done := false

		// We are called on the first '"', so skip it.
		for i = 1; !done && i < len(data); {
			c := data[i]
			switch {
			case c == '"':
				done = true
			default:
				i++
			}
		}

		return i, data[1:i], nil
	}

	split := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF {
			return 0, nil, nil
		}
		c := data[0]
		switch {
		case c >= 'a' && c <= 'z':
			fallthrough
		case c >= 'A' && c <= 'Z':
			fallthrough
		case c >= '0' && c <= '9':
			return scanWord(data, atEOF)
		case c == ' ' || c == '\t':
			return scanWhitespace(data, atEOF)
		case c == '"':
			return scanStringLiteral(data, atEOF)
		case c == '{':
			fallthrough
		case c == '}':
			fallthrough
		case c == ';':
			fallthrough
		case c == '=':
			return 1, []byte{c}, nil
		default:
			log.Printf("WAT: %q", data)
		}
		panic("boom")
	}
	scanner.Split(split)

	tokens := []string{}

	for scanner.Scan() {
		tokens = append(tokens, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Invalid input: %s", err)
	}

	return tokens
}

func cleanTokens(tokens []string) []string {
	cleanTokens := []string{}

	// remove whitespace to make life easier
	for _, tok := range tokens {
		if isWhitespace(tok[0]) {
			continue
		}
		cleanTokens = append(cleanTokens, tok)
	}

	return cleanTokens
}

func parseBuffer(buf []byte) []string {
	scanner := bufio.NewScanner(bytes.NewBuffer(buf))
	tokens := []string{}
	for scanner.Scan() {
		tokens = append(tokens, parseLine(scanner.Text())...)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading buffer:", err)
	}

	tokens = cleanTokens(tokens)
	return tokens
}

type MessageField struct {
	Type        string
	RawName     string
	DisplayName string
	FieldNumber int
	IsRepeated  bool
}

// Return the protobuf wiretype for a field.
func (this MessageField) WireType() WireType {
	switch this.Type {
	case "uint64":
		return VarIntWireType
	case "int64":
		return VarIntWireType
	case "uint32":
		return VarIntWireType
	case "int32":
		return VarIntWireType
	case "float":
		return Fixed32WireType
	case "double":
		return Fixed64WireType
	case "bytes":
		return LengthDelimitedWireType
	case "string":
		return LengthDelimitedWireType
	}

	// This is some kind of custom field (or we just don't know about it yet).
	// TODO: handle bool, etc.
	return LengthDelimitedWireType
}

type Message struct {
	Type   string
	Fields []MessageField
}

// TODO: return error instead of panicing
func ParseTypes(buf []byte) []Message {
	tokens := parseBuffer(buf)
	types := []Message{}
	for i := 0; i < len(tokens); {
		tok := tokens[i]
		switch tok {
		case "message":
			i++
			consumed, msg := handleMessage(tokens[i:])
			i += consumed
			types = append(types, msg)
		case "syntax":
			i++
			consumed := handleSyntax(tokens[i:])
			i += consumed
		default:
			panic(fmt.Sprintf("Unexpected top level token: %s", tok))
		}
	}

	return types
}

func handleSyntax(tokens []string) int {
	if len(tokens) < 3 {
		panic("Expected: syntax = \"proto3\" (insufficient tokens)")
	}

	if tokens[0] != "=" {
		panic(fmt.Sprintf("Expected: syntax = \"proto3\"; (%s is not =)", tokens[0]))
	}

	// Technically this means we are laxer than we should be.
	// We accept "syntax = proto3", though it must be quoted...
	// We should return string literals quoted, perhaps?
	if tokens[1] != "proto3" {
		panic(fmt.Sprintf("Expected: syntax = \"proto3\"; (%s is not proto3)", tokens[1]))
	}

	if tokens[2] != ";" {
		panic(fmt.Sprintf("Expected: syntax = \"proto3\"; (%s is not ;)", tokens[2]))
	}

	return 3
}

func handleMessage(tokens []string) (int, Message) {
	if len(tokens) < 3 {
		panic("Expected: message Foo {} (insufficient tokens)")
	}

	typeName := tokens[0]
	if !isAlphaNum(typeName[0]) {
		panic(fmt.Sprintf("Expected: message typeName {} (%s is not a type name)", typeName))
	}

	if tokens[1] != "{" {
		panic(fmt.Sprintf("Expected: message must have an open brace after type name, got %s)", tokens[1]))
	}

	m := Message{Type: typeName}

	var typeIndex int
	// now read type -> var = fieldNames
	for typeIndex = 2; typeIndex < len(tokens)-4 && isAlphaNum(tokens[typeIndex+0][0]); {
		fieldTypeName := tokens[typeIndex+0]
		isRepeated := false
		if fieldTypeName == "repeated" {
			isRepeated = true
			typeIndex++
			fieldTypeName = tokens[typeIndex+0]
		}

		varName := tokens[typeIndex+1]
		equals := tokens[typeIndex+2]
		if equals != "=" {
			panic(fmt.Sprintf("Field %s has no '= fieldnum' (got %s instead of equals)", varName, equals))
		}
		fieldNumber := tokens[typeIndex+3]
		semi := tokens[typeIndex+4]
		if semi != ";" {
			panic(fmt.Sprintf("Field %s has no terminating semicolon (got %s instead of ;)", varName, semi))
		}

		fieldNum, err := strconv.Atoi(fieldNumber)
		if err != nil {
			panic(fmt.Sprintf("Field number (%d) is not a number", fieldNum))
		}

		m.Fields = append(m.Fields, MessageField{
			Type:        fieldTypeName,
			RawName:     varName,
			DisplayName: CamelCaseName(varName),
			FieldNumber: fieldNum,
			IsRepeated:  isRepeated,
		})
		typeIndex += 5
	}

	if typeIndex >= len(tokens) {
		panic(fmt.Sprintf("Expected: message must have a close brace after vars, got unexpected EOF"))
	}

	if tokens[typeIndex] != "}" {
		panic(fmt.Sprintf("Expected: message must have a close brace after vars, got %s)", tokens[typeIndex]))
	}
	return typeIndex + 1, m
}
