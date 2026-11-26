// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unsafe"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/desktop"
	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
)

type timelineItem struct {
	message wechat.Message
	date    string
	cover   *qrBitmap
}

func (a *window) dateLabel(t time.Time) string {
	now := time.Now()
	if t.Format("2006-01-02") == now.Format("2006-01-02") {
		return a.tr("今天  "+t.Format("1 月 2 日"), "Today  "+t.Format("2 Jan"))
	}
	return a.tr(t.Format("2006 年 1 月 2 日"), t.Format("2 Jan 2006"))
}

func (a *window) canPreview(m wechat.Message) bool {
	_, inspected := a.account.Cover(m)
	return !inspected && desktop.IsPresentation(m.Text) && m.CanDownload() && m.Size > 0 && m.Size <= wechat.MaxFileSize
}

func (a *window) updateTimeline(switched bool) {
	var items []timelineItem
	var rows, ids []string
	newImages := make(map[string]*qrBitmap)
	day := ""
	for _, m := range a.account.Messages {
		if a.filesOnly && !isAttachment(m) {
			continue
		}
		when := time.Unix(m.Timestamp, 0)
		date := when.Format("2006-01-02")
		item := timelineItem{message: m}
		if date != day {
			item.date = a.dateLabel(when)
			day = date
		}
		if data, ok := a.account.Cover(m); ok && len(data) > 0 {
			bitmap := a.coverImages[m.ID]
			if bitmap == nil {
				bitmap = new(qrBitmap)
			}
			bitmap.load(data)
			if len(bitmap.pixels) > 0 {
				item.cover = bitmap
				newImages[m.ID] = bitmap
			}
		}
		who := a.tr("我的手机", "My phone")
		if m.Outgoing {
			who = a.tr("此电脑", "This PC")
		}
		label := item.date + " " + when.Format("15:04") + " " + who + " " + m.Text
		if isAttachment(m) && m.Size > 0 {
			label += " · " + fileSize(m.Size)
		}
		items = append(items, item)
		ids = append(ids, m.ID)
		rows = append(rows, label)
	}
	a.coverImages = newImages
	h := a.children[idHistory]
	same := !switched && slices.Equal(ids, a.messageIDs) && slices.Equal(rows, a.historyRows)
	if same {
		for i := range items {
			if items[i].cover != a.timeline[i].cover {
				same = false
				break
			}
		}
	}
	if same {
		a.timeline = items
		return
	}
	oldSel, _, _ := send.Call(h, 0x188, 0, 0)
	oldTop, _, _ := send.Call(h, 0x18e, 0, 0)
	selectedID, topID := "", ""
	if !switched && int(oldSel) >= 0 && int(oldSel) < len(a.messageIDs) {
		selectedID = a.messageIDs[oldSel]
	}
	if !switched && int(oldTop) >= 0 && int(oldTop) < len(a.messageIDs) {
		topID = a.messageIDs[oldTop]
	}
	var client, last rect
	user32.NewProc("GetClientRect").Call(h, uintptr(unsafe.Pointer(&client)))
	atEnd := switched || len(a.timeline) == 0
	if !atEnd {
		ok, _, _ := send.Call(h, 0x198, uintptr(len(a.timeline)-1), uintptr(unsafe.Pointer(&last)))
		atEnd = int32(ok) != -1 && last.Bottom <= client.Bottom+int32(a.px(3))
	}
	send.Call(h, 0x0b, 0, 0)
	send.Call(h, 0x184, 0, 0)
	a.timeline, a.historyRows, a.messageIDs = items, rows, ids
	for _, r := range rows {
		send.Call(h, 0x180, 0, uintptr(unsafe.Pointer(wide(r))))
	}
	if index := slices.Index(ids, selectedID); selectedID != "" && index >= 0 {
		send.Call(h, 0x186, uintptr(index), 0)
	}
	if len(items) > 0 {
		if atEnd && selectedID == "" {
			send.Call(h, 0x197, uintptr(len(items)-1), 0)
		} else if index := slices.Index(ids, topID); topID != "" && index >= 0 {
			send.Call(h, 0x197, uintptr(index), 0)
		}
	}
	// WM_SETREDRAW can change WS_VISIBLE. Layout owns child visibility.
	send.Call(h, 0x0b, 1, 0)
	if _, visible := a.placements[idHistory]; !visible {
		showWindow.Call(h, 0)
	}
	redraw(h)
}

func (a *window) rowPixels(i int) int {
	h := 64
	if i >= 0 && i < len(a.timeline) {
		item := a.timeline[i]
		if isAttachment(item.message) {
			h = 84
		}
		if item.date != "" {
			h += 30
		}
		if item.cover != nil {
			h += 90
		}
	}
	return min(255, a.px(h))
}

func (a *window) sizeTimeline() {
	for i := range a.timeline {
		send.Call(a.children[idHistory], 0x1a0, uintptr(i), uintptr(a.rowPixels(i)))
	}
}

func attachmentType(m wechat.Message) (string, uintptr, uintptr) {
	ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(m.Text), "."))
	switch ext {
	case "PPT", "PPTX", "ODP":
		return ext, color(247, 230, 215), color(147, 64, 29)
	case "XLS", "XLSX", "CSV", "ODS":
		return ext, color(222, 239, 224), color(31, 100, 64)
	case "PDF":
		return ext, color(251, 228, 224), color(161, 51, 44)
	case "ZIP", "RAR", "7Z", "GZ":
		return ext, color(238, 229, 248), color(109, 66, 142)
	}
	if len(ext) > 5 {
		ext = ""
	}
	if ext == "" {
		ext = map[string]string{"image": "IMG", "video": "VID", "voice": "AUD"}[m.Kind]
		if ext == "" {
			ext = "FILE"
		}
	}
	return ext, color(227, 237, 242), color(57, 91, 108)
}

func (a *window) drawTimelineItem(dc uintptr, r rect, item timelineItem, stroke uintptr) {
	p := func(d int) int32 { return int32(a.px(d)) }
	if item.date != "" {
		date := r
		date.Bottom = min(r.Bottom, r.Top+p(27))
		textAt(dc, item.date, date, a.smallFont, muted, singleLine|1)
		r.Top = date.Bottom
	}
	m := item.message
	author := a.tr("我的手机", "My phone")
	bg := white
	align := uintptr(singleLine)
	if m.Outgoing {
		author = a.tr("此电脑", "This PC")
		bg = accentWash
		align |= 2
	}
	body := shortPreview(m.Text)
	file := isAttachment(m)
	contentWidth := textWidth(dc, body, a.font) + p(27)
	if file {
		contentWidth = max(p(167), textWidth(dc, body, a.brandFont)+p(74))
	}
	if item.cover != nil {
		contentWidth = max(contentWidth, p(284))
	}
	room := r.Right - r.Left
	inset := min(p(72), max(p(6), room/14))
	maxW := max(p(60), room-inset*2)
	w := min(maxW, contentWidth)
	left := r.Left + inset
	if m.Outgoing {
		left = r.Right - inset - w
	}
	meta := rect{r.Left + inset, r.Top, r.Right - inset, r.Top + p(22)}
	textAt(dc, author+"   "+time.Unix(m.Timestamp, 0).Format("15:04"), meta, a.smallFont, muted, align)
	bubble := rect{left, meta.Bottom, left + w, r.Bottom - p(6)}
	if bubble.Bottom <= bubble.Top {
		return
	}
	if stroke == 0 {
		stroke = border
	}
	roundedSurface(dc, bubble, a.px(7), bg, stroke)
	if !file {
		label := bubble
		label.Left += p(12)
		label.Right -= p(12)
		textAt(dc, body, label, a.font, ink, singleLine)
		return
	}
	infoTop := bubble.Top
	if item.cover != nil {
		art := rect{bubble.Left + p(2), bubble.Top + p(2), bubble.Right - p(2), bubble.Bottom - p(53)}
		if art.Bottom > art.Top {
			item.cover.draw(dc, art, 0)
			infoTop = art.Bottom
		}
	}
	ext, tint, tone := attachmentType(m)
	x := bubble.Left + p(11)
	cy := (infoTop + bubble.Bottom) / 2
	tile := rect{x, cy - p(18), x + p(37), cy + p(18)}
	roundedSurface(dc, tile, a.px(4), tint, 0)
	textAt(dc, ext, tile, a.smallFont, tone, singleLine|1)
	label := rect{tile.Right + p(9), cy - p(22), bubble.Right - p(9), cy + p(2)}
	textAt(dc, body, label, a.brandFont, ink, singleLine)
	details := ext
	if m.Size > 0 {
		details = fileSize(m.Size) + " · " + details
	}
	if data, known := a.account.Cover(m); known && len(data) == 0 && desktop.IsPresentation(m.Text) {
		details += " · " + a.tr("无内嵌封面", "No embedded cover")
	}
	if a.account.Previewing == m.ID {
		details += " · " + a.tr("读取封面中", "Loading cover")
	}
	label.Top = cy + p(2)
	label.Bottom = bubble.Bottom - p(2)
	textAt(dc, details, label, a.smallFont, muted, singleLine)
}
