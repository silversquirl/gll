package gll

/*
#include <stdint.h>
void gllCall_glDebugMessageCallback(void *_func, uintptr_t userParam);
*/
import "C"
import (
	"runtime"
	"sync"
)

type debugState struct {
	debugProc func(source, type_, id, severity uint32, message string)
	debugIdx  int
}

func (gl *lib) DebugMessageCallback(callback func(source, type_, id, severity uint32, message string)) {
	if gl.debugProc == nil {
		gl.debugIdx = debugAdd(gl)
		runtime.SetFinalizer(gl, func(gl *lib) {
			debugDel(gl.debugIdx)
		})
	}
	gl.debugProc = callback
	C.gllCall_glDebugMessageCallback(gl.glDebugMessageCallback, C.uintptr_t(gl.debugIdx))
}

func debugGet(idx int) *lib {
	debugLock.RLock()
	defer debugLock.RUnlock()
	return debugLibs[idx]
}
func debugAdd(gl *lib) int {
	debugLock.Lock()
	defer debugLock.Unlock()
	if len(debugFree) > 0 {
		// Grab a free slot
		idx := debugFree[len(debugFree)-1]
		debugFree = debugFree[:len(debugFree)-1]
		return idx
	} else {
		// Create a new slot
		debugLibs = append(debugLibs, gl)
		return len(debugLibs) - 1
	}
}
func debugDel(idx int) {
	debugLock.Lock()
	defer debugLock.Unlock()
	debugLibs[idx] = nil
	debugFree = append(debugFree, idx)
}

var debugLibs []*lib
var debugFree []int
var debugLock sync.RWMutex

//export gllDebugProc
func gllDebugProc(source, type_, id, severity, length C.uint32_t, message *C.char, userParam C.uintptr_t) {
	gl := debugGet(int(userParam))
	gl.debugProc(uint32(source), uint32(type_), uint32(id), uint32(severity), C.GoStringN(message, C.int(length)))
}
