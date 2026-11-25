// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"strings"
	"time"
	"unicode"
	"unsafe"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
)

// radius is in device pixels. An outline of zero omits the border.
func roundedSurface(dc uintptr, r rect, radius int, background, outline uintptr) {
	if r.Right <= r.Left || r.Bottom <= r.Top {
		return
	}
	save, _, _ := gdi32.NewProc("SaveDC").Call(dc)
	brush, _, _ := gdi32.NewProc("GetStockObject").Call(18) // DC_BRUSH
	selectObject.Call(dc, brush)
	gdi32.NewProc("SetDCBrushColor").Call(dc, background)
	penType := uintptr(8) // NULL_PEN
	if outline != 0 {
		penType = 19
		gdi32.NewProc("SetDCPenColor").Call(dc, outline)
	}
	pen, _, _ := gdi32.NewProc("GetStockObject").Call(penType)
	selectObject.Call(dc, pen)
	gdi32.NewProc("RoundRect").Call(dc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), uintptr(radius*2), uintptr(radius*2))
	gdi32.NewProc("RestoreDC").Call(dc, save)
}

func (a *window) glyph(dc uintptr, key string, x, y, side int32, tone uintptr) {
	paths := [][]point{}
	switch key {
	case "document":
		paths = [][]point{{{4, 2}, {12, 2}, {17, 7}, {17, 18}, {4, 18}, {4, 2}}, {{12, 2}, {12, 7}, {17, 7}}, {{7, 10}, {14, 10}}, {{7, 13}, {12, 13}}}
	case "chat":
		paths = [][]point{{{2, 3}, {18, 3}, {18, 14}, {9, 14}, {5, 18}, {5, 14}, {2, 14}, {2, 3}}}
	case "plus":
		paths = [][]point{{{3, 10}, {17, 10}}, {{10, 3}, {10, 17}}}
	case "more":
		d := max(int32(3), int32(a.px(3)))
		for _, i := range []int32{3, 10, 17} {
			cx := x + side*i/20
			cy := y + side/2
			fill(dc, rect{cx - d/2, cy - d/2, cx + (d+1)/2, cy + (d+1)/2}, tone)
		}
		return
	}
	pen, _, _ := gdi32.NewProc("CreatePen").Call(0, uintptr(max(1, a.px(1))), tone)
	old, _, _ := selectObject.Call(dc, pen)
	for _, p := range paths {
		for i := range p {
			p[i] = point{x + (p[i].X*side+10)/20, y + (p[i].Y*side+10)/20}
		}
		gdi32.NewProc("Polyline").Call(dc, uintptr(unsafe.Pointer(&p[0])), uintptr(len(p)))
	}
	selectObject.Call(dc, old)
	deleteObject.Call(pen)
}

func (a *window) drawButton(d *drawItem) {
	id := int(d.ID)
	r := d.Bounds
	rail := id == idChat || id == idFiles || id == idDownloads || id == idUpdate || id == idAbout
	quiet := rail || id == idAdd || id == idMenu || id == idCopy || id == idSave || id == idRemoveFile
	surface := base
	if rail {
		surface = railBG
	} else if id == idAdd || id == idMenu || id == idRemoveFile {
		surface = white
	}
	bg, fg, stroke := white, ink, border
	if quiet {
		bg, fg, stroke = surface, muted, 0
	}
	active := id == idChat && !a.filesOnly || id == idFiles && a.filesOnly || id == idNotice
	if active {
		bg, fg, stroke = paleBlue, blue, 0
	}
	primary := id == idSend || id == idRefresh
	if primary {
		bg, fg, stroke = blue, white, 0
	}
	if a.hot == id && d.State&4 == 0 {
		if primary {
			bg = color(0, 85, 155)
		} else {
			bg = color(228, 237, 246)
		}
	}
	if d.State&1 != 0 {
		if primary {
			bg = color(0, 72, 131)
		} else {
			bg = color(211, 225, 239)
		}
	}
	if d.State&4 != 0 {
		fg = color(109, 123, 139)
		if quiet {
			bg, stroke = surface, 0
		} else {
			bg, stroke = color(235, 240, 245), color(218, 226, 233)
		}
	}
	if d.State&0x10 != 0 && d.State&0x200 == 0 {
		stroke = blue
	}
	fill(d.DC, r, surface)
	roundedSurface(d.DC, r, a.px(6), bg, stroke)
	side := int32(a.px(17))
	cy := (r.Top + r.Bottom - side) / 2
	if id == idAdd || id == idMenu {
		key := "plus"
		if id == idMenu {
			key = "more"
		}
		a.glyph(d.DC, key, (r.Left+r.Right-side)/2, cy, side, fg)
		return
	}
	flags := uintptr(singleLine | 1)
	font := a.font
	if rail {
		flags = singleLine
		r.Left += int32(a.px(12))
		if id == idChat || id == idFiles {
			key := "chat"
			if id == idFiles {
				key = "document"
			}
			a.glyph(d.DC, key, r.Left, cy, side, fg)
			r.Left += side + int32(a.px(6))
		}
	}
	if active || primary {
		font = a.brandFont
	}
	textAt(d.DC, getText(d.Window), r, font, fg, flags)
}

func (a *window) drawTab(d *drawItem) {
	index := int(d.Item)
	if index < 0 || index >= len(a.snap.Accounts) {
		return
	}
	r := d.Bounds
	fill(d.DC, r, white)
	r.Left += int32(a.px(1))
	r.Right -= int32(a.px(2))
	r.Top += int32(a.px(1))
	current := a.snap.Accounts[index].ID == a.snap.Active
	bg, tone, stroke := color(240, 244, 248), muted, color(224, 231, 238)
	if current {
		bg, tone, stroke = base, blue, border
	}
	if d.State&0x10 != 0 && d.State&0x200 == 0 {
		stroke = blue
	}
	roundedSurface(d.DC, r, a.px(6), bg, stroke)
	if current {
		w := int32(a.px(26))
		x := (r.Left + r.Right - w) / 2
		h := int32(a.px(2))
		fill(d.DC, rect{x, r.Bottom - h*2, x + w, r.Bottom - h}, blue)
	}
	title := ""
	if index < len(a.tabTexts) {
		title = a.tabTexts[index]
	}
	r.Left += int32(a.px(7))
	r.Right -= int32(a.px(7))
	r.Bottom -= int32(a.px(3))
	textAt(d.DC, title, r, a.font, tone, singleLine|1)
}

func shortPreview(s string) string {
	var b strings.Builder
	space := false
	n := 0
	for _, r := range s {
		if unicode.IsSpace(r) {
			space = b.Len() > 0
			continue
		}
		if n == 220 {
			b.WriteRune('…')
			break
		}
		if space {
			b.WriteByte(' ')
			space = false
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}
func textWidth(dc uintptr, s string, font uintptr) int32 {
	old, _, _ := selectObject.Call(dc, font)
	var bounds rect
	drawText.Call(dc, uintptr(unsafe.Pointer(wide(s))), ^uintptr(0), uintptr(unsafe.Pointer(&bounds)), 0x400|0x800|0x20)
	selectObject.Call(dc, old)
	return bounds.Right
}
func (a *window) drawListRow(d *drawItem) {
	r := d.Bounds
	queue := d.ID == idQueue
	fill(d.DC, r, base)
	if d.Item == 0xffffffff {
		return
	}
	i := int(d.Item)
	stroke := uintptr(0)
	if d.State&1 != 0 || d.State&0x10 != 0 && d.State&0x200 == 0 {
		stroke = blue
	}
	if queue {
		if i < 0 || i >= len(a.account.Files) {
			return
		}
		if stroke != 0 {
			roundedSurface(d.DC, r, a.px(5), paleBlue, stroke)
		}
		f := a.account.Files[i]
		r.Left += int32(a.px(10))
		r.Right -= int32(a.px(12))
		icon := int32(a.px(14))
		a.glyph(d.DC, "document", r.Left, (r.Top+r.Bottom-icon)/2, icon, muted)
		r.Left += icon + int32(a.px(7))
		name := r
		name.Right -= int32(a.px(95))
		textAt(d.DC, f.Name, name, a.font, ink, singleLine)
		textAt(d.DC, fileSize(f.Size), r, a.smallFont, muted, singleLine|2)
		return
	}
	if i < 0 || i >= len(a.messageIDs) {
		return
	}
	for _, m := range a.account.Messages {
		if m.ID == a.messageIDs[i] {
			a.drawMessage(d.DC, r, m, stroke)
			return
		}
	}
}

func (a *window) drawMessage(dc uintptr, r rect, m wechat.Message, stroke uintptr) {
	author := a.tr("手机微信", "Phone")
	tint := color(236, 242, 247)
	if m.Outgoing {
		author = a.tr("此电脑", "This PC")
		tint = paleBlue
	}
	meta := author + " · " + time.Unix(m.Timestamp, 0).Format("15:04")
	file := isAttachment(m)
	body := shortPreview(m.Text)
	font := a.font
	if file {
		font = a.brandFont
		if m.Size > 0 {
			meta += " · " + fileSize(m.Size)
		}
	}
	pad := int32(a.px(13))
	extra := int32(0)
	if file {
		extra = int32(a.px(33))
	}
	maxW := max(int32(a.px(64)), (r.Right-r.Left)*4/5)
	w := min(maxW, max(int32(a.px(103)), max(textWidth(dc, meta, a.smallFont), textWidth(dc, body, font))+extra+pad*2))
	left := r.Left + int32(a.px(2))
	if m.Outgoing {
		left = r.Right - int32(a.px(2)) - w
	}
	bubble := rect{left, r.Top + int32(a.px(4)), left + w, r.Bottom - int32(a.px(5))}
	roundedSurface(dc, bubble, a.px(9), tint, stroke)
	content := bubble
	content.Left += pad
	content.Right -= pad
	if file {
		side := int32(a.px(26))
		a.glyph(dc, "document", content.Left, (content.Top+content.Bottom-side)/2, side, blue)
		content.Left += extra
	}
	y := (content.Top + content.Bottom) / 2
	upper, lower := content, content
	upper.Top += int32(a.px(5))
	upper.Bottom = y
	lower.Top = y
	lower.Bottom -= int32(a.px(5))
	if file {
		textAt(dc, body, upper, font, ink, singleLine)
		textAt(dc, meta, lower, a.smallFont, muted, singleLine)
	} else {
		textAt(dc, meta, upper, a.smallFont, muted, singleLine)
		textAt(dc, body, lower, font, ink, singleLine)
	}
}
