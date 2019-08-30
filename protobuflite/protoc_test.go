package main

import (
	"fmt"
	"github.com/rburchell/gobo/lib/proto_parser"
	"github.com/stvp/assert"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type testField struct {
	name            string
	goName          string
	printfSpecifier string
	value           string
	isFloat         bool
}

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
		fmt.Fprintf(out, `fprintf(stderr, "Encoded field: %s => %s (wanted: %s)\n", valueToEncode.%s());`, field.name, field.printfSpecifier, field.value, proto_parser.CamelCaseName(field.name))
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
		fmt.Fprintf(out, `fprintf(stderr, "Decoded field: %s => %s (wanted: %s)\n", valueToDecode.%s());`, field.name, field.printfSpecifier, field.value, proto_parser.CamelCaseName(field.name))
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
		fmt.Fprintf(goSource, `log.Printf("Decoded field: %s => %s (wanted: %s)\n", pmsg.%s);`, field.name, field.printfSpecifier, field.value, field.goName)
		fmt.Fprintf(goSource, "\n")
		fmt.Fprintf(goSource, "if (pmsg.%s != %s) {\n", field.goName, field.value)
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

func TestSint32(t *testing.T) {
	// unsupported at present:
	//  However, there is an important difference between the signed int types (sint32 and sint64) and the "standard" int types (int32 and int64) when it comes to encoding negative numbers. If you use int32 or int64 as the type for a negative number, the resulting varint is always ten bytes long – it is, effectively, treated like a very large unsigned integer. If you use one of the signed types, the resulting varint uses ZigZag encoding, which is much more efficient.
	defer assert.Panic(t, "unsupported")
	runTest(`syntax = "proto3"; message PlaybackHeader { sint32 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

func TestSint64(t *testing.T) {
	// unsupported at present:
	//  However, there is an important difference between the signed int types (sint32 and sint64) and the "standard" int types (int32 and int64) when it comes to encoding negative numbers. If you use int32 or int64 as the type for a negative number, the resulting varint is always ten bytes long – it is, effectively, treated like a very large unsigned integer. If you use one of the signed types, the resulting varint uses ZigZag encoding, which is much more efficient.
	defer assert.Panic(t, "unsupported")
	runTest(`syntax = "proto3"; message PlaybackHeader { sint64 magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "1010",
		},
	})
}

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

func TestBool(t *testing.T) {
	runTest(`syntax = "proto3"; message PlaybackHeader { bool magic = 1; }`, "PlaybackHeader", []testField{
		testField{
			name:            "magic",
			goName:          "Magic",
			printfSpecifier: "%d",
			value:           "true",
		},
	})
}

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
