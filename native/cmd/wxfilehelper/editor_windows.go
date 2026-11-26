// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"syscall"
	"unsafe"
)

var editorCallback uintptr

func init() { editorCallback = syscall.NewCallback(editorProc) }

// Win32's built-in cue banner is available only on single-line edits.
// Let the multiline edit paint and handle IME, selection and input itself.
func editorProc(h uintptr, msg uint32, w, l, sub, ref uintptr) uintptr {
	if msg == 0x82 {
		comctl32.NewProc("RemoveWindowSubclass").Call(h, editorCallback, sub)
	}
	result, _, _ := comctl32.NewProc("DefSubclassProc").Call(h, uintptr(msg), w, l)
	if msg != 0x0f && msg != 0x317 && msg != 0x318 {
		return result
	}
	if ui == nil || !ui.ready {
		return result
	}
	if n, _, _ := user32.NewProc("GetWindowTextLengthW").Call(h); n != 0 {
		return result
	}
	dc := w
	if msg == 0x0f {
		dc, _, _ = user32.NewProc("GetDC").Call(h)
		defer user32.NewProc("ReleaseDC").Call(h, dc)
	}
	if dc != 0 {
		var r rect
		send.Call(h, 0xb2, 0, uintptr(unsafe.Pointer(&r))) // EM_GETRECT
		r.Left += int32(ui.px(2))
		textAt(dc, ui.tr("输入消息…", "Type a message…"), r, ui.font, muted, 0x20|0x8000)
	}
	return result
}
