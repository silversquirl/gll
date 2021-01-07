package gll

// #include <stdint.h>
// #include <stdlib.h>
import "C"
import (
	"reflect"
	"unsafe"
)

func mkslice(data uintptr, cap_ int) unsafe.Pointer {
	return unsafe.Pointer(&reflect.SliceHeader{Data: data, Len: cap_, Cap: cap_})
}

func Strs(strs ...string) (strp **uint8, lenp *int32, free func()) {
	var strpLen, lenpLen, strbLen uintptr
	for _, s := range strs {
		strpLen += unsafe.Sizeof((*uint8)(nil))
		lenpLen += unsafe.Sizeof(int32(0))
		strbLen += uintptr(len(s) + 1)
	}
	mem := C.malloc(C.size_t(strpLen + lenpLen + strbLen))
	p := uintptr(mem)

	strS := *(*[]*uint8)(mkslice(p, len(strs)))
	lenS := *(*[]int32)(mkslice(p+strpLen, len(strs)))
	strb := *(*[]byte)(mkslice(p+strpLen+lenpLen, int(strbLen)))

	nbyte := 0
	for i, s := range strs {
		copy(strb[nbyte:], []byte(s))
		strS[i] = &strb[nbyte]
		lenS[i] = int32(len(s))
		nbyte += len(s) + 1 // + 1 for NUL terminator
	}

	return &strS[0], &lenS[0], func() {
		C.free(mem)
	}
}

func Str(str string) *uint8 {
	if str[len(str)-1] != 0 {
		panic("Argument to Str is not NUL-terminated")
	}
	hdr := *(*reflect.StringHeader)(unsafe.Pointer(&str))
	return (*uint8)(unsafe.Pointer(hdr.Data))
}

func Ptr(value interface{}) unsafe.Pointer {
	v := reflect.ValueOf(value)
	switch v.Type().Kind() {
	case reflect.Ptr, reflect.UnsafePointer, reflect.Slice:
		return unsafe.Pointer(v.Pointer())
	case reflect.Array:
		return unsafe.Pointer(v.Index(0).UnsafeAddr())
	default:
		panic("Unsupported type; must be pointer, unsafe.Pointer, slice or array")
	}
}

func Offset(offset int) unsafe.Pointer {
	return unsafe.Pointer(uintptr(offset))
}
