package gleg

type Registry struct {
	Types    map[string]Type
	Enums    []Enum
	Commands []Command
}

type Type int

const (
	InvalidType Type = iota

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

	Pointer

	GLhandleARB
	GLsync
	GLdebugProc

	// TODO: _cl_context, _cl_event, GLVULKANPROCNV
)

type Enum struct {
	Name  string
	Type  string
	Value int
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
