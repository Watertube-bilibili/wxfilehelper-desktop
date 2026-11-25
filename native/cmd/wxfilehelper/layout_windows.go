// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/desktop"
)

type caption struct {
	text             string
	box              rect
	font, ink, flags uintptr
}

const singleLine = 0x20 | 0x4 | 0x8000 // DT_SINGLELINE | DT_VCENTER | DT_END_ELLIPSIS

func (a *window) box(x, y, w, h int) rect {
	return rect{int32(a.px(x)), int32(a.px(y)), int32(a.px(x + w)), int32(a.px(y + h))}
}
func (a *window) caption(s string, x, y, w, h int, font, c, flags uintptr) {
	a.labels = append(a.labels, caption{s, a.box(x, y, w, h), font, c, flags})
}
func (a *window) layout() {
	if !a.ready {
		return
	}
	var r rect
	user32.NewProc("GetClientRect").Call(a.h, uintptr(unsafe.Pointer(&r)))
	a.w = int(r.Right) * 96 / a.dpi
	a.hgt = int(r.Bottom) * 96 / a.dpi
	w, h := a.w, a.hgt
	a.rail = w >= 900 && h >= 492 && !a.room
	railW := 0
	if a.rail {
		railW = 192
	}
	planned := map[int]rect{}
	put := func(id, x, y, width, height int) { planned[id] = a.box(x, y, max(1, width), max(1, height)) }
	a.labels = nil
	margin := 20
	if w < 680 {
		margin = 12
	}
	if a.room {
		margin = 28
	}
	left, right := railW+margin, w-margin
	headerWidth := right - left
	btnH := 34
	titleH := 30
	headerY := 12
	tabsY := 72
	if h < 500 {
		tabsY = 53
		headerY = 4
		titleH = 25
	}
	if a.room {
		btnH = 46
		titleH = 36
		tabsY = 86
		headerY = 11
	}
	menuW := 42
	classW := 98
	if a.room {
		classW = 183
		menuW = 54
	}
	put(idMenu, right-menuW, headerY+2, menuW, btnH)
	put(idClass, right-menuW-classW-6, headerY+2, classW, btnH)
	a.caption(a.tr("文件传输助手", "File Transfer"), left, headerY, headerWidth-classW-menuW-14, titleH, a.titleFont, ink, singleLine)
	stateInk := muted
	if a.account.Error != "" {
		stateInk = color(149, 48, 39)
	} else if a.account.SignedIn {
		stateInk = blue
	}
	a.caption(a.stateLabel(), left, headerY+titleH, headerWidth, max(20, tabsY-headerY-titleH-3), a.font, stateInk, singleLine)
	tabW := 110
	if a.room {
		tabW = 155
	}
	tabH := 29
	if a.room {
		tabH = 42
	}
	n := len(a.snap.Accounts)
	send.Call(a.children[idTabs], 0x1329, 0, uintptr(a.px(tabW))|(uintptr(a.px(tabH-3))<<16))
	put(idTabs, left, tabsY, tabW*n+6, tabH+2)
	if n < 2 {
		put(idAdd, left+tabW*n+12, tabsY, btnH, tabH)
	}
	top := tabsY + tabH + 12
	if a.snap.Update != nil {
		put(idNotice, left, top, min(380, headerWidth), btnH)
		top += btnH + 6
	}
	a.contentLeft, a.contentRight, a.contentTop = left, right, top
	if a.rail {
		a.caption("WX FileHelper", 18, 72, railW-34, 30, a.brandFont, ink, singleLine)
		a.caption("v"+desktop.Version, 18, 100, railW-34, 20, a.font, muted, singleLine)
		put(idChat, 12, 148, railW-24, 38)
		put(idFiles, 12, 191, railW-24, 38)
		put(idDownloads, 12, h-136, railW-24, 36)
		put(idUpdate, 12, h-94, railW-24, 36)
		put(idAbout, 12, h-52, railW-24, 36)
	}
	if width := right - left; width > 1130 {
		pad := (width - 1130) / 2
		left += pad
		right -= pad
	}
	width := right - left
	a.qrBounds = rect{}
	if !a.account.SignedIn {
		center := (left + right) / 2
		maxSize := 246
		if a.room {
			maxSize = 430
		}
		btnW := 146
		hintH := 24
		headingH := 32
		if a.room {
			btnW = 220
			hintH = 36
			headingH = 46
		}
		bottom := h - 12
		gaps := 20
		size := max(24, min(maxSize, width-20, bottom-top-headingH-hintH-btnH-gaps))
		group := headingH + size + hintH + btnH + gaps
		start := top + max(0, (bottom-top-group)/2)
		a.caption(a.tr("扫码登录", "Scan to sign in"), left, start, width, headingH, a.titleFont, ink, singleLine|1)
		y := start + headingH + 4
		a.qrBounds = a.box(center-size/2, y, size, size)
		instruction := a.tr("打开手机微信，使用「扫一扫」。", "Open WeChat on your phone and scan this code.")
		if a.account.Status == "scanned" {
			instruction = a.tr("扫码成功，请在手机上确认登录。", "Code scanned. Confirm sign-in on your phone.")
		}
		a.caption(instruction, left, y+size+4, width, hintH, a.font, muted, singleLine|1)
		put(idRefresh, center-btnW/2, y+size+hintH+12, btnW, btnH)
	} else {
		bottom := h - 12
		rowY := bottom - btnH
		sendW := 95
		fileW := 120
		stopW := 120
		if a.room {
			sendW = 137
			fileW = 168
			stopW = 170
		}
		put(idChoose, left, rowY, fileW, btnH)
		put(idSend, right-sendW, rowY, sendW, btnH)
		if a.account.Busy {
			put(idStop, left+fileW+6, rowY, stopW, btnH)
		}
		editH := max(32, min(88, h/12))
		if a.room {
			editH = max(52, min(118, h/9))
		}
		editY := rowY - editH - 6
		put(idEditor, left, editY, width, editH)
		cueHeight := 21
		if a.room {
			cueHeight = 30
		}
		a.caption(a.tr("消息  ·  Ctrl+Enter 发送", "Message  ·  Ctrl+Enter to send"), left, editY-cueHeight, width, cueHeight, a.smallFont, muted, singleLine)
		mid := editY - cueHeight - 4
		if len(a.account.Files) > 0 {
			queueH := max(28, min(h/11, (btnH-5)*min(3, len(a.account.Files))+4))
			if h < 480 {
				queueH = 27
			}
			put(idQueue, left, mid-queueH, width, queueH)
			mid -= queueH
			qTitleH := btnH - 4
			a.caption(fmt.Sprintf(a.tr("待发送  ·  %d 个文件", "Queued  ·  %d files"), len(a.account.Files)), left, mid-qTitleH, width-120, qTitleH, a.brandFont, ink, singleLine)
			put(idRemoveFile, right-91, mid-qTitleH, 91, qTitleH-1)
			mid -= qTitleH + 4
		}
		barH := btnH - 2
		historyY := top + barH + 3
		title := a.tr("对话", "Conversation")
		if a.filesOnly {
			title = a.tr("本次会话文件", "Session files")
		}
		toolsW := 152
		if a.room {
			toolsW = 218
		}
		a.caption(title, left, top, width-toolsW-6, barH, a.brandFont, ink, singleLine)
		put(idCopy, right-toolsW, top, toolsW*4/10-4, barH)
		put(idSave, right-toolsW*6/10, top, toolsW*6/10, barH)
		put(idHistory, left, historyY, width, max(22, mid-historyY))
	}
	for id, child := range a.children {
		next, visible := planned[id]
		old, was := a.placements[id]
		if !visible {
			if was {
				showWindow.Call(child, 0)
			}
			continue
		}
		if !was || old != next {
			moveWindow.Call(child, uintptr(next.Left), uintptr(next.Top), uintptr(next.Right-next.Left), uintptr(next.Bottom-next.Top), 1)
		}
		if !was {
			showWindow.Call(child, 5)
		}
	}
	a.placements = planned
}

func (a *window) paint() {
	var ps paintStruct
	dc, _, _ := user32.NewProc("BeginPaint").Call(a.h, uintptr(unsafe.Pointer(&ps)))
	defer user32.NewProc("EndPaint").Call(a.h, uintptr(unsafe.Pointer(&ps)))
	a.drawSurface(dc, ps.Paint)
}

func (a *window) drawSurface(dc uintptr, bounds rect) {
	fill(dc, bounds, base)
	if a.rail {
		fill(dc, a.box(0, 0, 192, a.hgt), railBG)
		fill(dc, a.box(191, 0, 1, a.hgt), border)
		user32.NewProc("DrawIconEx").Call(dc, uintptr(a.px(18)), uintptr(a.px(19)), a.icon, uintptr(a.px(43)), uintptr(a.px(43)), 0, 0, 3)
	}
	headerLeft := 0
	if a.rail {
		headerLeft = 192
	}
	fill(dc, a.box(headerLeft, 0, a.w-headerLeft, a.contentTop-4), white)
	fill(dc, a.box(headerLeft, a.contentTop-4, a.w-headerLeft, 1), border)
	for _, label := range a.labels {
		textAt(dc, label.text, label.box, label.font, label.ink, label.flags)
	}
	if r := a.qrBounds; r.Right > r.Left {
		fill(dc, r, border)
		r.Left++
		r.Top++
		r.Right--
		r.Bottom--
		fill(dc, r, white)
		if len(a.qr.pixels) > 0 {
			a.qr.draw(dc, r, a.px(6))
		} else {
			s := a.tr("正在获取二维码…", "Requesting code…")
			if a.account.Status == "idle" || a.account.Status == "expired" || a.account.Error != "" {
				s = a.tr("请刷新二维码", "Refresh to get a code")
			}
			textAt(dc, s, r, a.font, muted, singleLine|1)
		}
	}
	if a.account.Busy && a.account.Total > 0 {
		x := a.contentLeft
		maxW := a.contentRight - x
		progress := min(100, max(0, int(a.account.Sent*100/a.account.Total)))
		fill(dc, a.box(x, a.contentTop-4, maxW, 3), paleBlue)
		fill(dc, a.box(x, a.contentTop-4, maxW*progress/100, 3), blue)
	}
}
func (q *qrBitmap) draw(dc uintptr, r rect, pad int) {
	type bitmapInfo struct {
		Size                   uint32
		Width, Height          int32
		Planes, BitCount       uint16
		Compression, ImageSize uint32
		XPels, YPels           int32
		Used, Important        uint32
	}
	header := bitmapInfo{Size: 40, Width: int32(q.width), Height: -int32(q.height), Planes: 1, BitCount: 32}
	availW, availH := int(r.Right-r.Left)-2*pad, int(r.Bottom-r.Top)-2*pad
	if availW <= 0 || availH <= 0 {
		return
	}
	w, h := availW, availW*q.height/q.width
	if h > availH {
		h = availH
		w = h * q.width / q.height
	}
	x, y := int(r.Left)+(int(r.Right-r.Left)-w)/2, int(r.Top)+(int(r.Bottom-r.Top)-h)/2
	gdi32.NewProc("SetStretchBltMode").Call(dc, 3) // COLORONCOLOR keeps QR module edges sharp
	gdi32.NewProc("StretchDIBits").Call(dc, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0, 0, uintptr(q.width), uintptr(q.height),
		uintptr(unsafe.Pointer(&q.pixels[0])), uintptr(unsafe.Pointer(&header)), 0, 0x00cc0020)
}
func (a *window) drawButton(d *drawItem) {
	id := int(d.ID)
	r := d.Bounds
	disabled := d.State&4 != 0
	pressed := d.State&1 != 0
	focus := d.State&0x10 != 0
	isRail := id == idChat || id == idFiles || id == idDownloads || id == idUpdate || id == idAbout
	bg, fg, edge := white, ink, border
	selected := (id == idChat && !a.filesOnly) || (id == idFiles && a.filesOnly)
	if isRail {
		bg, edge = railBG, railBG
	}
	if selected || id == idNotice {
		bg, fg, edge = paleBlue, blue, paleBlue
	}
	if id == idSend || id == idRefresh {
		bg, fg, edge = blue, white, blue
	}
	if pressed {
		bg = color(209, 225, 240)
		fg = ink
	}
	if disabled {
		bg = color(240, 243, 246)
		fg = color(110, 123, 138)
		edge = border
	}
	fill(d.DC, r, edge)
	r.Left++
	r.Top++
	r.Right--
	r.Bottom--
	fill(d.DC, r, bg)
	flags := uintptr(singleLine | 1)
	if isRail {
		flags = singleLine
		r.Left += int32(a.px(12))
		if selected {
			fill(d.DC, a.box(0, 7, 3, 23), blue)
		}
	}
	textAt(d.DC, getText(d.Window), r, a.font, fg, flags)
	if focus {
		r = d.Bounds
		r.Left += 3
		r.Top += 3
		r.Right -= 3
		r.Bottom -= 3
		user32.NewProc("DrawFocusRect").Call(d.DC, uintptr(unsafe.Pointer(&r)))
	}
}

func (a *window) drawListRow(d *drawItem) {
	r := d.Bounds
	selected := d.State&1 != 0
	bg := white
	if selected {
		bg = paleBlue
	}
	fill(d.DC, r, bg)
	if d.Item == 0xffffffff {
		return
	}
	index := int(d.Item)
	row := r
	row.Left += int32(a.px(11))
	row.Right -= int32(a.px(10))
	if d.ID == idQueue {
		if index >= len(a.account.Files) {
			return
		}
		file := a.account.Files[index]
		name := row
		name.Right -= int32(a.px(110))
		textAt(d.DC, file.Name, name, a.font, ink, singleLine)
		textAt(d.DC, fileSize(file.Size), row, a.smallFont, muted, singleLine|2)
	} else if index < len(a.messageIDs) {
		for _, m := range a.account.Messages {
			if m.ID != a.messageIDs[index] {
				continue
			}
			who := a.tr("来自手机", "From phone")
			if m.Outgoing {
				who = a.tr("此电脑", "This PC")
			}
			metadata := who + "   ·   " + time.Unix(m.Timestamp, 0).Format("15:04")
			if isAttachment(m) && m.Size > 0 {
				metadata += "   ·   " + fileSize(m.Size)
			}
			meta := row
			meta.Top += int32(a.px(2))
			meta.Bottom = r.Top + (r.Bottom-r.Top)*46/100
			tone := muted
			if m.Outgoing {
				tone = blue
			}
			textAt(d.DC, metadata, meta, a.smallFont, tone, singleLine)
			body := row
			body.Top = meta.Bottom
			body.Bottom -= int32(a.px(3))
			textAt(d.DC, strings.Join(strings.Fields(m.Text), " "), body, a.font, ink, singleLine)
			break
		}
		fill(d.DC, rect{row.Left, r.Bottom - 1, row.Right, r.Bottom}, railBG)
	}
	if d.State&0x10 != 0 {
		user32.NewProc("DrawFocusRect").Call(d.DC, uintptr(unsafe.Pointer(&r)))
	}
}
