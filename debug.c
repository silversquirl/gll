#include <stdint.h>
#include "_cgo_export.h"

// This prototype is different from the one defined by OpenGL, but it should be compatible on all relevant systems
typedef void (*DEBUGPROC)(uint32_t source, uint32_t type, uint32_t id, uint32_t severity, uint32_t length, char *message, uintptr_t userParam);
void gllCall_glDebugMessageCallback(void *_func, uintptr_t userParam) {
	((void (*)(DEBUGPROC, void*))_func)(gllDebugProc, (void *)userParam);
}
