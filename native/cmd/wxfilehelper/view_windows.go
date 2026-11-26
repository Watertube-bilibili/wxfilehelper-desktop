// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"slices"
	"strings"
	"time"
	"unsafe"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/desktop"
	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
)

var (
	base     = color(249, 251, 253)
	railBG   = color(238, 243, 248)
	border   = color(206, 217, 229)
	ink      = color(31, 46, 65)
	muted    = color(82, 101, 125)
	blue     = color(0, 98, 174)
	paleBlue = color(224, 239, 253)
	white    = color(255, 255, 255)
)

type qrBitmap struct {
	source, pixels []byte
	width, height  int
}

func (q *qrBitmap) load(data []byte) {
	if bytes.Equal(data, q.source) {
		return
	}
	*q = qrBitmap{source: data}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 2048 || cfg.Height > 2048 {
		return
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}
	q.width, q.height = cfg.Width, cfg.Height
	q.pixels = make([]byte, q.width*q.height*4)
	for y := 0; y < q.height; y++ {
		for x := 0; x < q.width; x++ {
			r, g, b, _ := img.At(img.Bounds().Min.X+x, img.Bounds().Min.Y+y).RGBA()
			i := 4 * (y*q.width + x)
			q.pixels[i] = byte(b >> 8)
			q.pixels[i+1] = byte(g >> 8)
			q.pixels[i+2] = byte(r >> 8)
		}
	}
}
func (a *window) createChildren() error {
	for _, id := range []int{idTabs, idAdd, idMenu, idChat, idFiles, idDownloads, idUpdate, idAbout, idRefresh, idHistory, idSave, idCopy, idQueue, idRemoveFile, idEditor, idChoose, idSend, idStop, idNotice, idServices, idWeChat, idNearby, idLocal, idQQ, idDefaultService, idHost, idPair, idPeer, idNearSend, idAccept, idReject} {
		class := "BUTTON"
		style := uintptr(0x40010000 | 0x0000000b)
		ex := uintptr(0) // WS_CHILD | WS_TABSTOP, owner drawn button
		switch id {
		case idTabs:
			class = "SysTabControl32"
			// Multiline mode suppresses the system up/down spinner. Our two fixed
			// slots fit in one row. The native tab retains its full input behavior.
			style = 0x40010000 | 0x0400 | 0x2000 | 0x200 | 0x800
		case idHistory, idQueue:
			class = "LISTBOX"
			style = 0x40210000 | 0x101 | 0x40 | 0x10 // notifications, fixed owner-drawn rows, accessible strings
		case idPeer:
			class = "EDIT"
			style = 0x40810080
		case idEditor:
			class = "EDIT"
			style = 0x40210000 | 0x4 | 0x40 | 0x1000 // multiline, autovscroll, want return
		}
		child, _, err := createWindow.Call(ex, uintptr(unsafe.Pointer(wide(class))), 0, style, 0, 0, 1, 1, a.h, uintptr(id), a.instance, 0)
		if child == 0 {
			return fmt.Errorf("%s (%d): %w", class, id, err)
		}
		a.children[id] = child
		if id == idTabs {
			if ok, _, err := comctl32.NewProc("SetWindowSubclass").Call(child, tabCallback, 1, 0); ok == 0 {
				return fmt.Errorf("Initialize account tabs: %w", err)
			}
		}
	}
	send.Call(a.children[idPeer], 0x1501, 0, uintptr(unsafe.Pointer(wide("LocalSend IP (192.168.1.20)"))))
	send.Call(a.children[idEditor], 0x00c5, 1024*1024, 0)
	return nil
}
func (a *window) freeFonts() {
	for _, font := range []uintptr{a.font, a.smallFont, a.brandFont, a.titleFont} {
		if font != 0 {
			deleteObject.Call(font)
		}
	}
}
func (a *window) setFonts() {
	old := []uintptr{a.font, a.smallFont, a.brandFont, a.titleFont}
	size := 15
	a.font = createFont(size, 400, a.dpi)
	a.smallFont = createFont(size-2, 400, a.dpi)
	a.brandFont = createFont(size+1, 600, a.dpi)
	a.titleFont = createFont(size+8, 600, a.dpi)
	for _, h := range a.children {
		send.Call(h, 0x30, a.font, 1)
	}
	send.Call(a.children[idHistory], 0x1a0, 0, uintptr(a.px(size*4+10)))
	send.Call(a.children[idQueue], 0x1a0, 0, uintptr(a.px(queueRowHeight)))
	for _, h := range old {
		if h != 0 {
			deleteObject.Call(h)
		}
	}
}

const queueRowHeight = 30

func (a *window) changeLabels() {
	for id, s := range map[int]string{
		idServices: a.tr("切换服务", "Services"), idAdd: a.tr("新建标签", "New tab"), idMenu: a.tr("更多操作", "More actions"), idChat: a.tr("文件传输", "Conversation"), idFiles: a.tr("本次会话文件", "Session files"),
		idDownloads: a.tr("下载目录", "Downloads"), idUpdate: a.tr("检查更新", "Check updates"), idAbout: a.tr("关于", "About"),
		idRefresh: a.tr("刷新二维码", "Refresh code"), idSave: a.tr("另存为…", "Save as…"), idCopy: a.tr("复制", "Copy"), idRemoveFile: a.tr("移除", "Remove"),
		idChoose: a.tr("添加文件…", "Add files…"), idSend: a.tr("发送", "Send"), idStop: a.tr("停止队列", "Stop queue"),
	} {
		setText(a.children[id], s)
	}
	if a.snap.Update != nil {
		setText(a.children[idNotice], a.tr("可更新至 ", "Update to ")+a.snap.Update.Version+"  ↗")
	}
	setText(a.h, a.tr("超级文件", "Ultra File"))
}
func (a *window) refresh(force bool) {
	a.display(a.c.Snapshot(), force)
}

func (a *window) display(s desktop.Snapshot, force bool) {
	if !force && s.Revision == a.snap.Revision {
		return
	}
	a.observeTransfers(s)
	switched := s.Active != a.account.ID
	if switched && a.account.ID != 0 {
		a.tab(a.account.ID).peer = getText(a.children[idPeer])
	}
	if switched {
		setText(a.children[idPeer], a.tab(s.Active).peer)
	}
	a.snap = s
	for _, v := range s.Accounts {
		a.reconcileDraft(v)
		if v.ID == s.Active {
			a.account = v
		}
	}
	if switched || getText(a.children[idEditor]) != a.drafts[s.Active] {
		setText(a.children[idEditor], a.drafts[s.Active])
	}
	if force || switched {
		a.historyRows = nil
		a.queueRows = nil
	}
	a.qr.load(a.account.QR)
	var titles []string
	activeIndex := 0
	for i, acct := range s.Accounts {
		title := fmt.Sprintf(a.tr("账号 %d", "Account %d"), i+1)
		if acct.Name != "" {
			r := []rune(acct.Name)
			if len(r) > 11 {
				r = append(r[:10], '…')
			}
			title = string(r)
		}
		if kind := a.tab(acct.ID).kind; kind != idWeChat {
			title = a.titleOf(kind)
		} else if acct.Name == "" {
			title = fmt.Sprintf(a.tr("微信 %d", "WeChat %d"), i+1)
		}
		if acct.Busy || acct.Downloading {
			title += " ·"
		}
		titles = append(titles, title)
		if acct.ID == s.Active {
			activeIndex = i
		}
	}
	if !slices.Equal(titles, a.tabTexts) {
		h := a.children[idTabs]
		a.tabTexts = titles
		send.Call(h, 0x1309, 0, 0)
		for i, title := range titles {
			item := tabItem{Mask: 1, Text: wide(title)}
			send.Call(h, 0x133e, uintptr(i), uintptr(unsafe.Pointer(&item)))
		}
	}
	send.Call(a.children[idTabs], 0x130c, uintptr(activeIndex), 0)
	var rows, ids []string
	for _, m := range a.account.Messages {
		if a.filesOnly && !isAttachment(m) {
			continue
		}
		who := a.tr("手机", "Phone")
		if m.Outgoing {
			who = a.tr("此电脑", "This PC")
		}
		desc := strings.Join(strings.Fields(m.Text), " ")
		if m.Kind != "text" && m.Size > 0 {
			desc += "   ·   " + fileSize(m.Size)
		}
		rows = append(rows, time.Unix(m.Timestamp, 0).Format("15:04")+"    "+who+"    "+desc)
		ids = append(ids, m.ID)
	}
	a.messageIDs = ids
	a.updateList(idHistory, rows, &a.historyRows, true, switched || force)
	rows = nil
	for _, f := range a.account.Files {
		rows = append(rows, f.Name+"    ·    "+fileSize(f.Size))
	}
	a.updateList(idQueue, rows, &a.queueRows, false, switched || force)
	a.changeLabels()
	a.enableActions()
	a.layout()
	redraw(a.h)
	if a.manualUpdate && (s.UpdateStatus == "available" || s.UpdateStatus == "current" || s.UpdateStatus == "failed") {
		a.manualUpdate = false
		if s.UpdateStatus != "available" {
			desc := a.tr("已是最新版本。", "You’re up to date.")
			if s.UpdateStatus == "failed" {
				desc = a.tr("检查更新失败，请稍后重试。", "Could not check for updates. Try again later.")
			}
			dialog(a.h, desc, "WX FileHelper", 0x40)
		}
	}
}
func (a *window) updateList(id int, rows []string, previous *[]string, tail, reset bool) {
	if !reset && slices.Equal(rows, *previous) {
		return
	}
	h := a.children[id]
	oldSel, _, _ := send.Call(h, 0x188, 0, 0)
	top, _, _ := send.Call(h, 0x18e, 0, 0)
	var client rect
	user32.NewProc("GetClientRect").Call(h, uintptr(unsafe.Pointer(&client)))
	height, _, _ := send.Call(h, 0x1a1, 0, 0)
	visible := max(1, int(client.Bottom)/max(1, int(height)))
	atEnd := int(top)+visible >= len(*previous)-1
	selected := ""
	if !reset && int(oldSel) >= 0 && int(oldSel) < len(*previous) {
		selected = (*previous)[int(oldSel)]
	}
	send.Call(h, 0x0b, 0, 0)
	send.Call(h, 0x184, 0, 0)
	for _, r := range rows {
		send.Call(h, 0x180, 0, uintptr(unsafe.Pointer(wide(r))))
	}
	if selected != "" {
		if i := slices.Index(rows, selected); i >= 0 {
			send.Call(h, 0x186, uintptr(i), 0)
		}
	}
	if len(rows) > 0 {
		if tail && (reset || atEnd) {
			top = uintptr(max(0, len(rows)-visible))
		} else if reset {
			top = 0
		}
		send.Call(h, 0x197, top, 0)
	}
	send.Call(h, 0x0b, 1, 0)
	redraw(h)
	*previous = rows
}
func fileSize(n int64) string {
	switch {
	case n >= 1024*1024*1024:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1024*1024*1024))
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1f KiB", float64(n)/1024)
	default:
		return fmt.Sprintf("%d B", n)
	}
}
func isAttachment(m wechat.Message) bool {
	return m.Kind == "file" || m.Kind == "image" || m.Kind == "video" || m.Kind == "voice"
}
func (a *window) selectedMessage() (wechat.Message, bool) {
	i, _, _ := send.Call(a.children[idHistory], 0x188, 0, 0)
	if int(i) >= 0 && int(i) < len(a.messageIDs) {
		for _, m := range a.account.Messages {
			if m.ID == a.messageIDs[i] {
				return m, true
			}
		}
	}
	return wechat.Message{}, false
}
func (a *window) enableActions() {
	ac := a.account
	ready := ac.SignedIn && !ac.Busy
	enabled(a.children[idChoose], ready)
	enabled(a.children[idEditor], ready)
	enabled(a.children[idSend], ready && (len(ac.Files) > 0 || strings.TrimSpace(a.drafts[ac.ID]) != ""))
	enabled(a.children[idAdd], len(a.snap.Accounts) < workspaceTabs)
	i, _, _ := send.Call(a.children[idQueue], 0x188, 0, 0)
	enabled(a.children[idRemoveFile], ready && int(i) >= 0)
	m, ok := a.selectedMessage()
	enabled(a.children[idCopy], ok)
	enabled(a.children[idSave], ac.SignedIn && !ac.Downloading && ok && m.CanDownload())
	enabled(a.children[idStop], ac.Busy)
	enabled(a.children[idRefresh], ac.Status != "connecting")
}
