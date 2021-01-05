package gleg

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type xRegistry struct {
	Types    []xType    `xml:"types>type"`
	Enums    []xEnum    `xml:"enums>enum"`
	Commands []xCommand `xml:"commands>command"`
}
type xType struct {
	Tdef  string  `xml:",chardata"`
	Name  xString `xml:"name"`
	NameA string  `xml:"name,attr"`
}
type xEnum struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"group,attr"`
	Value string `xml:"value,attr"`
}
type xCommand struct {
	Proto  xProto   `xml:"proto"`
	Params []xParam `xml:"param"`
}
type xProto struct {
	Ret  string  `xml:",chardata"`
	Name xString `xml:"name"`
}
type xParam struct {
	Name  xString `xml:"name"`
	Group string  `xml:"group,attr"`
	Type  xString `xml:"ptype"`
}
type xString struct {
	S string `xml:",chardata"`
}

func Parse(r io.Reader) (*Registry, error) {
	// Parse XML
	dec := xml.NewDecoder(r)
	var xreg xRegistry
	if err := dec.Decode(&xreg); err != nil {
		return nil, err
	}

	// Convert registry
	reg := &Registry{
		Types:    make(map[string]Type, len(xreg.Types)),
		Enums:    make([]Enum, len(xreg.Enums)),
		Commands: make([]Command, len(xreg.Commands)),
	}

	for _, xty := range xreg.Types {
		name := xty.NameA
		ty := InvalidType
		switch name {
		case "khrplatform":
			continue // Skip khrplatform
		case "GLhandleARB":
			ty = GLhandleARB
		case "":
			name = xty.Name.S
			switch name {
			case "GLvoid", "GLeglClientBufferEXT", "GLeglImageOES", "struct _cl_context", "struct _cl_event", "GLVULKANPROCNV":
				continue
			case "GLhandleARB":
				ty = GLhandleARB
			case "GLsync":
				ty = GLsync
			case "GLDEBUGPROC", "GLDEBUGPROCARB", "GLDEBUGPROCKHR", "GLDEBUGPROCAMD":
				ty = GLdebugProc
			default:
				ty = parseTypeDef(xty.Tdef)
			}
		}
		if ty == InvalidType {
			return nil, fmt.Errorf("Cannot parse typedef for %s: %q", name, xty.Tdef)
		}
		reg.Types[name] = ty
	}

	for i, xenum := range xreg.Enums {
		reg.Enums[i] = Enum{xenum.Name, xenum.Type, xenum.Value}
	}

	for i, xcmd := range xreg.Commands {
		cmd := Command{
			xcmd.Proto.Name.S,
			make([]Param, len(xcmd.Params)),
			strings.Trim(xcmd.Proto.Ret, " \t\n"),
		}
		for j, xpar := range xcmd.Params {
			cmd.Params[j] = Param{xpar.Name.S, xpar.Type.S}
		}
		reg.Commands[i] = cmd
	}

	return reg, nil
}

func parseTypeDef(tdef string) Type {
	if !strings.HasPrefix(tdef, "typedef ") || !strings.HasSuffix(tdef, " ;") {
		return InvalidType
	}
	tdef = tdef[len("typedef ") : len(tdef)-len(" ;")]

	switch tdef {
	case "khronos_int8_t":
		return Int8
	case "khronos_int16_t":
		return Int16
	case "khronos_int32_t", "int":
		return Int32
	case "khronos_int64_t":
		return Int64
	case "khronos_intptr_t", "GLintptr":
		return Intptr
	case "khronos_ssize_t":
		return Intsize

	case "khronos_uint8_t", "char", "unsigned char":
		return Uint8
	case "khronos_uint16_t", "unsigned short":
		return Uint16
	case "khronos_uint32_t", "unsigned int":
		return Uint32
	case "khronos_uint64_t":
		return Uint64
	case "khronos_uintptr_t":
		return Uintptr
	case "khronos_size_t":
		return Uintsize

	case "khronos_float_t":
		return Float32
	case "double":
		return Float64

	default:
		return InvalidType
	}
}
