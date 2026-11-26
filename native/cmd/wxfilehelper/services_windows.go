// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"context"
	"embed"
	"fmt"
	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/nearby"
	"github.com/skip2/go-qrcode"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"
)

const (
	idServices = 500 + iota
	idWeChat
	idNearby
	idLocal
	idQQ
	idDefaultService
	idHost
	idPair
	idPeer
	idNearSend
	idAccept
	idReject
)
const workspaceTabs = 4

//go:embed icons/*.png
var icons embed.FS

type serviceTab struct {
	kind    int
	picker  bool
	servers map[int]*nearby.Server
	pair    qrBitmap
	peer    string
	status  string
	running bool
	cancel  context.CancelFunc
}
type completion struct {
	tab  *serviceTab
	kind int
	err  error
}
type serviceSpace struct {
	slots     map[uint64]*serviceTab
	logos     map[int]*qrBitmap
	defaultID int
	completed chan completion
	done      chan struct{}
	once      sync.Once
}

func (a *window) initServices() {
	a.services = serviceSpace{slots: map[uint64]*serviceTab{}, logos: map[int]*qrBitmap{}, defaultID: idWeChat, completed: make(chan completion, 32), done: make(chan struct{})}
	if b, e := os.ReadFile(prefFile()); e == nil {
		for id, name := range serviceKeys {
			if string(b) == name && id != idQQ {
				a.services.defaultID = id
			}
		}
	}
	for id, name := range serviceKeys {
		if b, e := icons.ReadFile("icons/" + name + ".png"); e == nil {
			q := new(qrBitmap)
			q.load(b)
			a.services.logos[id] = q
		}
	}
}

var serviceKeys = map[int]string{idWeChat: "wechat", idNearby: "ultra", idLocal: "localsend", idQQ: "qq"}

func prefFile() string {
	d, e := os.UserConfigDir()
	if e != nil {
		return ""
	}
	return filepath.Join(d, "UltraFile", "startup-service")
}
func (a *window) tab(id uint64) *serviceTab {
	p := a.services.slots[id]
	if p == nil {
		p = &serviceTab{kind: a.services.defaultID, servers: map[int]*nearby.Server{}}
		a.services.slots[id] = p
	}
	return p
}
func (a *window) titleOf(id int) string {
	switch id {
	case idNearby:
		return a.tr("超级文件直传", "Ultra File LAN")
	case idLocal:
		return "LocalSend"
	case idQQ:
		return a.tr("QQ · 我的电脑", "QQ · My Computer")
	}
	return a.tr("微信文件传输助手", "WeChat File Transfer")
}
func (t *serviceTab) close() {
	if t.cancel != nil {
		t.cancel()
	}
	for _, s := range t.servers {
		s.Close()
	}
}
func (a *window) closeServices() {
	a.services.once.Do(func() {
		close(a.services.done)
		for _, t := range a.services.slots {
			t.close()
		}
	})
}
func (a *window) startTab(id uint64) {
	if a.tab(id).kind == idWeChat {
		a.fail(a.c.Start(id))
	}
}
func (a *window) serviceLayout(put positionControl, left, right, top, btn int) {
	t := a.tab(a.snap.Active)
	w := right - left
	if t.picker {
		for i, id := range []int{idWeChat, idNearby, idLocal, idQQ} {
			put(id, left, top+i*(btn+7), min(w, 460), btn)
			setText(a.children[id], a.titleOf(id))
			if id == idQQ {
				setText(a.children[id], a.titleOf(id)+a.tr("  ·  尚无接入接口", "  ·  Unavailable"))
			}
			enabled(a.children[id], id != idQQ)
		}
		return
	}
	s := t.servers[t.kind]
	heading := a.titleOf(t.kind)
	a.caption(heading, left, top, w, 30, a.brandFont, ink, singleLine)
	top += 34
	put(idDefaultService, right-146, top-34, 146, btn)
	setText(a.children[idDefaultService], a.tr("设为启动默认", "Use at startup"))
	if s == nil {
		put(idHost, left, top, 196, btn)
		setText(a.children[idHost], a.tr("启用局域网连接", "Enable LAN transfer"))
		return
	}
	if t.kind == idNearby {
		size := min(190, max(80, a.hgt-top-btn*2-36))
		a.qrBounds = a.box(left, top, size, size)
		a.caption(a.tr("用超级文件安卓版扫描配对", "Scan in Ultra File for Android"), left+size+10, top, w-size-10, 48, a.font, ink, 0x10)
		put(idPair, left+size+10, top+53, min(w-size-10, 146), btn)
		setText(a.children[idPair], a.tr("复制配对信息", "Copy pairing link"))
		top += size + 7
	} else {
		a.caption(s.URL, left, top, w, 23, a.font, ink, singleLine)
		top += 27
		put(idPeer, left, top, min(w-160, 270), btn)
		top += btn + 7
	}
	put(idNearSend, left, top, 169, btn)
	setText(a.children[idNearSend], a.tr("选择发送文件…", "Choose a file…"))
	enabled(a.children[idNearSend], !t.running)
	top += btn + 4
	status := t.status
	if status == "" {
		status = s.Status()
	}
	if p := s.Pending(); p != "" {
		a.caption(p, left, top, w, 23, a.font, ink, singleLine)
		top += 24
		put(idAccept, left, top, 85, btn)
		put(idReject, left+92, top, 85, btn)
		setText(a.children[idAccept], a.tr("接收", "Accept"))
		setText(a.children[idReject], a.tr("拒绝", "Decline"))
	} else {
		a.caption(status, left, top, w, 32, a.smallFont, muted, singleLine)
	}
}
func (a *window) pollServices() {
	for _, account := range a.snap.Accounts {
		for kind, server := range a.tab(account.ID).servers {
			files := server.Receipts()
			if len(files) == 0 || !a.tray.enabled || !a.isBackground() {
				continue
			}
			route, ok := a.ensureIcon(account)
			if !ok {
				continue
			}
			n := a.iconData(route.iconID)
			n.Flags, n.InfoFlags = 0x10, 0x81
			setShellText(n.Title[:], a.tr("收到文件", "File received")+" · "+a.titleOf(kind))
			setShellText(n.Info[:], strings.Join(files, ", "))
			shellNotify(1, &n)
		}
	}
	select {
	case e := <-a.services.completed:
		e.tab.running = false
		if e.err != nil {
			e.tab.status = e.err.Error()
		} else {
			e.tab.status = ""
		}
		a.refresh(true)
	default:
	}
	t := a.tab(a.snap.Active)
	if s := t.servers[t.kind]; s != nil && !t.picker {
		current := s.Status() + s.Pending() + t.status
		if current != a.lastServiceStatus {
			a.lastServiceStatus = current
			a.layout()
			redraw(a.h)
		}
	}
}
func (a *window) serviceCommand(id int) bool {
	t := a.tab(a.snap.Active)
	switch id {
	case idServices:
		t.picker = !t.picker
	case idWeChat, idNearby, idLocal:
		t.kind = id
		t.picker = false
		t.status = ""
		if id == idWeChat && a.account.Status == "idle" {
			a.startTab(a.snap.Active)
		}
	case idDefaultService:
		path := prefFile()
		err := os.MkdirAll(filepath.Dir(path), 0700)
		if err == nil {
			err = os.WriteFile(path, []byte(serviceKeys[t.kind]), 0600)
		}
		if err == nil {
			a.services.defaultID = t.kind
		}
		a.fail(err)
	case idHost:
		addrs := nearby.Addresses()
		if len(addrs) == 0 {
			a.fail(fmt.Errorf("未找到局域网 IPv4 地址 / No LAN IPv4 address"))
			return true
		}
		ip := addrs[0]
		if len(addrs) > 1 {
			menu, _, _ := user32.NewProc("CreatePopupMenu").Call()
			defer user32.NewProc("DestroyMenu").Call(menu)
			for i, v := range addrs {
				user32.NewProc("AppendMenuW").Call(menu, 0, uintptr(i+1), uintptr(unsafe.Pointer(wide(v))))
			}
			var pt point
			user32.NewProc("GetCursorPos").Call(uintptr(unsafe.Pointer(&pt)))
			n, _, _ := user32.NewProc("TrackPopupMenu").Call(menu, 0x102, uintptr(pt.X), uintptr(pt.Y), 0, a.h, 0)
			if n < 1 || int(n) > len(addrs) {
				return true
			}
			ip = addrs[n-1]
		}
		s, e := nearby.Start(ip, t.kind == idLocal, filepath.Join(downloadsDirectory(), "Ultra File"))
		if e != nil {
			a.fail(e)
			return true
		}
		t.servers[t.kind] = s
		if t.kind == idNearby {
			b, _ := qrcode.Encode(s.Pair(), qrcode.Medium, 320)
			t.pair.load(b)
		}
	case idPair:
		if s := t.servers[idNearby]; s != nil {
			a.fail(copyText(a.h, s.Pair()))
		}
	case idNearSend:
		s := t.servers[t.kind]
		if s == nil || t.running {
			return true
		}
		addr := strings.TrimSpace(getText(a.children[idPeer]))
		paths, e := fileDialog(a.h, a.tr("选择一个文件", "Choose one file"), "", a.lastDir, false)
		if e != nil {
			a.fail(e)
			return true
		}
		if len(paths) == 0 {
			return true
		}
		if len(paths) != 1 {
			a.fail(fmt.Errorf("每次选择一个文件 / Select one file at a time"))
			return true
		}
		t.running = true
		t.status = a.tr("处理中…", "Working…")
		ctx, cancel := context.WithCancel(context.Background())
		t.cancel = cancel
		kind := t.kind
		hwnd := a.h
		go func() {
			defer cancel()
			var err error
			if kind == idNearby {
				err = s.Share(paths[0])
			} else {
				err = s.SendLocal(ctx, addr, paths[0], func(pin string) bool {
					return dialog(hwnd, "接收设备 / Receiver: "+addr+"\nSHA-256: "+pin+"\n发送文件 / Send file?", "LocalSend", 0x124) == 6
				})
			}
			select {
			case a.services.completed <- completion{t, kind, err}:
			case <-a.services.done:
			}
		}()
	case idAccept, idReject:
		if s := t.servers[idLocal]; s != nil {
			s.Decide(id == idAccept)
		}
	default:
		return false
	}
	a.refresh(true)
	return true
}
