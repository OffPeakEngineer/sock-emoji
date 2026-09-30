package main

import (
	"fmt"
	"unsafe"

	"github.com/lxn/win"
)

const iconSize = 32

func renderIcon(value string) (win.HICON, error) {
	pixels, err := renderEmojiPixels(value)
	if err != nil {
		return 0, err
	}
	header := win.BITMAPINFOHEADER{BiWidth: iconSize, BiHeight: -iconSize, BiPlanes: 1, BiBitCount: 32, BiCompression: win.BI_RGB}
	header.BiSize = uint32(unsafe.Sizeof(header))
	var bits unsafe.Pointer
	bitmap := win.CreateDIBSection(0, &header, win.DIB_RGB_COLORS, &bits, 0, 0)
	if bitmap == 0 {
		return 0, fmt.Errorf("create icon bitmap failed")
	}
	defer win.DeleteObject(win.HGDIOBJ(bitmap))
	// Direct2D and Windows alpha icons both use premultiplied BGRA pixels.
	copy(unsafe.Slice((*byte)(bits), len(pixels)), pixels)
	mask := win.CreateBitmap(iconSize, iconSize, 1, 1, nil)
	if mask == 0 {
		return 0, fmt.Errorf("create icon mask failed")
	}
	defer win.DeleteObject(win.HGDIOBJ(mask))
	info := win.ICONINFO{FIcon: 1, HbmMask: mask, HbmColor: bitmap}
	icon := win.CreateIconIndirect(&info)
	if icon == 0 {
		return 0, fmt.Errorf("create tray icon failed")
	}
	return icon, nil
}
