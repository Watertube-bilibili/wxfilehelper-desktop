// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"fmt"
	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/remote"
	"os"
	"path/filepath"
	"strings"
)

const webURL = "https://watertube-bilibili.github.io/wxfilehelper-desktop/"

type directState struct {
	remoteMode bool
	session    *remote.Session
	last       remote.Status
}

func (a *window) remoteText(st remote.Status) string {
	switch st.State {
	case "signaling":
		return a.tr("正在连接配对服务…", "Contacting pairing service…")
	case "waiting":
		return a.tr("等待发送方输入配对码", "Waiting for the sender")
	case "connecting", "connected":
		return a.tr("正在协商 UDP 直连…", "Connecting over UDP…")
	case "offered":
		return a.tr("等待对方接收", "Waiting for acceptance")
	case "incoming":
		return a.tr("有文件等待接收", "Incoming file")
	case "sending":
		return a.tr("正在发送", "Sending")
	case "receiving":
		return a.tr("正在接收", "Receiving")
	case "confirming":
		return a.tr("等待对方校验…", "Waiting for verification…")
	case "complete":
		return a.tr("已完成 · SHA-256 校验通过", "Complete · SHA-256 verified")
	case "cancelled":
		return a.tr("已取消", "Cancelled")
	case "error":
		return st.Error
	}
	return a.tr("UDP 直连 · 16 KiB 分块", "Direct UDP · 16 KiB chunks")
}
func (a *window) directLayout(put positionControl, left, right, top, btn int) {
	t := a.tab(a.snap.Active)
	d := &t.direct
	w := right - left
	compact := a.hgt < 505
	seg := min(127, (w-8)/2)
	put(idLANMode, left, top, seg, btn)
	put(idUDPMode, left+seg+4, top, seg, btn)
	if w > 365 {
		put(idWeb, right-94, top, 94, btn)
	}
	top += btn + 8
	hero := 82
	if compact {
		hero = 50
	}
	a.directHero = a.box(left, top, w, hero)
	side := 62
	if compact {
		side = 43
	}
	title := a.tr("同一 Wi-Fi 直传", "On the same Wi-Fi")
	detail := a.tr("用超级文件安卓版扫码配对", "Pair with the Ultra File Android app")
	if d.remoteMode {
		title = a.tr("远程文件直传", "Remote file transfer")
		detail = a.tr("用配对码连接电脑或浏览器", "Pair with another PC or browser")
	}
	a.caption(title, left+side+15, top+3, w-side-24, hero/2+5, a.brandFont, ink, singleLine)
	a.caption(detail, left+side+15, top+hero/2, w-side-24, hero/2-2, a.smallFont, muted, singleLine)
	top += hero + 9
	wide := w >= 540 && a.hgt-top >= 172
	cardWidth := (w - 12) / 2
	card := func(x int, heading, helper string) {
		a.directCards = append(a.directCards, a.box(x, top, cardWidth, 158))
		a.caption(heading, x+17, top+12, cardWidth-34, 30, a.brandFont, ink, singleLine)
		a.caption(helper, x+17, top+49, cardWidth-34, 46, a.smallFont, muted, 0x10)
	}
	if !d.remoteMode {
		s := t.servers[idNearby]
		if s == nil {
			if wide {
				x := left + cardWidth + 12
				card(left, a.tr("配对手机", "Pair a phone"), a.tr("开启连接后，用安卓端扫描二维码。", "Enable the connection and scan its QR on Android."))
				card(x, a.tr("收到的文件", "Received files"), a.tr("局域网收件保存在下载目录的 Ultra File 文件夹。", "LAN files are saved in Downloads / Ultra File."))
				put(idHost, left+17, top+108, min(cardWidth-34, 182), btn)
				setText(a.children[idHost], a.tr("启用局域网连接", "Enable LAN transfer"))
				put(idLANFolder, x+17, top+108, min(cardWidth-34, 177), btn)
				return
			}
			a.caption(a.tr("配对后可以双向传文件。", "Pair your phone to send and receive files."), left+3, top, w-6, 28, a.font, ink, singleLine)
			put(idHost, left, top+33, min(196, w), btn)
			setText(a.children[idHost], a.tr("启用局域网连接", "Enable LAN transfer"))
			return
		}
		size := min(182, max(65, a.hgt-top-14))
		a.qrCard = a.box(left, top, size, size)
		a.qrBounds = a.box(left+6, top+6, size-12, size-12)
		x := left + size + 14
		rx := right - x
		put(idPair, x, top, min(rx, 166), btn)
		setText(a.children[idPair], a.tr("复制配对信息", "Copy pairing link"))
		put(idNearSend, x, top+btn+6, min(rx, 166), btn)
		setText(a.children[idNearSend], a.tr("选择发送文件…", "Share a file…"))
		enabled(a.children[idNearSend], !t.running)
		state := s.Status()
		if t.status != "" {
			state = t.status
		}
		a.caption(state, x, top+2*(btn+6), rx, 48, a.smallFont, muted, 0x10)
		return
	}
	st := d.last
	if st.State == "" || st.State == "error" || st.State == "complete" || st.State == "cancelled" {
		if st.State != "" {
			a.caption(a.remoteText(st), left+2, top, w-4, 26, a.smallFont, muted, singleLine)
			top += 28
		}
		if wide {
			x := left + cardWidth + 12
			card(left, a.tr("在此设备接收", "Receive on this device"), a.tr("生成配对码后发给对方，收到文件邀请时确认。", "Share a pairing code, then accept the incoming file."))
			card(x, a.tr("发往其他设备", "Send to another device"), a.tr("输入接收方的 12 位配对码。", "Enter the receiver’s 12-character code."))
			put(idRemoteListen, left+17, top+108, min(cardWidth-34, 179), btn)
			put(idPeer, x+17, top+72, cardWidth-34, 28)
			put(idRemoteSend, x+17, top+110, min(145, cardWidth-34), btn)
			return
		}
		put(idRemoteListen, left, top, min(185, w), btn)
		top += btn + 8
		a.caption(a.tr("对方的接收码", "Receiver’s pairing code"), left+2, top, w, 22, a.smallFont, muted, singleLine)
		top += 23
		sendWidth := min(139, w/3)
		put(idPeer, left, top, w-sendWidth-7, btn)
		put(idRemoteSend, right-sendWidth, top, sendWidth, btn)
		return
	}
	a.caption(a.remoteText(st), left+2, top, w-100, btn, a.font, ink, singleLine)
	put(idRemoteStop, right-82, top, 82, btn)
	top += btn + 6
	if st.State == "waiting" {
		c := strings.ToUpper(st.Code)
		if len(c) == 12 {
			c = c[:4] + "  " + c[4:8] + "  " + c[8:]
		}
		a.caption(c, left+2, top, w-112, 36, a.titleFont, accent, singleLine)
		put(idRemoteCopy, right-102, top, 102, btn)
	} else if st.Name != "" {
		a.caption(st.Name+"  ·  "+fileSize(st.Size), left+2, top, w-4, 27, a.brandFont, ink, singleLine)
		top += 29
		if st.State == "incoming" {
			put(idRemoteAccept, left, top, min(173, w), btn)
		} else {
			a.directProgress = a.box(left+2, top+29, w-4, 5)
			a.caption(fmt.Sprintf("%s / %s", fileSize(st.Done), fileSize(st.Size)), left+2, top, w-4, 24, a.smallFont, muted, singleLine)
		}
	}
}
func (a *window) directCommand(id int) bool {
	if id < idLANMode || id > idLANFolder {
		return false
	}
	t := a.tab(a.snap.Active)
	d := &t.direct
	switch id {
	case idLANMode:
		d.remoteMode = false
	case idUDPMode:
		d.remoteMode = true
	case idLANFolder:
		folder := filepath.Join(downloadsDirectory(), "Ultra File")
		e := os.MkdirAll(folder, 0700)
		if e == nil {
			e = openTarget(a.h, folder)
		}
		a.fail(e)
	case idWeb:
		a.fail(openTarget(a.h, webURL))
	case idRemoteStop:
		if d.session != nil {
			d.session.Close()
		}
	case idRemoteCopy:
		if d.last.Code != "" {
			a.fail(copyText(a.h, d.last.Code))
		}
	case idRemoteListen, idRemoteSend:
		path, code := "", ""
		if id == idRemoteSend {
			code = getText(a.children[idPeer])
			paths, e := fileDialog(a.h, a.tr("选择一个文件", "Choose one file"), "", a.lastDir, false)
			if e != nil {
				a.fail(e)
				return true
			}
			if len(paths) != 1 {
				return true
			}
			path = paths[0]
			a.lastDir = filepath.Dir(path)
		}
		if d.session != nil {
			d.session.Close()
		}
		s, e := remote.Start(code, path)
		if e != nil {
			a.fail(e)
			return true
		}
		d.session = s
		d.last = s.Snapshot()
	case idRemoteAccept:
		if d.session != nil {
			paths, e := fileDialog(a.h, a.tr("接收并保存文件", "Receive and save file"), d.last.Name, downloadsDirectory(), true)
			if e != nil {
				a.fail(e)
				return true
			}
			if len(paths) > 0 {
				a.fail(d.session.Accept(paths[0]))
			}
		}
	}
	a.refresh(true)
	return true
}
func (a *window) pollDirect() {
	for _, ac := range a.snap.Accounts {
		t := a.tab(ac.ID)
		d := &t.direct
		if d.session == nil {
			continue
		}
		st := d.session.Snapshot()
		if st == d.last {
			continue
		}
		if st.Saved != "" && d.last.Saved == "" && a.tray.enabled && a.isBackground() {
			if route, ok := a.ensureIcon(ac); ok {
				n := a.iconData(route.iconID)
				n.Flags, n.InfoFlags = 0x10, 0x81
				setShellText(n.Title[:], a.tr("远程文件已收到", "Remote file received"))
				setShellText(n.Info[:], st.Name)
				shellNotify(1, &n)
			}
		}
		d.last = st
		if ac.ID == a.snap.Active && t.kind == idNearby && !t.picker {
			a.layout()
			redraw(a.h)
		}
	}
}
