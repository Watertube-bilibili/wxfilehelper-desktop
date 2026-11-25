// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	gdi32          = syscall.NewLazyDLL("gdi32.dll")
	shell32        = syscall.NewLazyDLL("shell32.dll")
	comdlg32       = syscall.NewLazyDLL("comdlg32.dll")
	comctl32       = syscall.NewLazyDLL("comctl32.dll")
	createWindow   = user32.NewProc("CreateWindowExW")
	defWindowProc  = user32.NewProc("DefWindowProcW")
	send           = user32.NewProc("SendMessageW")
	showWindow     = user32.NewProc("ShowWindow")
	moveWindow     = user32.NewProc("MoveWindow")
	enableWindow   = user32.NewProc("EnableWindow")
	setWindowText  = user32.NewProc("SetWindowTextW")
	invalidateRect = user32.NewProc("InvalidateRect")
	deleteObject   = gdi32.NewProc("DeleteObject")
	selectObject   = gdi32.NewProc("SelectObject")
	setTextColor   = gdi32.NewProc("SetTextColor")
	setBkMode      = gdi32.NewProc("SetBkMode")
	drawText       = user32.NewProc("DrawTextW")
)

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }
type message struct {
	Window  uintptr
	ID      uint32
	W, L    uintptr
	Time    uint32
	Point   point
	Private uint32
}
type windowClass struct {
	Size, Style                        uint32
	Proc                               uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background uintptr
	Menu, Name                         *uint16
	IconSmall                          uintptr
}
type paintStruct struct {
	DC                 uintptr
	Erase              int32
	Paint              rect
	Restore, IncUpdate int32
	Reserved           [32]byte
}
type minMaxInfo struct{ Reserved, MaxSize, MaxPosition, MinTrack, MaxTrack point }
type drawItem struct {
	Type, ID, Item, Action, State uint32
	Window, DC                    uintptr
	Bounds                        rect
	Data                          uintptr
}
type notification struct {
	Window, ID uintptr
	Code       int32
}
type tabItem struct {
	Mask, State, MaskState uint32
	Text                   *uint16
	TextMax, Image         int32
	Param                  uintptr
}

// Colorref values are stored in BGR order by GDI.
func color(r, g, b byte) uintptr { return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16 }
func wide(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(strings.ReplaceAll(s, "\x00", "�"))
	return p
}
func setText(h uintptr, s string) { setWindowText.Call(h, uintptr(unsafe.Pointer(wide(s)))) }
func getText(h uintptr) string {
	n, _, _ := user32.NewProc("GetWindowTextLengthW").Call(h)
	buf := make([]uint16, n+1)
	user32.NewProc("GetWindowTextW").Call(h, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}
func redraw(h uintptr) { invalidateRect.Call(h, 0, 1) }
func enabled(h uintptr, yes bool) {
	var v uintptr
	if yes {
		v = 1
	}
	enableWindow.Call(h, v)
}
func fill(dc uintptr, r rect, c uintptr) {
	if r.Right <= r.Left || r.Bottom <= r.Top {
		return
	}
	brush, _, _ := gdi32.NewProc("CreateSolidBrush").Call(c)
	user32.NewProc("FillRect").Call(dc, uintptr(unsafe.Pointer(&r)), brush)
	deleteObject.Call(brush)
}
func textAt(dc uintptr, s string, r rect, font, ink, flags uintptr) {
	old, _, _ := selectObject.Call(dc, font)
	setTextColor.Call(dc, ink)
	setBkMode.Call(dc, 1)
	drawText.Call(dc, uintptr(unsafe.Pointer(wide(s))), ^uintptr(0), uintptr(unsafe.Pointer(&r)), flags|0x800) // DT_NOPREFIX
	selectObject.Call(dc, old)
}
func createFont(size, weight int, dpi int) uintptr {
	h, _, _ := gdi32.NewProc("CreateFontW").Call(uintptr(-size*dpi/96), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(wide("Segoe UI"))))
	return h
}
func getDPI(h uintptr) int {
	g := user32.NewProc("GetDpiForWindow")
	if g.Find() == nil {
		if n, _, _ := g.Call(h); n != 0 {
			return int(n)
		}
	}
	dc, _, _ := user32.NewProc("GetDC").Call(h)
	n, _, _ := gdi32.NewProc("GetDeviceCaps").Call(dc, 88)
	user32.NewProc("ReleaseDC").Call(h, dc)
	if n == 0 {
		return 96
	}
	return int(n)
}
func setDPIAwareness() {
	// The manifest supplies the default. These fallbacks also cover unmanifested
	// development builds and Windows 10 versions before per-monitor v2.
	p := user32.NewProc("SetProcessDpiAwarenessContext")
	if p.Find() == nil {
		if ok, _, _ := p.Call(^uintptr(3)); ok != 0 {
			return
		}
	}
	q := syscall.NewLazyDLL("shcore.dll").NewProc("SetProcessDpiAwareness")
	if q.Find() == nil {
		q.Call(2)
		return
	}
	user32.NewProc("SetProcessDPIAware").Call()
}
func dialog(h uintptr, s, title string, flags uintptr) uintptr {
	r, _, _ := user32.NewProc("MessageBoxW").Call(h, uintptr(unsafe.Pointer(wide(s))), uintptr(unsafe.Pointer(wide(title))), flags)
	return r
}
