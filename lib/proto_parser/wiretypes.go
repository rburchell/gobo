package proto_parser

type WireType int

// These match up with the protobuf wiretypes.
// That's also why they are not sequential.
const (
	VarIntWireType          WireType = 0
	Fixed64WireType                  = 1
	LengthDelimitedWireType          = 2
	Fixed32WireType                  = 5
)
