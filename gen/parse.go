package gen

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

type xRegistry struct {
	Types      []xType      `xml:"types>type"`
	Enums      []xEnum      `xml:"enums>enum"`
	Commands   []xCommand   `xml:"commands>command"`
	Features   []xFeature   `xml:"feature"`
	Extensions []xExtension `xml:"extensions>extension"`
}
type xType struct {
	Tdef  string  `xml:",chardata"`
	Name  xString `xml:"name"`
	NameA string  `xml:"name,attr"`
}
type xEnum struct {
	API   string `xml:"api,attr"`
	Name  string `xml:"name,attr"`
	Type  string `xml:"group,attr"`
	Value string `xml:"value,attr"`
}
type xCommand struct {
	Proto  xParam   `xml:"proto"`
	Params []xParam `xml:"param"`
}
type xParam struct {
	Name  xString `xml:"name"`
	Group string  `xml:"group,attr"`
	Raw   []byte  `xml:",innerxml"`
}
type xFeature struct {
	API      string     `xml:"api,attr"`
	Number   float64    `xml:"number,attr"`
	Commands []xFeatCmd `xml:"require>command"`
}
type xFeatCmd struct {
	Name string `xml:"name,attr"`
}
type xExtension struct {
	Name      string     `xml:"name,attr"`
	Supported string     `xml:"supported,attr"`
	Commands  []xFeatCmd `xml:"require>command"`
}
type xString struct {
	S string `xml:",chardata"`
}

func (par xParam) Type() (string, error) {
	r := bytes.NewReader(par.Raw)
	b := strings.Builder{}
	dec := xml.NewDecoder(r)
loop:
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "name" {
				break loop
			}
		case xml.CharData:
			b.Write([]byte(tok))
		}
	}

	ty := b.String()
	ty = strings.ReplaceAll(ty, "const", "")
	ty = strings.Trim(ty, " \t\n")
	return ty, nil
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
		Types:      make(map[string]Type, len(xreg.Types)),
		Enums:      make([]Enum, 0, len(xreg.Enums)),
		Commands:   make([]Command, len(xreg.Commands)),
		Features:   make([]Feature, 0, len(xreg.Features)),
		Extensions: make([]Extension, 0, len(xreg.Extensions)),
	}

	for _, xty := range xreg.Types {
		name := xty.NameA
		ty := InvalidType
		switch name {
		case "khrplatform":
			continue // Ignore khrplatform
		case "GLhandleARB":
			ty = GLhandleARB
		case "":
			name = xty.Name.S
			switch name {
			case "GLvoid":
				continue // Ignore void, it's unused
			case "GLeglClientBufferEXT", "GLeglImageOES", "struct _cl_context", "struct _cl_event", "GLVULKANPROCNV":
				ty = UnsupportedType
			case "GLDEBUGPROC", "GLDEBUGPROCARB", "GLDEBUGPROCKHR", "GLDEBUGPROCAMD":
				ty = GLDEBUGPROC
			case "GLboolean":
				ty = Bool
			case "GLhandleARB":
				ty = GLhandleARB
			case "GLsync":
				ty = GLsync
			default:
				ty = parseTypeDef(xty.Tdef)
			}
		}
		if ty == InvalidType {
			return nil, fmt.Errorf("Cannot parse typedef for %s: %q", name, xty.Tdef)
		}
		reg.Types[name] = ty
	}

	for _, xenum := range xreg.Enums {
		if xenum.API == "" || xenum.API == "gl" {
			reg.Enums = append(reg.Enums, Enum{xenum.Name, xenum.Type, xenum.Value})
		}
	}

	for i, xcmd := range xreg.Commands {
		ty, err := xcmd.Proto.Type()
		if err != nil {
			return nil, err
		}
		cmd := Command{
			xcmd.Proto.Name.S,
			make([]Param, len(xcmd.Params)),
			ty,
		}
		for j, xpar := range xcmd.Params {
			ty, err := xpar.Type()
			if err != nil {
				return nil, err
			}
			cmd.Params[j] = Param{xpar.Name.S, ty}
		}
		reg.Commands[i] = cmd
	}

	for _, xfeat := range xreg.Features {
		if xfeat.API != "gl" {
			continue
		}

		feat := Feature{
			int(xfeat.Number*100 + 0.5),
			make([]string, len(xfeat.Commands)),
		}
		for i, cmd := range xfeat.Commands {
			feat.Commands[i] = cmd.Name
		}
		reg.Features = append(reg.Features, feat)
	}

	for _, xext := range xreg.Extensions {
		support := strings.Split(xext.Supported, "|")
		glSupported := false
		for _, s := range support {
			if s == "gl" {
				glSupported = true
				break
			}
		}
		if !glSupported {
			continue
		}

		ext := Extension{make([]string, len(xext.Commands))}
		for i, cmd := range xext.Commands {
			ext.Commands[i] = cmd.Name
		}
		reg.Extensions = append(reg.Extensions, ext)
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
