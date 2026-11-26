// SPDX-License-Identifier: AGPL-3.0-only
package main

import "unsafe"

type toolInfo struct {
	Size, Flags     uint32
	Owner, ID       uintptr
	Bounds          rect
	Instance        uintptr
	Text            *uint16
	Param, Reserved uintptr
}

func (a *window) createTooltips() {
	h, _, _ := createWindow.Call(0, uintptr(unsafe.Pointer(wide("tooltips_class32"))), 0, 0x80000003, 0, 0, 0, 0, a.h, 0, a.instance, 0)
	a.tooltip = h
}

func (a *window) labelTooltips() {
	if a.tooltip == 0 || (a.tipStrings != nil && a.tipsEnglish == a.english) {
		return
	}
	first := a.tipStrings == nil
	if first {
		a.tipStrings = map[int]*uint16{}
	}
	a.tipsEnglish = a.english
	for _, id := range []int{idAdd, idMenu, idChat, idFiles, idTransfers, idNavWeChat, idNavNearby, idNavLocal, idDownloads} {
		t := toolInfo{Flags: 0x11, Owner: a.h, ID: a.children[id], Text: wide(getText(a.children[id]))}
		t.Size = uint32(unsafe.Sizeof(t))
		msg := uintptr(0x439) // TTM_UPDATETIPTEXTW
		if first {
			msg = 0x432
		} // TTM_ADDTOOLW
		send.Call(a.tooltip, msg, 0, uintptr(unsafe.Pointer(&t)))
		a.tipStrings[id] = t.Text
	}
}

func (a *window) updateEditorScroll() {
	h := a.children[idEditor]
	if _, visible := a.placements[idEditor]; h == 0 || !visible {
		return
	}
	var bounds rect
	send.Call(h, 0xb2, 0, uintptr(unsafe.Pointer(&bounds)))
	count, _, _ := send.Call(h, 0xba, 0, 0) // EM_GETLINECOUNT
	height := max(1, a.px(20))
	show := int(count) > max(1, int(bounds.Bottom-bounds.Top)/height)
	style, _, _ := user32.NewProc("GetWindowLongW").Call(h, ^uintptr(15)) // GWL_STYLE
	if (style&0x200000 != 0) != show {
		on := uintptr(0)
		if show {
			on = 1
		}
		user32.NewProc("ShowScrollBar").Call(h, 1, on)
	}
}
