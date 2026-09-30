//go:build windows && !arm64

package main

import "unsafe"

func createTextFormat(factory *comObject, family, locale *uint16, size uint32, format **comObject) uintptr {
	// On AMD64 (and 386), this float32 argument is passed on the stack.
	return comCall(factory, 15, uintptr(unsafe.Pointer(family)), 0, 400, 0, 5,
		uintptr(size), uintptr(unsafe.Pointer(locale)), uintptr(unsafe.Pointer(format)))
}
