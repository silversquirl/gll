package gleg

import (
	"os"
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	f, err := os.Open("testdata/test.xml")
	if err != nil {
		t.Fatal(err)
	}
	reg, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}

	expected := &Registry{
		Types: map[string]Type{
			"GLenum":     Uint32,
			"GLbitfield": Uint32,
		},
		Enums: []Enum{
			{"GL_CLIENT_PIXEL_STORE_BIT", "ClientAttribMask", "0x00000001"},
			{"GL_CLIENT_VERTEX_ARRAY_BIT", "ClientAttribMask", "0x00000002"},
			{"GL_CLIENT_ALL_ATTRIB_BITS", "ClientAttribMask", "0xFFFFFFFF"},
		},
		Commands: []Command{
			{"glClientAttribDefaultEXT", []Param{
				{"mask", "GLbitfield"},
			}, "void"},
		},
	}

	if !reflect.DeepEqual(expected.Types, reg.Types) {
		t.Errorf("Types do not match:\n\t%q\n\t%q", expected.Types, reg.Types)
	}
	if !reflect.DeepEqual(expected.Enums, reg.Enums) {
		t.Errorf("Enums do not match:\n\t%q\n\t%q", expected.Enums, reg.Enums)
	}
	if !reflect.DeepEqual(expected.Commands, reg.Commands) {
		t.Errorf("Commands do not match:\n\t%q\n\t%q", expected.Commands, reg.Commands)
	}
}
