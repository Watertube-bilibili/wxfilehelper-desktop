// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
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
	idDefaultService
	idHost
	idPair
	idPeer
	idNearSend
	idAccept
	idReject
)
const workspaceTabs = 4

//go:embed icons/ultra.png icons/localsend.png icons/wechat.png
var icons embed.FS

type serviceTab struct {
	direct  directState
	kind    int
	view    int
	picker  bool
	servers map[int]*nearby.Server
	pair    qrBitmap
	peer    string
	status  string
	running bool
}
type completion struct {
	tab  *serviceTab
	kind int
	err  error
}
type serviceSpace struct {
	slots     map[uint64]*serviceTab
	logos     map[int]*serviceIcon
	defaultID int
	completed chan completion
	done      chan struct{}
	once      sync.Once
}

func (a *window) initServices() {
	a.services = serviceSpace{slots: map[uint64]*serviceTab{}, logos: map[int]*serviceIcon{}, defaultID: idWeChat, completed: make(chan completion, 32), done: make(chan struct{})}
	if b, e := os.ReadFile(prefFile()); e == nil {
		for id, name := range serviceKeys {
			if string(b) == name {
				a.services.defaultID = id
			}
		}
	}
	for id, name := range serviceKeys {
		if b, e := icons.ReadFile("icons/" + name + ".png"); e == nil {
			q := new(serviceIcon)
			q.load(b)
			a.services.logos[id] = q
		}
	}
}

var serviceKeys = map[int]string{idWeChat: "wechat", idNearby: "ultra", idLocal: "localsend"}

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
		p = &serviceTab{kind: a.services.defaultID, view: idChat, servers: map[int]*nearby.Server{}}
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
	}
	return a.tr("微信文件传输助手", "WeChat File Transfer")
}
func (t *serviceTab) close() {
	if t.direct.session != nil {
		t.direct.session.Close()
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
	if a.tab(id).kind == idLocal {
		a.maybeOfficial()
	}
	if a.tab(id).kind == idWeChat {
		a.fail(a.c.Start(id))
	}
}
func (a *window) serviceLayout(put positionControl, left, right, top, btn int) {
	t := a.tab(a.snap.Active)
	w := right - left
	if t.picker {
		for i, id := range []int{idWeChat, idNearby, idLocal} {
			put(id, left, top+i*(btn+7), min(w, 460), btn)
			setText(a.children[id], a.titleOf(id))
		}
		return
	}
	if t.kind == idLocal {
		a.officialLayout(put, left, right, top, btn)
		return
	}
	a.directLayout(put, left, right, top, btn)
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
	if a.directCommand(id) {
		return true
	}
	t := a.tab(a.snap.Active)
	switch id {
	case idNavWeChat, idNavNearby, idNavLocal:
		kind := navService(id)
		if kind != t.kind {
			for _, ac := range a.snap.Accounts {
				if other := a.tab(ac.ID); other.kind == kind {
					other.picker = false
					a.switchTo(ac.ID)
					return true
				}
			}
			if len(a.snap.Accounts) < workspaceTabs {
				next, err := a.c.AddAccount()
				if err != nil {
					a.fail(err)
					return true
				}
				t = a.tab(next)
				t.kind = kind
				a.startTab(next)
			} else {
				t.picker = true
				a.refresh(true)
				return true
			}
		}
		t.picker = false
	case idPickOfficial:
		paths, err := fileDialog(a.h, a.tr("选择官方便携版的 localsend_app.exe", "Select the official portable localsend_app.exe"), "localsend_app.exe", a.lastDir, false)
		if err != nil {
			a.fail(err)
			return true
		}
		if len(paths) == 1 {
			a.selectOfficial(paths[0])
		}
	case idGetZIP:
		a.fail(openTarget(a.h, "https://github.com/localsend/localsend/releases/download/v1.19.0/LocalSend-1.19.0-windows-x86-64.zip"))
	case idOfficial:
		a.startOfficial()
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
		s, e := nearby.Start(ip, false, filepath.Join(downloadsDirectory(), "Ultra File"))
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
		kind := t.kind
		go func() {
			err := s.Share(paths[0])
			select {
			case a.services.completed <- completion{t, kind, err}:
			case <-a.services.done:
			}
		}()
	default:
		return false
	}
	if t.kind == idLocal && !t.picker {
		a.maybeOfficial()
	}
	a.refresh(true)
	return true
}
