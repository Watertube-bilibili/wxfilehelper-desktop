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
	"unsafe"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/desktop"
	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
)

var (
	base       = color(252, 253, 249)
	railBG     = color(242, 245, 239)
	border     = color(220, 227, 219)
	ink        = color(36, 49, 46)
	muted      = color(88, 103, 97)
	accent     = color(37, 105, 83)
	accentWash = color(231, 241, 233)
	tabBG      = color(239, 242, 236)
	white      = color(255, 255, 255)
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
			r, g, b, alpha := img.At(img.Bounds().Min.X+x, img.Bounds().Min.Y+y).RGBA()
			i := 4 * (y*q.width + x)
			q.pixels[i] = byte((b + 65535 - alpha) >> 8)
			q.pixels[i+1] = byte((g + 65535 - alpha) >> 8)
			q.pixels[i+2] = byte((r + 65535 - alpha) >> 8)
		}
	}
}
func (a *window) createChildren() error {
	for _, id := range []int{idTabs, idAdd, idMenu, idChat, idFiles, idDownloads, idUpdate, idAbout, idRefresh, idHistory, idSave, idCopy, idQueue, idRemoveFile, idEditor, idChoose, idSend, idStop, idNotice, idServices, idWeChat, idNearby, idLocal, idDefaultService, idHost, idPair, idPeer, idNearSend, idTransfers, idCover, idNavWeChat, idNavNearby, idNavLocal, idOfficial, idLANMode, idUDPMode, idWeb, idRemoteListen, idRemoteSend, idRemoteStop, idRemoteCopy, idRemoteAccept} {
		class := "BUTTON"
		style := uintptr(0x40010000 | 0x0000000b)
		ex := uintptr(0) // WS_CHILD | WS_TABSTOP, owner drawn button
		switch id {
		case idTabs:
			class = "SysTabControl32"
			// Multiline mode suppresses the system up/down spinner. Our two fixed
			// slots fit in one row. The native tab retains its full input behavior.
			style = 0x40010000 | 0x0400 | 0x2000 | 0x200 | 0x800
		case idHistory:
			class = "LISTBOX"
			style = 0x40210000 | 0x101 | 0x40 | 0x20 // LBS_OWNERDRAWVARIABLE
		case idQueue:
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
		if class == "BUTTON" {
			comctl32.NewProc("SetWindowSubclass").Call(child, hoverCallback, 2, uintptr(id))
		}
		if id == idEditor {
			if ok, _, err := comctl32.NewProc("SetWindowSubclass").Call(child, editorCallback, 1, 0); ok == 0 {
				return fmt.Errorf("Initialize composer: %w", err)
			}
		}
		if id == idTabs {
			if ok, _, err := comctl32.NewProc("SetWindowSubclass").Call(child, tabCallback, 1, 0); ok == 0 {
				return fmt.Errorf("Initialize account tabs: %w", err)
			}
		}
	}
	a.createTooltips()
	send.Call(a.children[idPeer], 0x1501, 0, uintptr(unsafe.Pointer(wide("abcd efgh jkmn"))))
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
	face := a.tr("Microsoft YaHei UI", "Segoe UI")
	a.font = createFont(size, 400, a.dpi, face)
	a.smallFont = createFont(size-2, 400, a.dpi, face)
	a.brandFont = createFont(size+1, 600, a.dpi, face)
	a.titleFont = createFont(size+8, 600, a.dpi, face)
	for _, h := range a.children {
		send.Call(h, 0x30, a.font, 1)
	}
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
		idLANMode: a.tr("同一 Wi-Fi", "Same Wi-Fi"), idUDPMode: a.tr("远程 UDP", "Remote UDP"), idWeb: a.tr("网页版", "Web app"), idRemoteListen: a.tr("生成接收码", "Create pairing code"), idRemoteSend: a.tr("发送文件…", "Send file…"), idRemoteStop: a.tr("取消", "Cancel"), idRemoteCopy: a.tr("复制配对码", "Copy code"), idRemoteAccept: a.tr("接收并保存…", "Receive and save…"), idServices: a.tr("切换服务", "Services"), idAdd: a.tr("新建标签", "New tab"), idMenu: a.tr("更多操作", "More actions"), idChat: a.tr("会话记录", "Conversation"), idFiles: a.tr("文件收件箱", "File inbox"), idTransfers: a.tr("传输队列", "Transfer queue"),
		idNavWeChat: a.tr("微信", "WeChat"), idNavNearby: a.tr("超级文件直传", "Ultra File LAN"), idNavLocal: "LocalSend", idCover: a.tr("预览封面", "Preview cover"),
		idDownloads: a.tr("下载目录", "Downloads"), idUpdate: a.tr("检查更新", "Check updates"), idAbout: a.tr("关于", "About"),
		idRefresh: a.tr("刷新二维码", "Refresh code"), idSave: a.tr("另存为…", "Save as…"), idCopy: a.tr("复制", "Copy"), idRemoveFile: a.tr("移除", "Remove"),
		idChoose: a.tr("添加文件", "Attach files"), idSend: a.tr("发送", "Send"), idStop: a.tr("停止队列", "Stop queue"),
	} {
		setText(a.children[id], s)
	}
	if a.tab(a.snap.Active).picker {
		setText(a.children[idServices], a.tr("返回", "Back"))
	}
	if a.snap.Update != nil {
		setText(a.children[idNotice], a.tr("可更新至 ", "Update to ")+a.snap.Update.Version+"  ↗")
	}
	setText(a.h, a.tr("超级文件", "Ultra File"))
	a.labelTooltips()
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
	if previous := a.services.slots[a.account.ID]; switched && previous != nil {
		previous.peer = getText(a.children[idPeer])
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
	a.filesOnly = a.tab(s.Active).view == idFiles
	if switched || getText(a.children[idEditor]) != a.drafts[s.Active] {
		setText(a.children[idEditor], a.drafts[s.Active])
	}
	if switched {
		a.coverImages = nil
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
	a.updateTimeline(switched)
	var rows []string
	for _, f := range a.account.Files {
		rows = append(rows, f.Name+"    ·    "+fileSize(f.Size))
	}
	a.updateList(idQueue, rows, &a.queueRows, false, switched)
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
	if _, visible := a.placements[id]; !visible {
		showWindow.Call(h, 0)
	}
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
	enabled(a.children[idCover], ac.SignedIn && ac.Previewing == "" && ok && a.canPreview(m))
	enabled(a.children[idSave], ac.SignedIn && !ac.Downloading && ok && m.CanDownload())
	enabled(a.children[idStop], ac.Busy)
	enabled(a.children[idRefresh], ac.Status != "connecting")
}
