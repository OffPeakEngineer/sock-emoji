package main

import (
	"fmt"
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
	"golang.org/x/sys/windows"
)

var (
	d2dCreateFactory    = windows.NewLazySystemDLL("d2d1.dll").NewProc("D2D1CreateFactory")
	dwriteCreateFactory = windows.NewLazySystemDLL("dwrite.dll").NewProc("DWriteCreateFactory")
	coCreateInstance    = windows.NewLazySystemDLL("ole32.dll").NewProc("CoCreateInstance")
	iidD2DFactory       = windows.GUID{Data1: 0x06152247, Data2: 0x6f50, Data3: 0x465a, Data4: [8]byte{0x92, 0x45, 0x11, 0x8b, 0xfd, 0x3b, 0x60, 0x07}}
	iidDWriteFactory    = windows.GUID{Data1: 0xb859ee5a, Data2: 0xd838, Data3: 0x4b5b, Data4: [8]byte{0xa2, 0xe8, 0x1a, 0xdc, 0x7d, 0x93, 0xdb, 0x48}}
	clsidWICFactory     = windows.GUID{Data1: 0xcacaf262, Data2: 0x9370, Data3: 0x4615, Data4: [8]byte{0xa1, 0x3b, 0x9f, 0x55, 0x39, 0xda, 0x4c, 0x0a}}
	iidWICFactory       = windows.GUID{Data1: 0xec5ec8a9, Data2: 0xc395, Data3: 0x4314, Data4: [8]byte{0x9c, 0x77, 0x54, 0xd7, 0xa9, 0x35, 0xff, 0x70}}
	wicPBGRA            = windows.GUID{Data1: 0x6fddc324, Data2: 0x4e03, Data3: 0x4bfe, Data4: [8]byte{0xb1, 0x85, 0x3d, 0x77, 0x76, 0x8d, 0xc9, 0x10}}
)

// The small COM surface used here is defined by d2d1.h, dwrite.h and
// wincodec.h in the Windows SDK. Slots include inherited IUnknown methods.
// Keeping these bindings here avoids requiring a C compiler or a UI framework.
type comObject struct{ vtable *[64]uintptr }

//go:uintptrescapes
func comCall(object *comObject, slot int, args ...uintptr) uintptr {
	callArgs := append([]uintptr{uintptr(unsafe.Pointer(object))}, args...)
	result, _, _ := syscall.SyscallN(object.vtable[slot], callArgs...)
	runtime.KeepAlive(object)
	return result
}

func (object *comObject) release() { comCall(object, 2) }

func checkHRESULT(operation string, result uintptr) error {
	if int32(result) < 0 {
		return fmt.Errorf("%s: HRESULT 0x%08x", operation, uint32(result))
	}
	return nil
}

type renderTargetProperties struct {
	Type, Format, AlphaMode uint32
	DpiX, DpiY              float32
	Usage, MinLevel         uint32
}

// renderEmojiPixels uses Segoe UI Emoji's color layers and DirectWrite's
// shaping, including ZWJ/skin-tone sequences. The software WIC target works
// without a GPU or visible window. All COM objects stay on one OS thread.
func renderEmojiPixels(value string) ([]byte, error) {
	text, err := syscall.UTF16FromString(value)
	if err != nil {
		return nil, err
	}
	if len(text) == 1 {
		return nil, fmt.Errorf("cannot render empty emoji")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr := win.CoInitializeEx(nil, win.COINIT_APARTMENTTHREADED)
	if hr >= 0 {
		defer win.CoUninitialize()
	} else if uint32(hr) != 0x80010106 { // RPC_E_CHANGED_MODE: already initialized as MTA.
		return nil, checkHRESULT("initialize COM", uintptr(hr))
	}
	var wic *comObject
	r, _, _ := coCreateInstance.Call(uintptr(unsafe.Pointer(&clsidWICFactory)), 0, 1,
		uintptr(unsafe.Pointer(&iidWICFactory)), uintptr(unsafe.Pointer(&wic)))
	if err := checkHRESULT("create WIC factory", r); err != nil {
		return nil, err
	}
	defer wic.release()
	var bitmap *comObject
	// IWICImagingFactory::CreateBitmap; WICBitmapCacheOnLoad = 2.
	r = comCall(wic, 17, iconSize, iconSize, uintptr(unsafe.Pointer(&wicPBGRA)), 2, uintptr(unsafe.Pointer(&bitmap)))
	if err := checkHRESULT("create WIC bitmap", r); err != nil {
		return nil, err
	}
	defer bitmap.release()
	var factory *comObject
	r, _, _ = d2dCreateFactory.Call(0, uintptr(unsafe.Pointer(&iidD2DFactory)), 0, uintptr(unsafe.Pointer(&factory)))
	if err := checkHRESULT("create Direct2D factory", r); err != nil {
		return nil, err
	}
	defer factory.release()
	// D2D1_RENDER_TARGET_TYPE_SOFTWARE, DXGI_FORMAT_B8G8R8A8_UNORM,
	// D2D1_ALPHA_MODE_PREMULTIPLIED. Fix DPI so the output is always iconSize.
	properties := renderTargetProperties{Type: 1, Format: 87, AlphaMode: 1, DpiX: 96, DpiY: 96}
	var target *comObject
	r = comCall(factory, 13, uintptr(unsafe.Pointer(bitmap)), uintptr(unsafe.Pointer(&properties)), uintptr(unsafe.Pointer(&target)))
	if err := checkHRESULT("create WIC render target", r); err != nil {
		return nil, err
	}
	defer target.release()
	var dwrite *comObject
	r, _, _ = dwriteCreateFactory.Call(0, uintptr(unsafe.Pointer(&iidDWriteFactory)), uintptr(unsafe.Pointer(&dwrite)))
	if err := checkHRESULT("create DirectWrite factory", r); err != nil {
		return nil, err
	}
	defer dwrite.release()
	family := syscall.StringToUTF16Ptr("Segoe UI Emoji")
	locale := syscall.StringToUTF16Ptr("en-us")
	var format *comObject
	// IDWriteFactory::CreateTextFormat. Float arguments use a different ABI on
	// ARM64; the architecture-specific shim supplies fontSize in the FP register.
	r = createTextFormat(dwrite, family, locale, math.Float32bits(26), &format)
	if err := checkHRESULT("create emoji text format", r); err != nil {
		return nil, err
	}
	defer format.release()
	for _, setting := range []struct {
		slot  int
		value uintptr
	}{
		{3, 2}, // SetTextAlignment(CENTER)
		{4, 1}, // SetParagraphAlignment(CENTER)
		{5, 1}, // SetWordWrapping(NO_WRAP)
	} {
		if err := checkHRESULT("configure emoji layout", comCall(format, setting.slot, setting.value)); err != nil {
			return nil, err
		}
	}
	var brush *comObject
	white := [4]float32{1, 1, 1, 1}
	r = comCall(target, 8, uintptr(unsafe.Pointer(&white)), 0, uintptr(unsafe.Pointer(&brush)))
	if err := checkHRESULT("create text brush", r); err != nil {
		return nil, err
	}
	defer brush.release()
	comCall(target, 34, 2) // SetTextAntialiasMode(GRAYSCALE) for transparent pixels.
	comCall(target, 48)    // BeginDraw
	transparent := [4]float32{}
	comCall(target, 47, uintptr(unsafe.Pointer(&transparent))) // Clear
	rect := [4]float32{0, 0, iconSize, iconSize}
	// ID2D1RenderTarget::DrawText, ENABLE_COLOR_FONT = 4.
	comCall(target, 27, uintptr(unsafe.Pointer(&text[0])), uintptr(len(text)-1), uintptr(unsafe.Pointer(format)),
		uintptr(unsafe.Pointer(&rect)), uintptr(unsafe.Pointer(brush)), 4, 0)
	if err := checkHRESULT("draw color emoji", comCall(target, 49, 0, 0)); err != nil {
		return nil, err
	} // EndDraw
	pixels := make([]byte, iconSize*iconSize*4)
	// IWICBitmapSource::CopyPixels returns premultiplied BGRA, top-down.
	r = comCall(bitmap, 7, 0, iconSize*4, uintptr(len(pixels)), uintptr(unsafe.Pointer(&pixels[0])))
	if err := checkHRESULT("copy emoji pixels", r); err != nil {
		return nil, err
	}
	return pixels, nil
}
