// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"fmt"
	"unsafe"
)

type caption struct {
	text             string
	box              rect
	font, ink, flags uintptr
}

const (
	singleLine = 0x20 | 0x4 | 0x8000 // DT_SINGLELINE | DT_VCENTER | DT_END_ELLIPSIS
	railWidth  = 196
)

func (a *window) box(x, y, w, h int) rect {
	return rect{int32(a.px(x)), int32(a.px(y)), int32(a.px(x + w)), int32(a.px(y + h))}
}

func (a *window) caption(s string, x, y, w, h int, font, c, flags uintptr) {
	a.labels = append(a.labels, caption{s, a.box(x, y, w, h), font, c, flags})
}

type positionControl func(id, x, y, width, height int)

func (a *window) layout() {
	if !a.ready {
		return
	}
	var r rect
	user32.NewProc("GetClientRect").Call(a.h, uintptr(unsafe.Pointer(&r)))
	a.w, a.hgt = int(r.Right)*96/a.dpi, int(r.Bottom)*96/a.dpi
	w, h := a.w, a.hgt
	historyRow := 70
	if h < 505 {
		historyRow = 61
	}
	if a.historyRow != historyRow {
		a.historyRow = historyRow
		send.Call(a.children[idHistory], 0x1a0, 0, uintptr(a.px(historyRow)))
	}
	a.rail = w >= 880 && h >= 490
	edge := 0
	if a.rail {
		edge = railWidth
	}
	planned := map[int]rect{}
	put := func(id, x, y, width, height int) { planned[id] = a.box(x, y, max(1, width), max(1, height)) }
	a.labels = nil
	a.qrBounds, a.qrCard, a.composer, a.queueSurface, a.statusDot = rect{}, rect{}, rect{}, rect{}, rect{}

	margin, btnH := 24, 36
	y, titleH, tabsY, tabH := 16, 32, 81, 33
	menuW := 38
	if w < 690 {
		margin = 16
	}
	if h < 505 {
		y, titleH, tabsY, tabH = 8, 27, 62, 32
		btnH = 30
	}
	left, right := edge+margin, w-margin
	put(idMenu, right-menuW, y+2, menuW, btnH)

	a.caption(a.tr("文件传输助手", "File Transfer"), left, y, right-left-menuW-12, titleH, a.titleFont, ink, singleLine)
	statusInk := muted
	if a.account.Error != "" {
		statusInk = color(155, 45, 43)
	} else if a.account.Busy || a.account.Status == "scan" || a.account.Status == "scanned" {
		statusInk = blue
	} else if a.account.Status == "online" {
		statusInk = color(25, 105, 72)
	}
	sy := y + titleH + 3
	a.statusDot = a.box(left+1, sy+7, 7, 7)
	a.statusInk = statusInk
	a.caption(a.stateLabel(), left+16, sy, right-left-16, 22, a.smallFont, statusInk, singleLine)

	tabW := 144
	if w < 690 {
		tabW = 124
	}
	n := len(a.snap.Accounts)
	send.Call(a.children[idTabs], 0x1329, 0, uintptr(a.px(tabW))|(uintptr(a.px(tabH-4))<<16))
	put(idTabs, left, tabsY, tabW*n+26, tabH)
	if n < 2 {
		put(idAdd, left+tabW*n+31, tabsY, tabH, tabH-2)
	}
	a.headerHeight = tabsY + tabH + 9

	if a.rail {
		a.caption("WX FileHelper", 20, 76, railWidth-38, 29, a.brandFont, ink, singleLine)
		put(idChat, 12, 129, railWidth-24, 40)
		put(idFiles, 12, 173, railWidth-24, 40)
		put(idDownloads, 12, h-135, railWidth-24, 36)
		put(idUpdate, 12, h-94, railWidth-24, 36)
		put(idAbout, 12, h-53, railWidth-24, 36)
	}
	panelWidth := 1120
	if width := right - left; width > panelWidth {
		inset := (width - panelWidth) / 2
		left, right = left+inset, right-inset
	}
	top := a.headerHeight + 12
	if a.snap.Update != nil {
		if h < 505 {
			put(idNotice, w-margin-154, tabsY, 154, tabH-2)
		} else {
			put(idNotice, left, top, min(360, right-left), btnH)
			top += btnH + 8
		}
	}
	a.contentLeft, a.contentRight, a.contentTop = left, right, top
	if a.account.SignedIn {
		a.layoutTransfer(put, left, right, top, btnH)
	} else {
		a.layoutLogin(put, left, right, top, btnH)
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

func (a *window) layoutLogin(put positionControl, left, right, top, btnH int) {
	width := right - left
	compact := a.hgt < 530
	heading := a.tr("用微信扫码登录", "Scan with WeChat")
	hint := a.tr("用手机微信「扫一扫」，在手机上确认登录。", "Open WeChat’s scanner, then confirm sign-in on your phone.")
	if a.account.Status == "scanned" {
		heading = a.tr("请在手机上确认", "Confirm on your phone")
		hint = a.tr("手机已识别二维码，请在微信中确认登录。", "WeChat found the code. Approve sign-in to continue.")
	}
	if width >= 665 && !compact {
		gap := 36
		size := min(284, a.hgt-top-72)
		cardWidth := size + 24
		textWidth := min(330, width-cardWidth-gap-24)
		x := left + (width-cardWidth-gap-textWidth)/2
		y := top + max(12, (a.hgt-top-cardWidth-24)/2)
		a.qrCard = a.box(x, y, cardWidth, cardWidth)
		a.qrBounds = a.box(x+12, y+12, size, size)
		tx := x + cardWidth + gap
		ty := y + (cardWidth-136)/2
		a.caption(heading, tx, ty, textWidth, 38, a.titleFont, ink, singleLine)
		a.caption(hint, tx, ty+46, textWidth, 48, a.font, muted, 0x10)
		put(idRefresh, tx, ty+108, max(152, btnH*3), btnH)
		return
	}
	center := (left + right) / 2
	headingH, hintH := 34, 27
	size := max(60, min(246, width-32, a.hgt-top-headingH-hintH-btnH-50))
	block := headingH + size + hintH + btnH + 34
	y := top + max(0, (a.hgt-top-block)/2)
	a.caption(heading, left, y, width, headingH, a.titleFont, ink, singleLine|1)
	qy := y + headingH + 7
	a.qrCard = a.box(center-size/2-7, qy-5, size+14, size+10)
	a.qrBounds = a.box(center-size/2, qy, size, size)
	a.caption(hint, left, qy+size+8, width, hintH, a.smallFont, muted, singleLine|1)
	put(idRefresh, center-82, qy+size+hintH+14, 164, btnH)
}

func (a *window) layoutTransfer(put positionControl, left, right, top, btnH int) {
	width := right - left
	inset, gap := 14, 8
	if a.hgt < 505 {
		inset, gap = 10, 6
	}
	innerLeft, innerRight := left+inset, right-inset
	bottom := a.hgt - 14
	y := bottom - inset - btnH
	sendW, fileW, stopW, saveW, copyW := 104, 124, 111, 100, 72
	put(idChoose, innerLeft, y, fileW, btnH)
	put(idSend, innerRight-sendW, y, sendW, btnH)
	if a.account.Busy {
		put(idStop, innerLeft+fileW+gap, y, stopW, btnH)
	}
	editH := max(31, min(61, a.hgt/11))
	if a.hgt < 505 {
		editH = 31
	}
	editY := y - editH - gap
	put(idEditor, innerLeft+1, editY, innerRight-innerLeft-2, editH)
	labelH := 24
	if a.hgt < 505 {
		labelH = 22
	}
	labelY := editY - labelH
	a.caption(a.tr("消息", "Message"), innerLeft, labelY, width/2, labelH, a.brandFont, ink, singleLine)
	a.caption(a.tr("Ctrl+Enter 发送", "Ctrl+Enter to send"), innerLeft, labelY, innerRight-innerLeft-2, labelH, a.smallFont, muted, singleLine|2)
	composerTop := labelY - 5
	if n := len(a.account.Files); n > 0 {
		queueH := queueRowHeight * min(n, 2)
		if a.hgt < 505 {
			queueH = queueRowHeight
		}
		listY := labelY - queueH - 8
		a.queueSurface = a.box(innerLeft-2, listY-2, innerRight-innerLeft+4, queueH+4)
		put(idQueue, innerLeft+1, listY, innerRight-innerLeft-2, queueH)
		barH := btnH - 4
		barY := listY - barH - 5
		a.caption(fmt.Sprintf(a.tr("待发送 · %d 项", "Queued · %d"), n), innerLeft, barY, innerRight-innerLeft-100, barH, a.smallFont, muted, singleLine)
		put(idRemoveFile, innerRight-88, barY, 88, barH)
		composerTop = min(composerTop, barY-5)
	}
	a.composer = a.box(left, composerTop, width, bottom-composerTop)

	title := a.tr("会话消息", "Conversation")
	if a.filesOnly {
		title = a.tr("会话中的文件", "Session files")
	}
	a.caption(title, left+2, top, width-saveW-copyW-16, btnH, a.brandFont, ink, singleLine)
	put(idCopy, right-copyW-saveW-7, top, copyW, btnH)
	put(idSave, right-saveW, top, saveW, btnH)
	historyY := top + btnH + 5
	historyH := max(20, composerTop-historyY-8)
	put(idHistory, left, historyY, width, historyH)
}

func (a *window) paint() {
	var ps paintStruct
	dc, _, _ := user32.NewProc("BeginPaint").Call(a.h, uintptr(unsafe.Pointer(&ps)))
	defer user32.NewProc("EndPaint").Call(a.h, uintptr(unsafe.Pointer(&ps)))
	a.drawSurface(dc, ps.Paint)
}

func (a *window) drawSurface(dc uintptr, bounds rect) {
	fill(dc, bounds, base)
	edge := 0
	if a.rail {
		edge = railWidth
		fill(dc, a.box(0, 0, edge, a.hgt), railBG)
		fill(dc, a.box(edge-1, 0, 1, a.hgt), border)
		fill(dc, a.box(18, a.hgt-147, edge-36, 1), border)
		user32.NewProc("DrawIconEx").Call(dc, uintptr(a.px(20)), uintptr(a.px(22)), a.icon, uintptr(a.px(44)), uintptr(a.px(44)), 0, 0, 3)
	}
	fill(dc, a.box(edge, 0, a.w-edge, a.headerHeight), white)
	fill(dc, a.box(edge, a.headerHeight-1, a.w-edge, 1), border)
	if a.statusDot.Right > a.statusDot.Left {
		roundedSurface(dc, a.statusDot, a.px(8), a.statusInk, a.statusInk)
	}
	if a.composer.Right > a.composer.Left {
		outline := border
		if focused, _, _ := user32.NewProc("GetFocus").Call(); focused == a.children[idEditor] {
			outline = blue
		}
		roundedSurface(dc, a.composer, a.px(12), white, outline)
	}
	if a.queueSurface.Right > a.queueSurface.Left {
		roundedSurface(dc, a.queueSurface, a.px(7), base, border)
	}
	for _, label := range a.labels {
		textAt(dc, label.text, label.box, label.font, label.ink, label.flags)
	}
	if r := a.qrBounds; r.Right > r.Left {
		roundedSurface(dc, a.qrCard, a.px(14), white, border)
		if len(a.qr.pixels) > 0 {
			a.qr.draw(dc, r, a.px(5))
		} else {
			text := a.tr("正在获取二维码…", "Requesting code…")
			if a.account.Status == "idle" || a.account.Status == "expired" || a.account.Error != "" {
				text = a.tr("等待获取二维码", "Waiting for a code")
			}
			textAt(dc, text, r, a.smallFont, muted, singleLine|1)
		}
	}
	if a.account.Busy && a.account.Total > 0 {
		width := a.contentRight - a.contentLeft
		progress := min(100, max(0, int(a.account.Sent*100/a.account.Total)))
		fill(dc, a.box(a.contentLeft, a.headerHeight-3, width, 2), paleBlue)
		fill(dc, a.box(a.contentLeft, a.headerHeight-3, width*progress/100, 2), blue)
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
	gdi32.NewProc("SetStretchBltMode").Call(dc, 3) // COLORONCOLOR keeps QR module edges sharp.
	gdi32.NewProc("StretchDIBits").Call(dc, uintptr(x), uintptr(y), uintptr(w), uintptr(h), 0, 0, uintptr(q.width), uintptr(q.height),
		uintptr(unsafe.Pointer(&q.pixels[0])), uintptr(unsafe.Pointer(&header)), 0, 0x00cc0020)
}
