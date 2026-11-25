// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"unsafe"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/desktop"
	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
)

type pendingDraft struct {
	text  string
	known map[string]bool
}

func (a *window) reconcileDraft(v desktop.Account) {
	if epoch, ok := a.epochs[v.ID]; ok && epoch != v.DraftEpoch {
		delete(a.drafts, v.ID)
		delete(a.pending, v.ID)
	}
	a.epochs[v.ID] = v.DraftEpoch
	p, ok := a.pending[v.ID]
	if !ok {
		return
	}
	for _, m := range v.Messages {
		if m.Outgoing && m.Kind == "text" && m.Text == p.text && !p.known[m.ID] {
			if a.drafts[v.ID] == p.text {
				delete(a.drafts, v.ID)
			}
			delete(a.pending, v.ID)
			return
		}
	}
	if !v.Busy {
		delete(a.pending, v.ID)
	}
}
func (a *window) key(m message) bool {
	state, _, _ := user32.NewProc("GetKeyState").Call(0x11)
	ctrl := state&0x8000 != 0
	if m.W == 0x7a || (m.W == 0x1b && a.room) {
		a.toggleClassroom()
		return true
	}
	if !ctrl {
		return false
	}
	switch m.W {
	case 9:
		for _, acct := range a.snap.Accounts {
			if acct.ID != a.snap.Active {
				a.switchTo(acct.ID)
				break
			}
		}
		return true
	case '1', '2':
		i := int(m.W - '1')
		if i < len(a.snap.Accounts) {
			a.switchTo(a.snap.Accounts[i].ID)
		}
		return true
	case 'T':
		if len(a.snap.Accounts) < 2 {
			a.command(idAdd)
		}
		return true
	case 'O':
		if a.account.SignedIn {
			a.command(idChoose)
		}
		return true
	case 13:
		if a.account.SignedIn && !a.account.Busy {
			a.command(idSend)
		}
		return true
	case 'V':
		paths, err := clipboardFiles(a.h)
		if err != nil {
			a.fail(err)
			return true
		}
		if len(paths) > 0 {
			a.fail(a.c.Queue(a.snap.Active, paths))
			a.refresh(false)
			return true
		}
	case 'C':
		if m.Window == a.children[idHistory] {
			a.command(idCopy)
			return true
		}
	}
	return false
}
func (a *window) command(id int) {
	ac := a.account
	var err error
	switch id {
	case idAdd:
		var next uint64
		next, err = a.c.AddAccount()
		if err == nil {
			err = a.c.Start(next)
		}
	case idClass:
		a.toggleClassroom()
		return
	case idMenu:
		a.menu()
		return
	case idLanguage:
		a.english = !a.english
		a.refresh(true)
		return
	case idChat, idFiles, idToggleFilter:
		a.filesOnly = id == idFiles || (id == idToggleFilter && !a.filesOnly)
		a.refresh(true)
		return
	case idDownloads:
		err = openTarget(a.h, downloadsDirectory())
	case idUpdate:
		a.manualUpdate = true
		a.c.CheckUpdate()
	case idNotice:
		if a.snap.Update != nil {
			err = openTarget(a.h, a.snap.Update.URL)
		}
	case idAbout:
		dialog(a.h, "WX FileHelper "+desktop.Version+"\n\n"+a.tr("独立开源的微信文件传输客户端。由社区维护，与腾讯无隶属关系。软件按 AGPL-3.0 提供，不附带担保。", "An independent open-source client for WeChat File Transfer. Community maintained and unaffiliated with Tencent. Licensed under AGPL-3.0, without warranty."), "WX FileHelper", 0x40)
		return
	case idRefresh:
		err = a.c.Start(ac.ID)
	case idSignout, idCloseTab:
		if dialog(a.h, a.tr("结束此账号会话？未发送的内容将被清除。", "End this session and clear its unsent items?"), "WX FileHelper", 0x124) != 6 {
			return
		}
		if id == idSignout {
			err = a.c.SignOut(ac.ID)
		} else {
			err = a.c.RemoveAccount(ac.ID)
		}
		if err == nil {
			delete(a.drafts, ac.ID)
			delete(a.pending, ac.ID)
			delete(a.epochs, ac.ID)
		}
	case idChoose:
		var paths []string
		paths, err = fileDialog(a.h, a.tr("选择要发送的文件", "Choose files to send"), "", a.lastDir, false)
		if err == nil && len(paths) > 0 {
			a.lastDir = filepath.Dir(paths[0])
			err = a.c.Queue(ac.ID, paths)
		}
	case idRemoveFile:
		i, _, _ := send.Call(a.children[idQueue], 0x188, 0, 0)
		if int(i) >= 0 {
			err = a.c.RemoveFile(ac.ID, int(i))
		}
	case idCopy:
		if m, ok := a.selectedMessage(); ok {
			err = copyText(a.h, m.Text)
		}
	case idSave:
		m, ok := a.selectedMessage()
		if !ok || !isAttachment(m) || m.Kind == "voice" {
			return
		}
		var paths []string
		paths, err = fileDialog(a.h, a.tr("保存文件", "Save file"), wechat.SafeFilename(m.Text), downloadsDirectory(), true)
		if err == nil && len(paths) > 0 {
			err = a.c.Download(ac.ID, m.ID, paths[0])
		}
	case idStop:
		err = a.c.StopSending(ac.ID)
	case idSend:
		draft := getText(a.children[idEditor])
		if strings.TrimSpace(draft) == "" && len(ac.Files) == 0 {
			return
		}
		err = a.c.Send(ac.ID, draft)
		if err == nil && strings.TrimSpace(draft) != "" {
			known := map[string]bool{}
			for _, m := range ac.Messages {
				if m.Outgoing && m.Kind == "text" && m.Text == draft {
					known[m.ID] = true
				}
			}
			a.pending[ac.ID] = pendingDraft{draft, known}
		}
	}
	a.fail(err)
	a.refresh(false)
}
func (a *window) openMessage() {
	m, ok := a.selectedMessage()
	if !ok {
		return
	}
	if isAttachment(m) && m.Kind != "voice" {
		a.command(idSave)
		return
	}
	dialog(a.h, m.Text, a.tr("消息内容", "Message text"), 0x40)
}
func (a *window) menu() {
	menu, _, _ := user32.NewProc("CreatePopupMenu").Call()
	if menu == 0 {
		return
	}
	defer user32.NewProc("DestroyMenu").Call(menu)
	add := func(id int, label string, allowed bool) {
		flags := uintptr(0)
		if !allowed {
			flags = 1
		}
		user32.NewProc("AppendMenuW").Call(menu, flags, uintptr(id), uintptr(unsafe.Pointer(wide(label))))
	}
	separator := func() { user32.NewProc("AppendMenuW").Call(menu, 0x800, 0, 0) }
	if !a.rail {
		label := a.tr("仅显示文件", "Show files only")
		if a.filesOnly {
			label = a.tr("显示全部消息", "Show all messages")
		}
		add(idToggleFilter, label, true)
		add(idDownloads, a.tr("打开下载目录", "Open downloads"), true)
		add(idUpdate, a.tr("检查更新", "Check for updates"), true)
		separator()
	}
	add(idLanguage, a.tr("English", "简体中文"), true)
	add(idSignout, a.tr("退出此账号", "Sign out"), a.account.SignedIn)
	add(idCloseTab, a.tr("关闭此标签页", "Close account tab"), len(a.snap.Accounts) > 1)
	if !a.rail {
		separator()
		add(idAbout, a.tr("关于", "About"), true)
	}
	var r rect
	user32.NewProc("GetWindowRect").Call(a.children[idMenu], uintptr(unsafe.Pointer(&r)))
	choice, _, _ := user32.NewProc("TrackPopupMenu").Call(menu, 0x100|0x8|0x2, uintptr(r.Right), uintptr(r.Bottom), 0, a.h, 0)
	if choice != 0 {
		a.command(int(choice))
	}
}
func (a *window) toggleClassroom() {
	// GWL_STYLE is a signed argument. ^uintptr(15) encodes -16 on x64.
	if !a.room {
		mon, _, _ := user32.NewProc("MonitorFromWindow").Call(a.h, 2)
		mi := monitorInfo{}
		mi.Size = uint32(unsafe.Sizeof(mi))
		if ok, _, _ := user32.NewProc("GetMonitorInfoW").Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
			return
		}
		user32.NewProc("GetWindowRect").Call(a.h, uintptr(unsafe.Pointer(&a.savedRect)))
		a.savedStyle, _, _ = user32.NewProc("GetWindowLongPtrW").Call(a.h, ^uintptr(15))
		a.room = true
		user32.NewProc("SetWindowLongPtrW").Call(a.h, ^uintptr(15), (a.savedStyle&^0xcf0000)|0x80000000)
		r := mi.Monitor
		user32.NewProc("SetWindowPos").Call(a.h, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0x34)
	} else {
		a.room = false
		user32.NewProc("SetWindowLongPtrW").Call(a.h, ^uintptr(15), a.savedStyle)
		r := a.savedRect
		user32.NewProc("SetWindowPos").Call(a.h, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0x34)
	}
	a.setFonts()
	a.refresh(true)
}
func (a *window) stateLabel() string {
	ac := a.account
	if ac.Error != "" {
		return ac.Error
	}
	if ac.Downloading {
		return a.tr("正在保存文件…", "Saving file…")
	}
	if ac.Busy {
		if ac.Total > 0 {
			return fmt.Sprintf(a.tr("正在发送  %d%%   ·   %s / %s", "Sending  %d%%   ·   %s / %s"), ac.Sent*100/ac.Total, fileSize(ac.Sent), fileSize(ac.Total))
		}
		return a.tr("正在发送…", "Sending…")
	}
	switch ac.Status {
	case "connecting":
		return a.tr("正在连接微信…", "Connecting to WeChat…")
	case "scan":
		return a.tr("等待手机扫码", "Waiting for scan")
	case "scanned":
		return a.tr("请在手机微信上确认", "Confirm in WeChat on your phone")
	case "online":
		return a.tr("已连接", "Connected")
	case "reconnecting":
		return a.tr("正在重新连接…", "Reconnecting…")
	case "expired":
		return a.tr("登录已过期", "Session expired")
	}
	return a.tr("未登录", "Signed out")
}
