package glh

import (
	"bytes"
	"errors"

	"github.com/vktec/gll"
)

func NewShader(gl gll.GL200, shaderType uint32, shaderSources ...string) (shad uint32, err error) {
	shad = gl.CreateShader(shaderType)
	csrc, clen, free := gll.Strs(shaderSources...)
	gl.ShaderSource(shad, int32(len(shaderSources)), csrc, clen)
	free()
	gl.CompileShader(shad)

	var result int32
	gl.GetShaderiv(shad, gll.COMPILE_STATUS, &result)
	if result == 0 {
		var bufSize, errLen int32
		gl.GetShaderiv(shad, gll.INFO_LOG_LENGTH, &bufSize)
		buf := make([]byte, bufSize)
		gl.GetShaderInfoLog(shad, bufSize, &errLen, &buf[0])
		gl.DeleteShader(shad)
		return 0, errors.New(string(bytes.TrimRight(buf[:errLen], " \t\r\n")))
	}
	return shad, nil
}

func LinkProgram(gl gll.GL200, shaders ...uint32) (prog uint32, err error) {
	prog = gl.CreateProgram()
	for _, shad := range shaders {
		gl.AttachShader(prog, shad)
	}
	gl.LinkProgram(prog)
	for _, shad := range shaders {
		gl.DetachShader(prog, shad)
	}

	var result int32
	gl.GetProgramiv(prog, gll.LINK_STATUS, &result)
	if result == 0 {
		var bufSize, errLen int32
		gl.GetProgramiv(prog, gll.INFO_LOG_LENGTH, &bufSize)
		if bufSize > 0 {
			buf := make([]byte, bufSize)
			gl.GetProgramInfoLog(prog, bufSize, &errLen, &buf[0])
			gl.DeleteProgram(prog)
			return 0, errors.New(string(bytes.TrimRight(buf[:errLen], " \t\r\n")))
		} else {
			gl.DeleteProgram(prog)
			return 0, errors.New("No message provided")
		}
	}
	return prog, nil
}
