package main

import (
	"fmt"
	"github.com/rburchell/gobo/lib/proto_parser"
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

func genTypes(types []proto_parser.Message) {
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
				fmt.Printf("    inline void %s(%s v) { m_%s = v; };\n", proto_parser.CamelCaseName("set_"+field.DisplayName), t, field.DisplayName)
			} else {
				fmt.Printf("    inline const %s %s() const { return m_%s; };\n", field.Type, field.DisplayName, field.DisplayName)
				fmt.Printf("    inline void %s(%s v) { m_%s = v; };\n", proto_parser.CamelCaseName("set_"+field.DisplayName), field.Type, field.DisplayName)
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
			wt := field.WireType()
			//fmt.Printf("    std::cout << \"Wrote field type and number \" << %d << %d << std::endl;\n", field.FieldNumber, wt)
			fmt.Printf("    ftd.v = ((%d << 3) | %d);\n", field.FieldNumber, wt)
			fmt.Printf("    stream.write(ftd);\n")
			switch wt {
			case proto_parser.VarIntWireType:
				fmt.Printf("    {\n")
				fmt.Printf("        VarInt vi;\n")
				fmt.Printf("        vi.v = t.%s();\n", field.DisplayName)
				fmt.Printf("        stream.write(vi);\n")
				fmt.Printf("    }\n")
			case proto_parser.Fixed64WireType:
				fmt.Printf("    {\n")
				fmt.Printf("        Fixed64 f64;\n")
				fmt.Printf("        auto tmp = t.%s();\n", field.DisplayName)
				fmt.Printf("        memcpy(&f64.v, &tmp, sizeof(f64.v));\n")
				fmt.Printf("        stream.write(f64);\n")
				fmt.Printf("    }\n")
			case proto_parser.LengthDelimitedWireType:
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
			case proto_parser.Fixed32WireType:
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
			switch field.WireType() {
			case proto_parser.VarIntWireType:
				fmt.Printf("            t.%s(vi.v);\n", proto_parser.CamelCaseName("set_"+field.DisplayName))
			case proto_parser.Fixed64WireType:
				fmt.Printf("            t.%s(*reinterpret_cast<%s*>(&f64.v));\n", proto_parser.CamelCaseName("set_"+field.DisplayName), field.Type)
			case proto_parser.LengthDelimitedWireType:
				t := mapTypeToCppType(field.Type)
				if t == "" {
					// message type
					fmt.Printf("            {\n")
					fmt.Printf("            %s td;\n", field.Type)
					fmt.Printf("            decode(ld.data, td);\n")
					fmt.Printf("            t.%s(td);\n", proto_parser.CamelCaseName("set_"+field.DisplayName))
					fmt.Printf("            }\n")
				} else {
					fmt.Printf("            t.%s(std::string((const char*)ld.data.data(), ld.data.size()));\n", proto_parser.CamelCaseName("set_"+field.DisplayName))
				}
			case proto_parser.Fixed32WireType:
				fmt.Printf("            t.%s(*reinterpret_cast<%s*>(&f32.v));\n", proto_parser.CamelCaseName("set_"+field.DisplayName), field.Type)
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

	types := proto_parser.ParseTypes(typeBuf)
	genTypes(types)
}
