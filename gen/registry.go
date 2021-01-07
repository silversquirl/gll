package gen

type Registry struct {
	Types    map[string]Type
	Enums    []Enum
	Commands []Command
	Features []Feature
}

type Type int

const (
	InvalidType Type = iota
	UnsupportedType

	Int8
	Int16
	Int32
	Int64
	Intptr
	Intsize

	Uint8
	Uint16
	Uint32
	Uint64
	Uintptr
	Uintsize

	Float32
	Float64

	Bool
	Pointer

	GLhandleARB
	GLsync

	// TODO: GLDEBUGPROC, _cl_context, _cl_event, GLVULKANPROCNV
)

type Enum struct {
	Name  string
	Type  string
	Value string
}

type Command struct {
	Name   string
	Params []Param
	Return string
}
type Param struct {
	Name string
	Type string
}

type Feature struct {
	Version  int
	Commands []string
}
