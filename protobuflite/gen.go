package main

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

func parseTypes(buf []byte) []Message {
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
				panic(fmt.Sprintf("Field number (%s) is not a number", fieldNum))
			}

			m.Fields = append(m.Fields, MessageField{
				Type:        fieldTypeName,
				RawName:     varName,
				DisplayName: camelCaseName(varName),
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

func mapTypeToCppType(typeName string) string {
	// ### XXX: not exhaustive
	switch typeName {
	case "uint64":
		return "uint64_t"
	case "int64":
		return "int64_t"
	case "uint32":
		return "uint32_t"
	case "int32":
		return "int32_t"
	case "float":
		return "float"
	case "double":
		return "double"
	case "bytes":
		return "std::string" // ### vector?
	case "string":
		return "std::string"
	}

	return ""
}

func wireTypeForField(typeName string) wireType {
	switch typeName {
	case "uint64":
		return varInt
	case "int64":
		return varInt
	case "uint32":
		return varInt
	case "int32":
		return varInt
	case "float":
		return fixed32
	case "double":
		return fixed64
	case "bytes":
		return lengthDelimited
	case "string":
		return lengthDelimited
	}

	// TODO: read known types and make sure it's one of them.
	log.Printf("Unknown type: %s", typeName)
	return lengthDelimited
}

func camelCaseName(name string) string {
	newName := ""
	nextIsUpper := false
	for _, c := range name {
		s := string(c)

		if s == "_" {
			nextIsUpper = true
			continue
		}

		if nextIsUpper {
			newName += strings.ToUpper(s)
			nextIsUpper = false
		} else {
			newName += s
		}
	}
	return newName
}

// TODO:
// varints
// encode type and field number together like protobuf

func genTypes(types []Message) {
	preamble()

	// headers
	for _, message := range types {
		fmt.Printf("struct %s\n", message.Type)
		fmt.Printf("{\n")
		for _, field := range message.Fields {
			t := mapTypeToCppType(field.Type)
			fmt.Printf("public:\n")
			if t != "" {
				fmt.Printf("    inline %s %s() const { return m_%s; };\n", t, field.DisplayName, field.DisplayName)
				fmt.Printf("    inline void %s(%s v) { m_%s = v; };\n", camelCaseName("set_"+field.DisplayName), t, field.DisplayName)
			} else {
				fmt.Printf("    inline const %s %s() const { return m_%s; };\n", field.Type, field.DisplayName, field.DisplayName)
				fmt.Printf("    inline void %s(%s v) { m_%s = v; };\n", camelCaseName("set_"+field.DisplayName), field.Type, field.DisplayName)
			}
			fmt.Printf("private:\n")
			if t != "" {
				fmt.Printf("    %s m_%s;\n", t, field.DisplayName)
			} else {
				fmt.Printf("    %s m_%s;\n", field.Type, field.DisplayName)
			}
		}
		fmt.Printf("};\n")

		fmt.Printf("std::vector<uint8_t> encode(const %s& t);\n", message.Type)
		fmt.Printf("void decode(const std::vector<uint8_t>& buffer, %s& v);\n", message.Type)
	}

	fmt.Printf("\n\n")

	for _, message := range types {
		fmt.Printf("inline std::vector<uint8_t> encode(const %s& t)\n", message.Type)
		fmt.Printf("{\n")
		fmt.Printf("    StreamWriter stream;\n")
		fmt.Printf("    VarInt ftd;\n")
		for _, field := range message.Fields {
			wt := wireTypeForField(field.Type)
			//fmt.Printf("    std::cout << \"Wrote field type and number \" << %d << %d << std::endl;\n", field.FieldNumber, wt)
			fmt.Printf("    ftd.v = ((%d << 3) | %d);\n", field.FieldNumber, wt)
			fmt.Printf("    stream.write(ftd);\n")
			switch wt {
			case varInt:
				fmt.Printf("    {\n")
				fmt.Printf("        VarInt vi;\n")
				fmt.Printf("        vi.v = t.%s();\n", field.DisplayName)
				fmt.Printf("        stream.write(vi);\n")
				fmt.Printf("    }\n")
			case fixed64:
				fmt.Printf("    {\n")
				fmt.Printf("        Fixed64 f64;\n")
				fmt.Printf("        auto tmp = t.%s();\n", field.DisplayName)
				fmt.Printf("        memcpy(&f64.v, &tmp, sizeof(f64.v));\n")
				fmt.Printf("        stream.write(f64);\n")
				fmt.Printf("    }\n")
			case lengthDelimited:
				fmt.Printf("    {\n")
				fmt.Printf("        LengthDelimited ld;\n")
				// We just assume that it contains valid UTF8.
				fmt.Printf("        auto data = t.%s();\n", field.DisplayName)
				t := mapTypeToCppType(field.Type)
				if t == "" {
					// message type
					fmt.Printf("        ld.data = encode(data);\n")
				} else {
					fmt.Printf("        ld.data = std::vector<uint8_t>(data.begin(), data.end());\n")
				}
				fmt.Printf("        stream.write(ld);\n")
				fmt.Printf("    }\n")
			case fixed32:
				fmt.Printf("    {\n")
				fmt.Printf("        Fixed32 f32;\n")
				fmt.Printf("        auto tmp = t.%s();\n", field.DisplayName)
				fmt.Printf("        memcpy(&f32.v, &tmp, sizeof(f32.v));\n")
				fmt.Printf("        stream.write(f32);\n")
				fmt.Printf("    }\n")
			default:
				panic("boom")
			}
		}
		fmt.Printf("    return stream.buffer();\n")
		fmt.Printf("}\n")
		fmt.Printf("inline void decode(const std::vector<uint8_t>& buffer, %s& t)\n", message.Type)
		fmt.Printf("{\n")
		fmt.Printf("    StreamReader stream(buffer);\n")
		fmt.Printf("    VarInt vi;\n")
		fmt.Printf("    Fixed64 f64;\n")
		fmt.Printf("    Fixed32 f32;\n")
		fmt.Printf("    LengthDelimited ld;\n")
		fmt.Printf("\n")
		fmt.Printf("    for (; !stream.is_eof(); ) {\n")
		fmt.Printf("        stream.start_transaction();\n")
		fmt.Printf("        VarInt ftd;\n")
		fmt.Printf("        stream.read(ftd);\n")
		fmt.Printf("        uint64_t fieldNumber = ftd.v >> 3;\n")
		fmt.Printf("        uint8_t fieldType = ftd.v & 0x07;\n")
		fmt.Printf("\n")
		//fmt.Printf("        std::cout << \"Read field type and number \" << fieldType << fieldNumber << std::endl;\n")
		fmt.Printf("\n")
		fmt.Printf("        switch (WireType(fieldType)) {\n")
		fmt.Printf("        case WireType::VarInt:\n")
		fmt.Printf("            stream.read(vi);\n")
		fmt.Printf("            break;\n")
		fmt.Printf("        case WireType::Fixed64:\n")
		fmt.Printf("            stream.read(f64);\n")
		fmt.Printf("            break;\n")
		fmt.Printf("        case WireType::Fixed32:\n")
		fmt.Printf("            stream.read(f32);\n")
		fmt.Printf("            break;\n")
		fmt.Printf("        case WireType::LengthDelimited:\n")
		fmt.Printf("            stream.read(ld);\n")
		fmt.Printf("            break;\n")
		fmt.Printf("        }\n")
		fmt.Printf("        if (!stream.commit_transaction()) {\n")
		fmt.Printf("            break;\n")
		fmt.Printf("        }\n")

		fmt.Printf("        switch (fieldNumber) {\n")
		for _, field := range message.Fields {
			fmt.Printf("        case %d:\n", field.FieldNumber)
			switch wireTypeForField(field.Type) {
			case varInt:
				fmt.Printf("            t.%s(vi.v);\n", camelCaseName("set_"+field.DisplayName))
			case fixed64:
				fmt.Printf("            t.%s(*reinterpret_cast<%s*>(&f64.v));\n", camelCaseName("set_"+field.DisplayName), field.Type)
			case lengthDelimited:
				t := mapTypeToCppType(field.Type)
				if t == "" {
					// message type
					fmt.Printf("            {\n")
					fmt.Printf("            %s td;\n", field.Type)
					fmt.Printf("            decode(ld.data, td);\n")
					fmt.Printf("            t.%s(td);\n", camelCaseName("set_"+field.DisplayName))
					fmt.Printf("            }\n")
				} else {
					fmt.Printf("            t.%s(std::string((const char*)ld.data.data(), ld.data.size()));\n", camelCaseName("set_"+field.DisplayName))
				}
			case fixed32:
				fmt.Printf("            t.%s(*reinterpret_cast<%s*>(&f32.v));\n", camelCaseName("set_"+field.DisplayName), field.Type)
			default:
				panic("boom")
			}
			fmt.Printf("            break;\n")
		}
		fmt.Printf("        }\n\n")
		fmt.Printf("    }\n\n")
		fmt.Printf("}\n\n")
	}
}

func main() {
	typeBuf := []byte(`
message PlaybackHeader {
	uint32 magic = 1;
	float testfloat = 2;
	double testdouble = 3;
}

message PlaybackFile {
	PlaybackHeader header = 1;
	bytes body = 2;
}
`)

	types := parseTypes(typeBuf)
	genTypes(types)
}
