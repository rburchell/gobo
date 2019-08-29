This is a .proto parser, parsing protobuf syntax files: https://developers.google.com/protocol-buffers/docs/proto3

It only supports proto3, and at the moment, only a subset of proto3.

Missing features (which may be added):

* Comments (`/* like so */`, or double slashed)
* Reserved fields to prevent field use: `message Foo { reserved 2, 15, 9 to 11; reserved "foo", "bar"; }`
* Enumerations (`enum Foo { BAR = 0; FOO = 1; }`)
    * Enumerations with aliases
* imports: `import "path/to/another.proto";`
* Nested types (`message Foo { message Bar { } Bar thing = 1; }`)
* oneof
* map: `map<key, value> field = 0;`
* Packages: `package foo.bar;`
* Field annotations, like deprecated: `int32 old_field = 6 [deprecated=true];`
* Options (e.g. `option optimize_for = CODE_SIZE;`)

Features that are probably out of scope:

* Service definitions (`service Foo { rpc ... }`) -- I don't mind adding parsing
  of this if it's useful, but I have no intention to write it myself.
