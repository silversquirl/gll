package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"sort"
	"strings"
)

func Generate(reg *Registry) (src []byte, err error) {
	buf := bytes.Buffer{}
	buf.WriteString("package gll\n\n")

	buf.WriteString("/*\n#cgo LDFLAGS: -lGL\n")
	genC(&buf, reg)
	buf.WriteString("*/\n")

	buf.WriteString("import \"C\"\n\n")
	buf.WriteString("import \"unsafe\"\n\n")
	cmdSigs := genLib(&buf, reg)
	genVersions(&buf, cmdSigs, reg)
	genExtensions(&buf, cmdSigs, reg)
	genTypes(&buf)
	genEnums(&buf, reg)

	return format.Source(buf.Bytes())
}

func genC(buf *bytes.Buffer, reg *Registry) {
	buf.WriteString("#include <stdint.h>\n")
	buf.WriteString("#include <sys/types.h>\n")

	buf.WriteString(`
#ifdef __APPLE__
typedef void *GLhandleARB;
#else
typedef unsigned int GLhandleARB;
#endif
typedef struct __GLsync *GLsync;

`)

commands:
	for _, cmd := range reg.Commands {
		if strings.HasPrefix(cmd.Name, "glDebugMessageCallback") {
			// We don't generate any wrapper code for this function, it's manually defined in debug.c
			continue
		}

		params := make([]string, len(cmd.Params))
		args := make([]string, len(cmd.Params))
		for i, par := range cmd.Params {
			ty, ok := cType(reg.Types, par.Type, false)
			if !ok {
				continue commands
			}
			params[i] = ty + par.Name
			args[i] = par.Name
		}
		paramS := strings.Join(params, ", ")
		argS := strings.Join(args, ", ")

		retTy, ok := cType(reg.Types, cmd.Return, false)
		if !ok {
			continue
		}

		comma := ""
		if paramS != "" {
			comma = ", "
		}
		fmt.Fprintf(buf, "%sgllCall_%s(void *_func%s%s) {\n", retTy, cmd.Name, comma, paramS)
		buf.WriteByte('\t')
		if cmd.Return != "void" {
			buf.WriteString("return ")
		}
		fmt.Fprintf(buf, "((%s(*)(%s))_func)(%s);\n", retTy, paramS, argS)
		buf.WriteString("}\n")
	}
}

func genLib(buf *bytes.Buffer, reg *Registry) (cmdSigs map[string]string) {
	cmdSigs = make(map[string]string, len(reg.Commands))
	names := make([]string, 0, len(reg.Commands))
commands:
	for _, cmd := range reg.Commands {
		if strings.HasPrefix(cmd.Name, "glDebugMessageCallback") {
			// We don't generate any wrapper code for this function, it's manually defined in debug.go
			names = append(names, cmd.Name)
			cmdSigs[cmd.Name] = "(callback func(source, type_, id, severity uint32, message string))"
			continue
		}

		params := make([]string, len(cmd.Params))
		args := make([]string, len(cmd.Params))
		for i, par := range cmd.Params {
			ty, ok := goType(reg.Types, par.Type)
			if !ok {
				continue commands
			}
			cty, _ := cType(reg.Types, par.Type, true)

			name := par.Name
			switch name {
			case "func", "type", "range", "map":
				name += "_"
			}

			params[i] = name + ty
			if cty[0] == '*' {
				args[i] = fmt.Sprintf("(%s)(unsafe.Pointer(%s))", cty, name)
			} else {
				args[i] = fmt.Sprintf("(%s)(%s)", cty, name)
			}
		}
		paramS := strings.Join(params, ", ")
		argS := strings.Join(args, ", ")

		retTy, ok := goType(reg.Types, cmd.Return)
		if !ok {
			continue
		}

		names = append(names, cmd.Name)
		cmdSigs[cmd.Name] = fmt.Sprintf("(%s)%s", paramS, retTy)
		fmt.Fprintf(buf, "func (gl *lib) %s(%s)%s {\n", strings.TrimPrefix(cmd.Name, "gl"), paramS, retTy)
		cast := false
		if retTy != "" {
			cast = true
			fmt.Fprintf(buf, "return (%s)(", retTy)
		}
		fmt.Fprintf(buf, "C.gllCall_%s(gl.%s, %s)", cmd.Name, cmd.Name, argS)
		if cast {
			buf.WriteByte(')')
		}
		buf.WriteString("\n}\n")
	}

	buf.WriteString("type lib struct {\n")
	buf.WriteString("debugState\n")
	for _, name := range names {
		buf.WriteString(name)
		buf.WriteString(" unsafe.Pointer\n")
	}
	buf.WriteString("}\n")

	return cmdSigs
}

func genVersions(buf *bytes.Buffer, cmdSigs map[string]string, reg *Registry) {
	cmds := make(map[string]struct{})
	// FIXME: This is a selection sort. Selection sort is trash, but as of writing there are only 20 OpenGL versions, so it's fine
	// Also the elements are sorted in the Khronos registry and this selection sort is optimized for that case
	for v, found := 0, true; found; {
		found = false
		for _, feat := range reg.Features {
			// TODO: support <remove> and profiles
			if feat.Version > v {
				found = true
				v = feat.Version
				for _, cmd := range feat.Commands {
					if _, ok := cmdSigs[cmd]; ok {
						cmds[cmd] = struct{}{}
					}
				}
				genVersion(buf, cmdSigs, v, cmds)
			}
		}
	}
}
func genVersion(buf *bytes.Buffer, cmdSigs map[string]string, v int, cmdMap map[string]struct{}) {
	cmds := make([]string, 0, len(cmdMap))
	for cmd := range cmdMap {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)

	fmt.Fprintf(buf, "type GL%d interface {\nExtensions\n", v)
	for _, cmd := range cmds {
		buf.WriteString(strings.TrimPrefix(cmd, "gl"))
		buf.WriteString(cmdSigs[cmd])
		buf.WriteByte('\n')
	}
	fmt.Fprintf(buf, "}\nfunc New%d(getProcAddr func(name string) unsafe.Pointer) GL%[1]d {\n", v)
	buf.WriteString("return &lib{\n")
	for _, cmd := range cmds {
		fmt.Fprintf(buf, "%s: getProcAddr(%[1]q),\n", cmd)
	}
	buf.WriteString("}\n}\n")
}

func genExtensions(buf *bytes.Buffer, cmdSigs map[string]string, reg *Registry) {
	cmds := make(map[string]struct{})
	buf.WriteString("type Extensions interface {\n")
	for _, ext := range reg.Extensions {
		for _, cmd := range ext.Commands {
			if strings.HasPrefix(cmd, "glDebugMessageCallback") { // TODO: support debug extensions
				continue
			}

			if _, ok := cmds[cmd]; ok {
				continue
			}
			cmds[cmd] = struct{}{}
			if sig, ok := cmdSigs[cmd]; ok {
				buf.WriteString(strings.TrimPrefix(cmd, "gl"))
				buf.WriteString(sig)
				buf.WriteByte('\n')
			}
		}
	}
	buf.WriteString("}\n")
}

func genTypes(buf *bytes.Buffer) {
	buf.WriteString(`
type GLhandleARB C.GLhandleARB
type GLsync C.GLsync
`)
}

func genEnums(buf *bytes.Buffer, reg *Registry) {
	buf.WriteString("const (\n")
	for _, enum := range reg.Enums {
		name := strings.TrimPrefix(enum.Name, "GL_")
		// A few names start with digits, don't remove the GL_ prefix for those
		if '0' <= name[0] && name[0] <= '9' {
			name = enum.Name
		}
		fmt.Fprintf(buf, "%s = %s\n", name, enum.Value)
	}
	buf.WriteString(")\n")
}

func ptrParse(ty string) (name, ptr string) {
	name = strings.TrimRight(ty, " \t\n")
	for {
		name = strings.TrimRight(name, " \t\n")
		if !strings.HasSuffix(name, "*") {
			return
		}
		name = name[:len(name)-1]
		ptr += "*"
	}
}

func cType(types map[string]Type, name string, cgo bool) (t string, ok bool) {
	name, ptr := ptrParse(name)
	if name == "void" {
		if cgo {
			return ptr[1:] + "unsafe.Pointer", true
		} else {
			return "void " + ptr, true
		}
	}

	cgoPrefix := true
	ty := types[name]
	switch ty {
	case InvalidType:
		panic(fmt.Sprintf("Invalid type: %q", name))
	case UnsupportedType:
		return "", false

	case Int8:
		t = "int8_t"
	case Int16:
		t = "int16_t"
	case Int32:
		t = "int32_t"
	case Int64:
		t = "int64_t"
	case Intptr:
		t = "intptr_t"
	case Intsize:
		t = "ssize_t"

	case Uint8:
		t = "uint8_t"
	case Uint16:
		t = "uint16_t"
	case Uint32:
		t = "uint32_t"
	case Uint64:
		t = "uint64_t"
	case Uintptr:
		t = "uintptr_t"
	case Uintsize:
		t = "size_t"

	case Float32:
		t = "float"
	case Float64:
		t = "double"

	case Bool:
		t = "_Bool"
	case Pointer:
		if cgo {
			cgoPrefix = false
			t = "unsafe.Pointer"
		} else {
			ptr += "*"
			t = "void"
		}

	case GLhandleARB:
		t = "GLhandleARB"
	case GLsync:
		t = "GLsync"
	case GLDEBUGPROC:
		panic("GLDEBUGPROC has no C representation")

	default:
		panic(fmt.Sprintf("Unknown type for %q: %d", name, ty))
	}
	if cgo {
		if cgoPrefix {
			t = "C." + t
		}
		return ptr + t, true
	} else {
		return t + " " + ptr, true
	}
}

func goType(types map[string]Type, name string) (t string, ok bool) {
	name, ptr := ptrParse(name)
	if name == "void" {
		if ptr == "" {
			return "", true
		} else {
			return " " + ptr[1:] + "unsafe.Pointer", true
		}
	}

	ty := types[name]
	switch ty {
	case InvalidType:
		panic(fmt.Sprintf("Invalid type: %q", name))
	case UnsupportedType:
		return "", false

	case Int8:
		t = "int8"
	case Int16:
		t = "int16"
	case Int32:
		t = "int32"
	case Int64:
		t = "int64"
	case Intptr:
		t = "uintptr"
	case Intsize:
		t = "int"

	case Uint8:
		t = "uint8"
	case Uint16:
		t = "uint16"
	case Uint32:
		t = "uint32"
	case Uint64:
		t = "uint64"
	case Uintptr:
		t = "uintptr"
	case Uintsize:
		t = "uint"

	case Float32:
		t = "float32"
	case Float64:
		t = "float64"

	case Bool:
		t = "bool"
	case Pointer:
		t = "unsafe.Pointer"

	case GLhandleARB:
		t = "GLhandleARB"
	case GLsync:
		t = "GLsync"
	case GLDEBUGPROC:
		t = "func(source, type_, id, severity uint32, message string)"

	default:
		panic(fmt.Sprintf("Unknown type for %q: %d", name, ty))
	}
	return " " + ptr + t, true
}
