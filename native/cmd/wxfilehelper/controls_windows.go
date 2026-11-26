// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"strings"
	"unicode"
	"unsafe"
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
	case "folder":
		paths = [][]point{{{2, 5}, {8, 5}, {10, 7}, {18, 7}, {18, 17}, {2, 17}, {2, 5}}}
	case "queue":
		paths = [][]point{{{3, 4}, {5, 4}}, {{8, 4}, {17, 4}}, {{3, 10}, {5, 10}}, {{8, 10}, {17, 10}}, {{3, 16}, {5, 16}}, {{8, 16}, {17, 16}}}
	case "monitor":
		paths = [][]point{{{2, 3}, {18, 3}, {18, 14}, {2, 14}, {2, 3}}, {{10, 14}, {10, 17}}, {{5, 17}, {15, 17}}}
	case "attach":
		paths = [][]point{{{7, 13}, {14, 6}, {12, 4}, {5, 11}, {4, 14}, {6, 17}, {9, 17}, {18, 8}, {18, 5}, {15, 2}, {12, 2}, {3, 11}}}
	case "send":
		paths = [][]point{{{2, 3}, {18, 10}, {2, 17}, {5, 10}, {2, 3}}, {{5, 10}, {14, 10}}}
	case "switch":
		paths = [][]point{{{2, 6}, {17, 6}, {13, 2}}, {{18, 14}, {3, 14}, {7, 18}}}
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

func navService(id int) int {
	switch id {
	case idNavWeChat:
		return idWeChat
	case idNavNearby:
		return idNearby
	case idNavLocal:
		return idLocal
	}
	return 0
}

func (a *window) drawButton(d *drawItem) {
	id := int(d.ID)
	r := d.Bounds
	kind := navService(id)
	rail := kind != 0 || id == idChat || id == idFiles || id == idTransfers || id == idDownloads
	bg, fg, stroke, surface := white, ink, border, base
	if rail {
		surface = railBG
		bg, stroke = surface, 0
	}
	if id == idAdd {
		surface = tabBG
		bg, stroke = surface, 0
	}
	if id == idMenu || id == idCopy || id == idSave || id == idRemoveFile || id == idCover {
		bg, stroke = surface, 0
	}
	t := a.tab(a.snap.Active)
	selected := !t.picker && t.kind == idWeChat && id == t.view
	if selected || id == idNotice {
		bg, fg, stroke = accentWash, accent, 0
	}
	primary := id == idSend || id == idRefresh || id == idHost || id == idNearSend || id == idAccept
	if primary {
		bg, fg, stroke = accent, white, 0
	}
	if d.State&4 != 0 {
		fg = color(100, 114, 105)
		if primary {
			bg, stroke = color(230, 237, 230), border
		}
	} else if d.State&1 != 0 {
		if primary {
			bg = color(24, 78, 58)
		} else {
			bg = color(212, 228, 218)
		}
	} else if a.hot == id {
		if primary {
			bg = color(28, 89, 68)
		} else {
			bg = color(226, 236, 227)
		}
	}
	if d.State&0x10 != 0 && d.State&0x200 == 0 {
		stroke = accent
	}
	fill(d.DC, r, surface)
	roundedSurface(d.DC, r, a.px(7), bg, stroke)
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
	if selected || primary {
		font = a.brandFont
	}
	// Icons for services are the same in the rail, picker and tabs.
	logo := kind
	if _, ok := serviceKeys[id]; ok {
		logo = id
	}
	if logo != 0 {
		side = int32(a.px(19))
		x := r.Left + int32(a.px(11))
		if rail && !a.rail {
			x = (r.Left + r.Right - side) / 2
		}
		cy = (r.Top + r.Bottom - side) / 2
		if icon := a.services.logos[logo]; icon != nil && len(icon.pixels) > 0 {
			icon.draw(d.DC, rect{x, cy, x + side, cy + side}, 0)
		}
		if rail && !a.rail {
			if kind == t.kind && !t.picker {
				fill(d.DC, rect{r.Left, cy, r.Left + int32(a.px(2)), cy + side}, accent)
			}
			return
		}
		r.Left = x + side + int32(a.px(9))
		flags = singleLine
		if rail && kind == t.kind && !t.picker {
			tag := r
			tag.Right -= int32(a.px(8))
			textAt(d.DC, a.tr("当前", "Active"), tag, a.smallFont, accent, singleLine|2)
			r.Right -= int32(a.px(43))
		}
	} else {
		key := map[int]string{idChat: "chat", idFiles: "folder", idTransfers: "queue", idDownloads: "folder", idChoose: "attach", idSend: "send", idServices: "switch"}[id]
		if key != "" {
			x := r.Left + int32(a.px(11))
			if rail && !a.rail {
				x = (r.Left + r.Right - side) / 2
			}
			a.glyph(d.DC, key, x, cy, side, fg)
			if rail && !a.rail {
				return
			}
			r.Left = x + side + int32(a.px(6))
			flags = singleLine
		}
	}
	textAt(d.DC, getText(d.Window), r, font, fg, flags)
}

func (a *window) drawTab(d *drawItem) {
	index := int(d.Item)
	if index < 0 || index >= len(a.snap.Accounts) {
		return
	}
	r := d.Bounds
	fill(d.DC, r, tabBG)
	r.Left += int32(a.px(1))
	r.Right -= int32(a.px(2))
	r.Top += int32(a.px(1))
	ac := a.snap.Accounts[index]
	tone := muted
	current := ac.ID == a.snap.Active
	if current || d.State&0x10 != 0 {
		stroke := uintptr(0)
		if d.State&0x10 != 0 {
			stroke = accent
		}
		roundedSurface(d.DC, r, a.px(6), base, stroke)
		tone = ink
	}
	if current {
		fill(d.DC, rect{r.Left, r.Bottom - int32(a.px(2)), r.Right, r.Bottom}, accent)
	}
	side := int32(a.px(16))
	x := r.Left + int32(a.px(9))
	cy := (r.Top + r.Bottom - side) / 2
	if icon := a.services.logos[a.tab(ac.ID).kind]; icon != nil && len(icon.pixels) > 0 {
		icon.draw(d.DC, rect{x, cy, x + side, cy + side}, 0)
	}
	r.Left = x + side + int32(a.px(6))
	r.Right -= int32(a.px(4))
	r.Bottom -= int32(a.px(2))
	if index < len(a.tabTexts) {
		textAt(d.DC, a.tabTexts[index], r, a.font, tone, singleLine)
	}
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
		stroke = accent
	}
	if queue {
		if i < 0 || i >= len(a.account.Files) {
			return
		}
		if stroke != 0 {
			roundedSurface(d.DC, r, a.px(5), accentWash, stroke)
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
	if i >= 0 && i < len(a.timeline) {
		a.drawTimelineItem(d.DC, r, a.timeline[i], stroke)
	}
}
