// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"syscall"
	"unsafe"
)

var tabCallback uintptr

func init() { tabCallback = syscall.NewCallback(tabProc) }

// Paint the entire tab strip so the native tab's beveled page frame does not
// extend beyond our buttons. Hit testing, navigation, focus and the UI Automation
// provider remain those of SysTabControl32.
func tabProc(h uintptr, msg uint32, w, l, sub, ref uintptr) uintptr {
	switch msg {
	case 0x14: // WM_ERASEBKGND
		return 1
	case 0x0f: // WM_PAINT
		var ps paintStruct
		dc, _, _ := user32.NewProc("BeginPaint").Call(h, uintptr(unsafe.Pointer(&ps)))
		paintTabStrip(h, dc)
		user32.NewProc("EndPaint").Call(h, uintptr(unsafe.Pointer(&ps)))
		return 0
	case 0x317, 0x318: // WM_PRINT, WM_PRINTCLIENT
		paintTabStrip(h, w)
		return 0
	case 0x82: // WM_NCDESTROY
		comctl32.NewProc("RemoveWindowSubclass").Call(h, tabCallback, sub)
	}
	result, _, _ := comctl32.NewProc("DefSubclassProc").Call(h, uintptr(msg), w, l)
	return result
}

func paintTabStrip(h, dc uintptr) {
	var client rect
	user32.NewProc("GetClientRect").Call(h, uintptr(unsafe.Pointer(&client)))
	fill(dc, client, tabBG)
	if ui == nil || !ui.ready {
		return
	}
	count, _, _ := send.Call(h, 0x1304, 0, 0)
	focused, _, _ := user32.NewProc("GetFocus").Call()
	cursor, _, _ := send.Call(h, 0x132f, 0, 0) // TCM_GETCURFOCUS
	state, _, _ := send.Call(h, 0x129, 0, 0)   // WM_QUERYUISTATE
	for i := uintptr(0); i < count; i++ {
		var b rect
		if ok, _, _ := send.Call(h, 0x130a, i, uintptr(unsafe.Pointer(&b))); ok == 0 {
			continue
		}
		// Use the native horizontal geometry, also used for keyboard/mouse hit tests.
		b.Top = client.Top + int32(ui.px(1))
		b.Bottom = client.Bottom - int32(ui.px(1))
		item := drawItem{ID: idTabs, Item: uint32(i), Window: h, DC: dc, Bounds: b}
		if focused == h && cursor == i && state&1 == 0 {
			item.State = 0x10
		}
		ui.drawTab(&item)
	}
	ui.tabUnderline(dc, h, client)
}
