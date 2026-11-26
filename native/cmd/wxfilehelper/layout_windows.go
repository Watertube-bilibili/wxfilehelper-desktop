// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"fmt"
	"math"
	"time"
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
	a.rail = w >= 880 && h >= 490
	dest := 52
	if a.rail {
		dest = railWidth
	}
	a.animate(&a.motion.rail, float64(dest), 180*time.Millisecond)
	edge := a.sidebarWidth()
	planned := map[int]rect{}
	put := func(id, x, y, width, height int) { planned[id] = a.box(x, y, max(1, width), max(1, height)) }
	a.labels = nil
	a.directHero = rect{}
	a.directCards = nil
	a.directProgress = rect{}
	a.qrBounds, a.qrCard, a.composer, a.queueSurface, a.statusDot = rect{}, rect{}, rect{}, rect{}, rect{}
	t := a.tab(a.snap.Active)
	margin, buttonH, stripH, titleY := 24, 34, 46, 58
	if w < 760 {
		margin = 14
	}
	if h < 505 {
		buttonH, stripH, titleY = 29, 36, 41
	}
	left, right := edge+margin, w-margin
	n := len(a.snap.Accounts)
	tabW := min(146, (w-edge-62)/max(1, n))
	tabH := stripH - 8
	send.Call(a.children[idTabs], 0x1329, 0, uintptr(a.px(tabW))|uintptr(a.px(tabH-5))<<16)
	put(idTabs, edge+10, stripH-tabH, tabW*n+8, tabH)
	if n < workspaceTabs {
		put(idAdd, edge+20+tabW*n, stripH-tabH, 32, tabH-2)
	}

	title := a.titleOf(t.kind)
	if t.picker {
		title = a.tr("切换服务", "Switch service")
	}
	if t.kind == idWeChat && a.account.SignedIn && !t.picker {
		if t.view == idFiles {
			title = a.tr("文件收件箱", "File inbox")
		}
		if t.view == idTransfers {
			title = a.tr("传输队列", "Transfer queue")
		}
	}
	titleH := 33
	if h < 505 {
		titleH = 27
	}
	a.caption(title, left, titleY, right-left-151, titleH, a.titleFont, ink, singleLine)
	put(idServices, right-139, titleY+3, 102, buttonH)
	put(idMenu, right-32, titleY+3, 32, buttonH)
	a.statusInk = muted
	if !t.picker {
		switch {
		case t.kind == idWeChat && a.account.Error != "":
			a.statusInk = color(157, 60, 43)
		case t.kind == idWeChat && a.account.SignedIn, t.servers[t.kind] != nil:
			a.statusInk = accent
		}
	}
	sy := titleY + titleH
	a.statusDot = a.box(left+1, sy+8, 7, 7)
	a.caption(a.stateLabel(), left+14, sy, right-left-14, 21, a.smallFont, muted, singleLine)
	a.headerHeight = sy + 29

	navTop, step, navH := 88, 41, 37
	if h < 505 {
		navTop, step, navH = 60, 33, 29
	}
	navX := 12
	if !a.rail {
		navX = 6
	}
	navW := edge - 2*navX
	if a.rail {
		a.caption(a.tr("超级文件", "Ultra File"), 66, 18, edge-70, 28, a.brandFont, ink, singleLine)
		a.caption(a.tr("Ultra File", "File transfer"), 66, 44, edge-70, 18, a.smallFont, muted, singleLine)
	}
	if t.kind == idWeChat {
		for i, id := range []int{idChat, idFiles, idTransfers} {
			put(id, navX, navTop+i*step, navW, navH)
		}
	} else {
		put(idDownloads, navX, navTop, navW, navH)
	}
	servicesTop := navTop + 3*step + 32
	if a.rail {
		a.caption(a.tr("传输服务", "Services"), 24, servicesTop-26, edge-40, 20, a.smallFont, muted, singleLine)
	}
	if !a.rail {
		servicesTop -= 17
	}
	for i, id := range []int{idNavWeChat, idNavNearby, idNavLocal} {
		put(id, navX, servicesTop+i*step, navW, navH)
	}
	if a.rail {
		a.caption(a.tr("此电脑 · Windows", "This PC · Windows"), 44, h-42, edge-48, 28, a.smallFont, muted, singleLine)
	}
	if width := right - left; width > 1160 {
		inset := (width - 1160) / 2
		left += inset
		right -= inset
	}

	top := a.headerHeight + 8
	if a.snap.Update != nil {
		put(idNotice, left, top, min(345, right-left), buttonH)
		top += buttonH + 5
	}
	a.contentLeft, a.contentRight, a.contentTop = left, right, top
	if t.picker || t.kind != idWeChat {
		a.serviceLayout(put, left, right, top+8, buttonH)
	} else if a.account.SignedIn {
		a.layoutTransfer(put, left, right, top, buttonH)
	} else {
		a.layoutLogin(put, left, right, top, buttonH)
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
	a.placeOfficial()
	a.updateEditorScroll()
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

func (a *window) sidebarWidth() int {
	if a.motion.rail.initialized {
		return int(math.Round(a.motion.rail.value(time.Now())))
	}
	if a.rail {
		return railWidth
	}
	return 52
}

func (a *window) layoutTransfer(put positionControl, left, right, top, btnH int) {
	width := right - left
	compact := a.hgt < 505
	inset := 12
	if compact {
		inset = 8
	}
	innerLeft, innerRight := left+inset, right-inset
	bottom := a.hgt - inset
	y := bottom - inset - btnH
	put(idChoose, innerLeft, y, 108, btnH)
	put(idSend, innerRight-83, y, 83, btnH)
	if a.account.Busy {
		put(idStop, innerLeft+116, y, 95, btnH)
	} else if width > 430 {
		a.caption(a.tr("Ctrl + Enter 发送", "Ctrl + Enter to send"), innerLeft+111, y, innerRight-innerLeft-199, btnH, a.smallFont, muted, singleLine|1)
	}
	editH := 32
	if compact {
		editH = 24
	}
	editY := y - editH - 4
	put(idEditor, innerLeft+2, editY, innerRight-innerLeft-4, editH)
	composerTop := editY - inset
	queueView := a.tab(a.snap.Active).view == idTransfers
	n := len(a.account.Files)
	if queueView {
		a.caption(fmt.Sprintf(a.tr("待发送 · %d 个文件", "Queued · %d files"), n), left, top, width-95, btnH, a.brandFont, ink, singleLine)
		if n > 0 {
			put(idRemoveFile, right-84, top, 84, btnH)
			listY := top + btnH + 5
			put(idQueue, left, listY, width, max(20, composerTop-listY-6))
		} else {
			a.caption(a.tr("添加文件，或将文件拖到窗口。", "Add files or drop them into the window."), left, top+btnH+20, width, 34, a.font, muted, singleLine|1)
		}
	} else {
		if n > 0 {
			queueH := queueRowHeight * min(n, 2)
			if compact {
				queueH = queueRowHeight
			}
			listY := composerTop - queueH - 3
			put(idQueue, innerLeft, listY, innerRight-innerLeft, queueH)
			barH := btnH - 5
			barY := listY - barH - 3
			a.caption(fmt.Sprintf(a.tr("待发送 · %d 项", "Queued · %d"), n), innerLeft, barY, innerRight-innerLeft-85, barH, a.smallFont, muted, singleLine)
			put(idRemoveFile, innerRight-76, barY, 76, barH)
			composerTop = barY - 4
		}
		historyBottom := composerTop
		if m, ok := a.selectedMessage(); ok {
			barH := btnH - 4
			barY := composerTop - barH - 3
			x := right
			if m.CanDownload() {
				x -= 89
				put(idSave, x, barY, 89, barH)
				x -= 5
			}
			x -= 65
			put(idCopy, x, barY, 65, barH)
			x -= 5
			if a.canPreview(m) {
				x -= 108
				put(idCover, x, barY, 108, barH)
			}
			historyBottom = barY
		}
		historyH := max(18, historyBottom-top-6)
		if len(a.timeline) > 0 {
			put(idHistory, left, top, width, historyH)
		} else {
			empty := a.tr("收到的消息会显示在这里", "Received messages will appear here")
			if a.filesOnly {
				empty = a.tr("这个会话还没有附件", "No attachments in this conversation yet")
			}
			a.caption(empty, left, top+max(0, (historyH-30)/2), width, 30, a.font, muted, singleLine|1)
		}
	}
	a.composer = a.box(left, composerTop, width, bottom-composerTop)
}

func (a *window) paint() {
	var ps paintStruct
	dc, _, _ := user32.NewProc("BeginPaint").Call(a.h, uintptr(unsafe.Pointer(&ps)))
	defer user32.NewProc("EndPaint").Call(a.h, uintptr(unsafe.Pointer(&ps)))
	a.drawSurface(dc, ps.Paint)
}

func (a *window) drawSurface(dc uintptr, bounds rect) {
	fill(dc, bounds, base)
	edge := a.sidebarWidth()
	fill(dc, a.box(0, 0, edge, a.hgt), railBG)
	fill(dc, a.box(edge-1, 0, 1, a.hgt), border)
	size, x, y := 40, 16, 22
	if !a.rail {
		size, x, y = 32, 10, 12
	}
	logo := a.box(x, y, size, size)
	if icon := a.services.logos[idNearby]; icon != nil && icon.image != nil {
		icon.drawOn(dc, logo, railBG)
	}
	side := int32(a.px(15))
	x = 22
	if !a.rail {
		x = 18
	}
	a.glyph(dc, "monitor", int32(a.px(x)), int32(a.px(a.hgt-34)), side, muted)
	fill(dc, a.box(edge, 0, a.w-edge, a.headerHeight), base)
	stripH := 46
	if a.hgt < 505 {
		stripH = 36
	}
	fill(dc, a.box(edge, 0, a.w-edge, stripH), tabBG)
	fill(dc, a.box(edge, stripH-1, a.w-edge, 1), border)
	fill(dc, a.box(edge, a.headerHeight-1, a.w-edge, 1), border)
	if a.statusDot.Right > a.statusDot.Left {
		roundedSurface(dc, a.statusDot, a.px(8), a.statusInk, a.statusInk)
	}
	if a.composer.Right > a.composer.Left {
		outline := border
		if focused, _, _ := user32.NewProc("GetFocus").Call(); focused == a.children[idEditor] {
			outline = accent
		}
		roundedSurface(dc, a.composer, a.px(9), white, outline)
	}
	if a.queueSurface.Right > a.queueSurface.Left {
		roundedSurface(dc, a.queueSurface, a.px(7), base, border)
	}
	if r := a.directHero; r.Right > r.Left {
		roundedSurface(dc, r, a.px(10), accentWash, border)
		side := int32(a.px(28))
		x := r.Left + int32(a.px(12))
		y := (r.Top + r.Bottom - side) / 2
		a.glyph(dc, "monitor", x, y, side, accent)
	}
	for _, card := range a.directCards {
		roundedSurface(dc, card, a.px(9), white, border)
	}
	if bar := a.directProgress; bar.Right > bar.Left {
		roundedSurface(dc, bar, a.px(3), accentWash, 0)
		s := a.tab(a.snap.Active).direct.last
		if s.Size > 0 && s.Done > 0 {
			bar.Right = bar.Left + int32(int64(bar.Right-bar.Left)*s.Done/s.Size)
			roundedSurface(dc, bar, a.px(3), accent, 0)
		}
	}
	for _, label := range a.labels {
		textAt(dc, label.text, label.box, label.font, label.ink, label.flags)
	}
	if r := a.qrBounds; r.Right > r.Left {
		roundedSurface(dc, a.qrCard, a.px(14), white, border)
		if t := a.tab(a.snap.Active); t.kind == idNearby && len(t.pair.pixels) > 0 {
			t.pair.draw(dc, r, 0)
		} else if len(a.qr.pixels) > 0 {
			a.qr.draw(dc, r, a.px(5))
		} else {
			text := a.tr("正在获取二维码…", "Requesting code…")
			if a.account.Status == "idle" || a.account.Status == "expired" || a.account.Error != "" {
				text = a.tr("等待获取二维码", "Waiting for a code")
			}
			textAt(dc, text, r, a.smallFont, muted, singleLine|1)
		}
	}
	if t := a.tab(a.snap.Active); t.kind == idWeChat && !t.picker && a.account.Busy && a.account.Total > 0 {
		width := a.contentRight - a.contentLeft
		progress := min(100, max(0, int(a.account.Sent*100/a.account.Total)))
		fill(dc, a.box(a.contentLeft, a.headerHeight-3, width, 2), accentWash)
		fill(dc, a.box(a.contentLeft, a.headerHeight-3, width*progress/100, 2), accent)
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
