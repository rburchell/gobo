package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rburchell/gobo/lib/proto_parser"
)

// TODO: consider splitting the way these tests work up a bit
// Right now, there's a full pipeline, so:
// input -> C++ encode/decode (& verify) -> go decode (& verify)
// But perhaps we should rather have:
//     * C++ encode to bytes (& verify)
//     * C++ decode from bytes (& verify)
//     * Go encode to bytes (& verify)
//     * Go decode from bytes (& verify)
//
// ... as a set of discrete steps, and then mix and match them, with validation on each step:
//
// For example:
//     * C++ encode -> go decode
//     * C++ encode -> C++ decode
//     * go encode -> C++ decode
//
// This would also mean we'd be able to introduce other, different tests that aren't just strictly
// based on encoding of fields. For instance, we could test our behaviour of decoding multiple
// fields with the same tag, and other stuff like that, that is required by protobuf to be compliant
// with other implementations.

type testField struct {
	// The name of the field in the data type
	name string

	// The name that Go uses. Not really necessary, could write a capitalization helper instead.
	goName string

	// The printf specifier to use for the type in code (e.g. %d, %s)
	printfSpecifier string

	// The actual value. Either a quoted string for string/bytes, or an unquoted numeric value.
	value string

	// Whether or not this field is floating point
	// This is needed as floating point comparison must be fuzzy
	isFloat bool

	// Whether or not this field is string/bytes
	// Needed to correctly printf the value in C++
	isStringOrBytes bool
}

// Return the value suitable for inclusion inside a printf string (for example).
func (this testField) valuePrintf() string {
	if this.isStringOrBytes {
		// Quoted string. So unquote it.
		return this.value[1 : len(this.value)-1]
	}
	return this.value
}

// Write a C++ application to 'out' that encodes, and decodes, and verifies a set of fields.
// The given headerName is used to know what to include: the header is expected to provide all of the
// required type definitions, and serialization code.
func outputTestMain(out io.Writer, headerName string, typeToInstantiate string, fieldsToSet []testField) {
	fmt.Fprintf(out, `#include "%s"
#include <cmath>

bool approximatelyEqual(float a, float b, float epsilon)
{
	return std::fabs(a - b) <= ( (std::fabs(a) < std::fabs(b) ? std::fabs(b) : std::fabs(a)) * epsilon);
}

bool essentiallyEqual(float a, float b, float epsilon)
{
	return std::fabs(a - b) <= ( (std::fabs(a) > std::fabs(b) ? std::fabs(b) : std::fabs(a)) * epsilon);
}

int main(int argc, char **argv) {
	if (argc < 2) {
		fprintf(stderr, "No filename for output given");
		exit(1);
	}

	%s valueToEncode;`, headerName, typeToInstantiate)

	for _, field := range fieldsToSet {
		fmt.Fprintf(out, "valueToEncode.%s(%s);\n", proto_parser.CamelCaseName("set_"+field.name), field.value)
		if field.isStringOrBytes {
			fmt.Fprintf(out, `fprintf(stderr, "Encoded field: %s => %s (wanted: %s)\n", valueToEncode.%s().data());`, field.name, field.printfSpecifier, field.valuePrintf(), proto_parser.CamelCaseName(field.name))
		} else {
			fmt.Fprintf(out, `fprintf(stderr, "Encoded field: %s => %s (wanted: %s)\n", valueToEncode.%s());`, field.name, field.printfSpecifier, field.valuePrintf(), proto_parser.CamelCaseName(field.name))
		}
		fmt.Fprintf(out, "\n")
		if field.isFloat {
			fmt.Fprintf(out, "if (!essentiallyEqual(valueToEncode.%s(), %s, 0.000000000001)) {\n", proto_parser.CamelCaseName(field.name), field.value)
		} else {
			fmt.Fprintf(out, "if (valueToEncode.%s() != %s) {\n", proto_parser.CamelCaseName(field.name), field.value)
		}
		fmt.Fprintf(out, "    fprintf(stderr, \"Values did not match\\n\");\n")
		fmt.Fprintf(out, "    exit(1);\n")
		fmt.Fprintf(out, "}\n")
	}

	fmt.Fprintf(out, `
	fprintf(stderr, "Encoding...\n");
	std::vector<uint8_t> data = encode(valueToEncode);
	fprintf(stderr, "Writing to %%s...\n", argv[1]);
	FILE* f = fopen(argv[1], "wb");
	if (f == nullptr) {
		perror("open");
		exit(1);
	}
	fwrite(data.data(), data.size(), 1, f);
	fclose(f);
	fprintf(stderr, "Decoding...\n");
	`)

	fmt.Fprintf(out, `
	{
		%s valueToDecode;
		decode(data, valueToDecode);
		`, typeToInstantiate)

	for _, field := range fieldsToSet {

		if field.isStringOrBytes {
			fmt.Fprintf(out, `fprintf(stderr, "Decoded field: %s => %s (wanted: %s)\n", valueToDecode.%s().data());`, field.name, field.printfSpecifier, field.valuePrintf(), proto_parser.CamelCaseName(field.name))
		} else {
			fmt.Fprintf(out, `fprintf(stderr, "Decoded field: %s => %s (wanted: %s)\n", valueToDecode.%s());`, field.name, field.printfSpecifier, field.valuePrintf(), proto_parser.CamelCaseName(field.name))
		}
		fmt.Fprintf(out, "\n")
		if field.isFloat {
			fmt.Fprintf(out, "if (!essentiallyEqual(valueToDecode.%s(), %s, 0.000000000001)) {\n", proto_parser.CamelCaseName(field.name), field.value)
		} else {
			fmt.Fprintf(out, "if (valueToDecode.%s() != %s) {\n", proto_parser.CamelCaseName(field.name), field.value)
		}
		fmt.Fprintf(out, "    fprintf(stderr, \"Values did not match\\n\");\n")
		fmt.Fprintf(out, "    exit(1);\n")
		fmt.Fprintf(out, "}\n")
	}
	//fmt.Fprintf(out, `printf("Read: %%d %%f %%f\n", valueToDecode.magic(), valueToDecode.testfloat(), valueToDecode.testdouble());`)

	fmt.Fprintf(out, "}\n}")
}

// Compile and run a C++ test file.
// The returned byte slice is the encoded bytes on stdout, so that they can be verified
// against other external implementations.
func compileAndRun(file string) []byte {
	// Compile...
	tempBinary, err := ioutil.TempFile("", "protobuflite.testbinary.*")
	if err != nil {
		panic(fmt.Sprintf("Error acquiring binary: %s", err))
	}
	log.Printf("Compiling %s -> %s", file, tempBinary.Name())
	tempBinary.Close()
	defer os.Remove(tempBinary.Name())
	cmd := exec.Command("g++", "-o", tempBinary.Name(), file)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Compile output: %s", out)
		panic(fmt.Sprintf("Error compiling binary: %s", err))
	}

	// Run...
	outFile, err := ioutil.TempFile("", "protobuflite.outfile.*")
	if err != nil {
		panic(fmt.Sprintf("Error acquiring test output: %s", err))
	}
	log.Printf("Running %s -> %s", tempBinary.Name(), outFile.Name())
	outFile.Close()
	defer os.Remove(outFile.Name())
	testExec := exec.Command(tempBinary.Name(), outFile.Name())
	out, err = testExec.CombinedOutput()
	if err != nil {
		log.Printf("Runtime output: %s", out)
		panic(fmt.Sprintf("Error running test binary: %s", err))
	}
	log.Printf("Output: %s", out)

	ret, err := ioutil.ReadFile(outFile.Name())
	if err != nil {
		panic(fmt.Sprintf("Error fetching test output: %s", err))
	}
	return ret
}

// Verify the results of the cpp encoding process against go-protobuf.
// cppOut contains the (encoded) message from C++. protoSource contains the proto definition.
// testMessageName contains the message name in the .proto to decode.
// testFields contains information about all the fields in the .proto.
func verifyAgainstGo(cppOut []byte, protoSource string, testMessageName string, testFields []testField) {
	tmpDir, err := ioutil.TempDir("", "protobuflite.test.")
	if err != nil {
		panic(fmt.Sprintf("Error creating Go testdir: %s", err))
	}
	defer os.RemoveAll(tmpDir)
	ioutil.WriteFile(fmt.Sprintf("%s/foo.proto", tmpDir), []byte(protoSource), 0660)

	//log.Printf("protoc -I %s %s %s", tmpDir, fmt.Sprintf("--go_out=paths=source_relative,import_path=main:."), fmt.Sprintf("%s/foo.proto", tmpDir))
	cmd := exec.Command("protoc", "-I", tmpDir, fmt.Sprintf("--go_out=paths=source_relative,import_path=main:."), fmt.Sprintf("%s/foo.proto", tmpDir))
	cmd.Dir = tmpDir
	out, err := cmd.CombinedOutput()
	log.Printf("protoc output: %s", out)
	if err != nil {
		panic(fmt.Sprintf("Error running protoc: %s", err))
	}

	goSource, err := os.Create(fmt.Sprintf("%s/test.go", tmpDir))
	if err != nil {
		panic(fmt.Sprintf("Error creating Go source: %s", err))
	}

	fmt.Fprintf(goSource, `
	package main

	import (
		"os"
		"log"
		"io/ioutil"
		"github.com/golang/protobuf/proto"
	)

	func main() {
		msg, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			panic(err)
		}

		log.Printf("Decoding bytes: %%+v", msg)
		pmsg := &%s{}
		err = proto.Unmarshal(msg, pmsg)
		if err != nil {
			panic(err)
		}
		
		`, testMessageName)

	for _, field := range testFields {
		fmt.Fprintf(goSource, `log.Printf("Decoded field: %s => %s (wanted: %s)\n", pmsg.%s);`, field.name, field.printfSpecifier, field.valuePrintf(), field.goName)
		fmt.Fprintf(goSource, "\n")
		if field.isStringOrBytes {
			fmt.Fprintf(goSource, "if (string(pmsg.%s) != string(%s)) {\n", field.goName, field.value)
		} else {
			fmt.Fprintf(goSource, "if (pmsg.%s != %s) {\n", field.goName, field.value)
		}
		fmt.Fprintf(goSource, "    panic(\"Values did not match\\n\");\n")
		fmt.Fprintf(goSource, "}\n")
	}

	fmt.Fprintf(goSource, `
	}
	`)

	goCmd := exec.Command("go", "build")
	goCmd.Dir = tmpDir
	out, err = goCmd.CombinedOutput()
	log.Printf("Go build output: %s", out)
	if err != nil {
		panic(fmt.Sprintf("Error building Go test: %s", err))
	}

	goTestCmd := exec.Command(fmt.Sprintf("%s/%s", tmpDir, filepath.Base(tmpDir)))
	goTestCmd.Dir = tmpDir
	goTestIn, err := goTestCmd.StdinPipe()
	if err != nil {
		panic(fmt.Sprintf("Error getting Go test stdin pipe: %s", err))
	}
	go func() {
		defer goTestIn.Close()
		goTestIn.Write(cppOut)
	}()
	out, err = goTestCmd.CombinedOutput()
	log.Printf("Go test output: %s", out)
	if err != nil {
		panic(fmt.Sprintf("Error running Go test: %s", err))
	}
}

// Run a test using a given proto definition, testMessageName name to encode/decode,
// and testFields containing information about the fields in the protoSource.
func runTest(protoSource string, testMessageName string, testFields []testField) {
	typeBuf := []byte(protoSource)

	testHeader, err := ioutil.TempFile("", "protobuflite.header.*.h")
	if err != nil {
		panic(fmt.Sprintf("Error acquiring header file: %s", err))
	}
	defer os.Remove(testHeader.Name())
	testSource, err := ioutil.TempFile("", "protobuflite.source.*.cpp")
	if err != nil {
		panic(fmt.Sprintf("Error acquiring source file: %s", err))
	}
	defer os.Remove(testSource.Name())

	types := proto_parser.ParseTypes(typeBuf)
	genTypes(testHeader, types)

	outputTestMain(testSource, testHeader.Name(), testMessageName, testFields)
	cppOutput := compileAndRun(testSource.Name())

	verifyAgainstGo(cppOutput, protoSource, testMessageName, testFields)
}

////////////////////////////////////////////////////////////////////////////////
// tests below this point
////////////////////////////////////////////////////////////////////////////////

// Test encoding of the double type.
func TestDouble(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { double magic = 1; } `, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%f",
			value:           "5.678",
			isFloat:         true,
		},
	})
}

// Test encoding of the float type.
func TestFloat(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { float magic = 1; } `, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%f",
			value:           "1.234",
			isFloat:         true,
		},
	})
}

// Test encoding of the uint32 type.
func TestUint32(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { uint32 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

// Test encoding of the int32 type.
func TestInt32(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { int32 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

// Test encoding of the uint64 type.
func TestUint64(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { uint64 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

// Test encoding of the int64 type.
func TestInt64(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { int64 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

// Test encoding of the sint32 type.
func TestSint32(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { sint32 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

// Test encoding of the sint64 type.
func TestSint64(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { sint64 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

// Test encoding of the fixed32 type.
func TestFixed32(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { fixed32 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

// Test encoding of the fixed64 type.
func TestFixed64(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { fixed64 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

// Test encoding of the sfixed32 type.
func TestSfixed32(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { sfixed32 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

// Test encoding of the sfixed64 type.
func TestSfixed64(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { sfixed64 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

// Test encoding of the bool type.
func TestBool(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { bool magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "true",
		},
	})
	runTest(`syntax = "proto3"; message PlaybackHeader { bool magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "false",
		},
	})
}

// Test encoding of the string type.
func TestString(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { string magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%s",
			value:           "\"hello world string\"",
			isStringOrBytes: true,
		},
	})
}

// Test encoding of the bytes type.
func TestBytes(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { bytes magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%s",
			value:           "\"hello world bytes\"",
			isStringOrBytes: true,
		},
	})
}

// Test behavior of multiple field encoding.
func TestMultipleFields(t *testing.T) {
	runTest(`
syntax = "proto3";
message PlaybackHeader {
	uint32 magic = 1;
	float testfloat = 2;
	double testdouble = 3;
}
`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
		testField{
			name:            "testfloat",
			goName:          "Testfloat",
			printfSpecifier: "%f",
			value:           "1.234",
			isFloat:         true,
		},
		testField{
			name:            "testdouble",
			goName:          "Testdouble",
			printfSpecifier: "%f",
			value:           "5.678",
			isFloat:         true,
		},
	})
}

// Test behavior of multiple field encoding, with a gap in the field numbers.
func TestMultipleFieldsWithNumberGap(t *testing.T) {
	runTest(`
syntax = "proto3";
message PlaybackHeader {
	uint32 magic = 1;
	float testfloat = 4;
	double testdouble = 5;
}
`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
		testField{
			name:            "testfloat",
			goName:          "Testfloat",
			printfSpecifier: "%f",
			value:           "1.234",
			isFloat:         true,
		},
		testField{
			name:            "testdouble",
			goName:          "Testdouble",
			printfSpecifier: "%f",
			value:           "5.678",
			isFloat:         true,
		},
	})
}
