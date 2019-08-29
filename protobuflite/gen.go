package main

import (
	"fmt"
	"github.com/rburchell/gobo/lib/proto_parser"
	"io"
	"os"
)

// Return the C++ type for a given protobuf type.
// If the type is not a built-in type (i.e. it's a reference to another type),
// then return empty string.
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

func genTypes(out io.Writer, types []proto_parser.Message) {
	preamble(out)

	// headers
	for _, message := range types {
		fmt.Fprintf(out, "struct %s\n", message.Type)
		fmt.Fprintf(out, "{\n")
		for _, field := range message.Fields {
			t := mapTypeToCppType(field.Type)
			fmt.Fprintf(out, "public:\n")
			if t != "" {
				fmt.Fprintf(out, "    inline %s %s() const { return m_%s; };\n", t, field.DisplayName, field.DisplayName)
				fmt.Fprintf(out, "    inline void %s(%s v) { m_%s = v; };\n", proto_parser.CamelCaseName("set_"+field.DisplayName), t, field.DisplayName)
			} else {
				fmt.Fprintf(out, "    inline const %s %s() const { return m_%s; };\n", field.Type, field.DisplayName, field.DisplayName)
				fmt.Fprintf(out, "    inline void %s(%s v) { m_%s = v; };\n", proto_parser.CamelCaseName("set_"+field.DisplayName), field.Type, field.DisplayName)
			}
			fmt.Fprintf(out, "private:\n")
			if t != "" {
				fmt.Fprintf(out, "    %s m_%s;\n", t, field.DisplayName)
			} else {
				fmt.Fprintf(out, "    %s m_%s;\n", field.Type, field.DisplayName)
			}
		}
		fmt.Fprintf(out, "};\n")

		fmt.Fprintf(out, "std::vector<uint8_t> encode(const %s& t);\n", message.Type)
		fmt.Fprintf(out, "void decode(const std::vector<uint8_t>& buffer, %s& v);\n", message.Type)
	}

	fmt.Fprintf(out, "\n\n")

	for _, message := range types {
		fmt.Fprintf(out, "inline std::vector<uint8_t> encode(const %s& t)\n", message.Type)
		fmt.Fprintf(out, "{\n")
		fmt.Fprintf(out, "    StreamWriter stream;\n")
		fmt.Fprintf(out, "    VarInt ftd;\n")
		for _, field := range message.Fields {
			wt := field.WireType()
			//fmt.Fprintf(out, "    std::cout << \"Wrote field type and number \" << %d << %d << std::endl;\n", field.FieldNumber, wt)
			fmt.Fprintf(out, "    ftd.v = ((%d << 3) | %d);\n", field.FieldNumber, wt)
			fmt.Fprintf(out, "    stream.write(ftd);\n")
			switch wt {
			case proto_parser.VarIntWireType:
				fmt.Fprintf(out, "    {\n")
				fmt.Fprintf(out, "        VarInt vi;\n")
				fmt.Fprintf(out, "        vi.v = t.%s();\n", field.DisplayName)
				fmt.Fprintf(out, "        stream.write(vi);\n")
				fmt.Fprintf(out, "    }\n")
			case proto_parser.Fixed64WireType:
				fmt.Fprintf(out, "    {\n")
				fmt.Fprintf(out, "        Fixed64 f64;\n")
				fmt.Fprintf(out, "        auto tmp = t.%s();\n", field.DisplayName)
				fmt.Fprintf(out, "        memcpy(&f64.v, &tmp, sizeof(f64.v));\n")
				fmt.Fprintf(out, "        stream.write(f64);\n")
				fmt.Fprintf(out, "    }\n")
			case proto_parser.LengthDelimitedWireType:
				fmt.Fprintf(out, "    {\n")
				fmt.Fprintf(out, "        LengthDelimited ld;\n")
				// We just assume that it contains valid UTF8.
				fmt.Fprintf(out, "        auto data = t.%s();\n", field.DisplayName)
				t := mapTypeToCppType(field.Type)
				if t == "" {
					// message type
					fmt.Fprintf(out, "        ld.data = encode(data);\n")
				} else {
					fmt.Fprintf(out, "        ld.data = std::vector<uint8_t>(data.begin(), data.end());\n")
				}
				fmt.Fprintf(out, "        stream.write(ld);\n")
				fmt.Fprintf(out, "    }\n")
			case proto_parser.Fixed32WireType:
				fmt.Fprintf(out, "    {\n")
				fmt.Fprintf(out, "        Fixed32 f32;\n")
				fmt.Fprintf(out, "        auto tmp = t.%s();\n", field.DisplayName)
				fmt.Fprintf(out, "        memcpy(&f32.v, &tmp, sizeof(f32.v));\n")
				fmt.Fprintf(out, "        stream.write(f32);\n")
				fmt.Fprintf(out, "    }\n")
			default:
				panic(fmt.Sprintf("Unknown wiretype on encode: %d", wt))
			}
		}
		fmt.Fprintf(out, "    return stream.buffer();\n")
		fmt.Fprintf(out, "}\n")
		fmt.Fprintf(out, "inline void decode(const std::vector<uint8_t>& buffer, %s& t)\n", message.Type)
		fmt.Fprintf(out, "{\n")
		fmt.Fprintf(out, "    StreamReader stream(buffer);\n")
		fmt.Fprintf(out, "    VarInt vi;\n")
		fmt.Fprintf(out, "    Fixed64 f64;\n")
		fmt.Fprintf(out, "    Fixed32 f32;\n")
		fmt.Fprintf(out, "    LengthDelimited ld;\n")
		fmt.Fprintf(out, "\n")
		fmt.Fprintf(out, "    for (; !stream.is_eof(); ) {\n")
		fmt.Fprintf(out, "        stream.start_transaction();\n")
		fmt.Fprintf(out, "        VarInt ftd;\n")
		fmt.Fprintf(out, "        stream.read(ftd);\n")
		fmt.Fprintf(out, "        uint64_t fieldNumber = ftd.v >> 3;\n")
		fmt.Fprintf(out, "        uint8_t fieldType = ftd.v & 0x07;\n")
		fmt.Fprintf(out, "\n")
		//fmt.Fprintf(out, "        std::cout << \"Read field type and number \" << fieldType << fieldNumber << std::endl;\n")
		fmt.Fprintf(out, "\n")
		fmt.Fprintf(out, "        switch (WireType(fieldType)) {\n")
		fmt.Fprintf(out, "        case WireType::VarInt:\n")
		fmt.Fprintf(out, "            stream.read(vi);\n")
		fmt.Fprintf(out, "            break;\n")
		fmt.Fprintf(out, "        case WireType::Fixed64:\n")
		fmt.Fprintf(out, "            stream.read(f64);\n")
		fmt.Fprintf(out, "            break;\n")
		fmt.Fprintf(out, "        case WireType::Fixed32:\n")
		fmt.Fprintf(out, "            stream.read(f32);\n")
		fmt.Fprintf(out, "            break;\n")
		fmt.Fprintf(out, "        case WireType::LengthDelimited:\n")
		fmt.Fprintf(out, "            stream.read(ld);\n")
		fmt.Fprintf(out, "            break;\n")
		fmt.Fprintf(out, "        }\n")
		fmt.Fprintf(out, "        if (!stream.commit_transaction()) {\n")
		fmt.Fprintf(out, "            break;\n")
		fmt.Fprintf(out, "        }\n")

		fmt.Fprintf(out, "        switch (fieldNumber) {\n")
		for _, field := range message.Fields {
			fmt.Fprintf(out, "        case %d:\n", field.FieldNumber)
			switch field.WireType() {
			case proto_parser.VarIntWireType:
				fmt.Fprintf(out, "            t.%s(vi.v);\n", proto_parser.CamelCaseName("set_"+field.DisplayName))
			case proto_parser.Fixed64WireType:
				fmt.Fprintf(out, "            t.%s(*reinterpret_cast<%s*>(&f64.v));\n", proto_parser.CamelCaseName("set_"+field.DisplayName), field.Type)
			case proto_parser.LengthDelimitedWireType:
				t := mapTypeToCppType(field.Type)
				if t == "" {
					// message type
					fmt.Fprintf(out, "            {\n")
					fmt.Fprintf(out, "            %s td;\n", field.Type)
					fmt.Fprintf(out, "            decode(ld.data, td);\n")
					fmt.Fprintf(out, "            t.%s(td);\n", proto_parser.CamelCaseName("set_"+field.DisplayName))
					fmt.Fprintf(out, "            }\n")
				} else {
					fmt.Fprintf(out, "            t.%s(std::string((const char*)ld.data.data(), ld.data.size()));\n", proto_parser.CamelCaseName("set_"+field.DisplayName))
				}
			case proto_parser.Fixed32WireType:
				fmt.Fprintf(out, "            t.%s(*reinterpret_cast<%s*>(&f32.v));\n", proto_parser.CamelCaseName("set_"+field.DisplayName), field.Type)
			default:
				panic(fmt.Sprintf("Unknown wiretype on decode: %d", field.WireType()))
			}
			fmt.Fprintf(out, "            break;\n")
		}
		fmt.Fprintf(out, "        }\n\n")
		fmt.Fprintf(out, "    }\n\n")
		fmt.Fprintf(out, "}\n\n")
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

	types := proto_parser.ParseTypes(typeBuf)
	genTypes(os.Stdout, types)
}
