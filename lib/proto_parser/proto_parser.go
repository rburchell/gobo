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
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	tokens = cleanTokens(tokens)
	return tokens
}

type MessageField struct {
	Type        string
	RawName     string
	DisplayName string
	FieldNumber int
}
type Message struct {
	Type   string
	Fields []MessageField
}

func ParseTypes(buf []byte) []Message {
	tokens := parseBuffer(buf)

	handleMessage := func(tokens []string) (int, Message) {
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
			})
			typeIndex += 5
		}

		if tokens[typeIndex] != "}" {
			panic(fmt.Sprintf("Expected: message must have a close brace after vars, got %s)", tokens[typeIndex]))
		}
		return typeIndex + 1, m
	}

	types := []Message{}
	for i := 0; i < len(tokens); {
		tok := tokens[i]
		switch tok {
		case "message":
			i++
			consumed, msg := handleMessage(tokens[i:])
			i += consumed
			types = append(types, msg)

		default:
			panic(fmt.Sprintf("Unexpected top level token: %s", tok))
		}
	}

	return types
}
