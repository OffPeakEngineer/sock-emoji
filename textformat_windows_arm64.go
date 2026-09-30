package main

import (
	"runtime"
	"syscall"
	"unsafe"
)

func textFormatTrampolineAddr() uintptr

func createTextFormat(factory *comObject, family, locale *uint16, size uint32, format **comObject) uintptr {
	r, _, _ := syscall.SyscallN(textFormatTrampolineAddr(), factory.vtable[15],
		uintptr(unsafe.Pointer(factory)), uintptr(unsafe.Pointer(family)), 0, 400, 0, 5,
		uintptr(size), uintptr(unsafe.Pointer(locale)), uintptr(unsafe.Pointer(format)))
	runtime.KeepAlive(factory)
	return r
}
