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
	if !ctrl {
		return false
	}
	switch m.W {
	case 9:
		for i, acct := range a.snap.Accounts {
			if acct.ID == a.snap.Active {
				a.switchTo(a.snap.Accounts[(i+1)%len(a.snap.Accounts)].ID)
				break
			}
		}
		return true
	case '1', '2', '3', '4':
		i := int(m.W - '1')
		if i < len(a.snap.Accounts) {
			a.switchTo(a.snap.Accounts[i].ID)
		}
		return true
	case 'T':
		if len(a.snap.Accounts) < workspaceTabs {
			a.command(idAdd)
		}
		return true
	case 'O':
		if t := a.tab(a.snap.Active); t.picker || t.kind != idWeChat {
			return false
		}
		if a.account.SignedIn {
			a.command(idChoose)
		}
		return true
	case 13:
		if t := a.tab(a.snap.Active); t.picker || t.kind != idWeChat {
			return false
		}
		if a.account.SignedIn && !a.account.Busy {
			a.command(idSend)
		}
		return true
	case 'V':
		if t := a.tab(a.snap.Active); t.picker || t.kind != idWeChat {
			return false
		}
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
	if a.serviceCommand(id) {
		return
	}
	ac := a.account
	var err error
	switch id {
	case idAdd:
		var next uint64
		next, err = a.c.AddAccount()
		if err == nil {
			a.startTab(next)
		}
	case idExit:
		a.requestExit()
		return
	case idMenu:
		a.menu()
		return
	case idLanguage:
		a.english = !a.english
		a.setFonts()
		a.refresh(true)
		return
	case idChat, idFiles, idTransfers, idToggleFilter:
		t := a.tab(a.snap.Active)
		if t.kind != idWeChat {
			return
		}
		if id == idToggleFilter {
			if t.view == idFiles {
				id = idChat
			} else {
				id = idFiles
			}
		}
		t.picker = false
		t.view = id
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
		dialog(a.h, "Ultra File "+desktop.Version+"\n\n"+a.tr("独立开源的多服务文件传输客户端。由社区维护，与腾讯无隶属关系。软件按 AGPL-3.0 提供，不附带担保。", "An independent open-source file transfer client. Community maintained and unaffiliated with Tencent. Licensed under AGPL-3.0, without warranty."), "Ultra File", 0x40)
		return
	case idRefresh:
		err = a.c.Start(ac.ID)
	case idSignout, idCloseTab:
		if dialog(a.h, a.tr("结束此账号会话？未发送的内容将被清除。", "End this session and clear its unsent items?"), "Ultra File", 0x124) != 6 {
			return
		}
		if id == idSignout {
			err = a.c.SignOut(ac.ID)
		} else {
			err = a.c.RemoveAccount(ac.ID)
			if err == nil {
				a.tab(ac.ID).close()
				delete(a.services.slots, ac.ID)
			}
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
	case idCover:
		if m, ok := a.selectedMessage(); ok && a.canPreview(m) {
			err = a.c.PreviewCover(ac.ID, m.ID)
		}
	case idCopy:
		if m, ok := a.selectedMessage(); ok {
			err = copyText(a.h, m.Text)
		}
	case idSave:
		m, ok := a.selectedMessage()
		if !ok || !m.CanDownload() {
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
	if m.CanDownload() {
		a.command(idSave)
		return
	}
	if isAttachment(m) {
		dialog(a.h, m.Text+"\n\n"+a.tr("此附件当前没有可用的下载凭据，请在手机微信中查看。", "No download credentials are available for this attachment. Open it in WeChat on your phone."), a.tr("附件详情", "Attachment details"), 0x40)
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
	add(idDownloads, a.tr("打开下载目录", "Open downloads"), true)
	add(idUpdate, a.tr("检查更新", "Check for updates"), true)
	separator()
	add(idDefaultService, a.tr("当前服务设为启动默认", "Use this service at startup"), true)
	add(idLanguage, a.tr("English", "简体中文"), true)
	add(idSignout, a.tr("退出此账号", "Sign out"), a.account.SignedIn)
	add(idCloseTab, a.tr("关闭此标签页", "Close account tab"), len(a.snap.Accounts) > 1)
	separator()
	add(idAbout, a.tr("关于", "About"), true)
	separator()
	add(idExit, a.tr("退出程序", "Quit Ultra File"), true)
	var r rect
	user32.NewProc("GetWindowRect").Call(a.children[idMenu], uintptr(unsafe.Pointer(&r)))
	choice, _, _ := user32.NewProc("TrackPopupMenu").Call(menu, 0x100|0x8|0x2, uintptr(r.Right), uintptr(r.Bottom), 0, a.h, 0)
	if choice != 0 {
		a.command(int(choice))
	}
}
func (a *window) stateLabel() string {
	if t := a.tab(a.snap.Active); t.picker {
		return a.tr("选择此标签的服务", "Choose a service for this tab")
	} else if t.kind != idWeChat {
		if s := t.servers[t.kind]; s != nil {
			return a.tr("局域网已启用", "LAN enabled") + " · " + s.URL
		}
		return a.tr("尚未启用局域网连接", "LAN transfer is off")
	}
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
		state := a.tr("已连接", "Connected")
		if ac.Name != "" {
			state += " · " + ac.Name
		}
		return state
	case "reconnecting":
		return a.tr("正在重新连接…", "Reconnecting…")
	case "expired":
		return a.tr("登录已过期", "Session expired")
	}
	return a.tr("未登录", "Signed out")
}
