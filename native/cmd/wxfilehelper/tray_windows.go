// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"fmt"
	"os"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/desktop"
)

const trayMessage = 0x8000 + 35 // WM_APP

// Full Windows 10+ NOTIFYICONDATAW layout, including the version/timeout union.
type notifyIconData struct {
	Size                uint32
	Window              uintptr
	ID, Flags, Callback uint32
	Icon                uintptr
	Tip                 [128]uint16
	State, StateMask    uint32
	Info                [256]uint16
	Version             uint32
	Title               [64]uint16
	InfoFlags           uint32
	GUID                syscall.GUID
	BalloonIcon         uintptr
}

type trayRoute struct {
	iconID uint32
	epoch  uint64
}
type fileNotice struct {
	account uint64
	first   string
	count   int
}
type observedFiles struct {
	epoch uint64
	seen  map[string]bool
	order []string
	next  int
}

const maxNotificationIDs = 4096 // comfortably exceeds the 300-row history, including message-ID aliases
func (s *observedFiles) remember(id string) {
	if id == "" || s.seen[id] {
		return
	}
	if len(s.order) == maxNotificationIDs {
		delete(s.seen, s.order[s.next])
		s.order[s.next] = id
		s.next = (s.next + 1) % maxNotificationIDs
	} else {
		s.order = append(s.order, id)
	}
	s.seen[id] = true
}

type trayState struct {
	enabled        bool
	taskbarCreated uint32
	activation     uintptr
	nextIcon       uint32
	icons          map[uint64]trayRoute
	observed       map[uint64]observedFiles
}

// The Shell provides one callback identifier per icon. Two account-specific
// routes let a click on an earlier notification still open its original tab.
func (a *window) iconData(id uint32) notifyIconData {
	n := notifyIconData{Window: a.h, ID: id}
	n.Size = uint32(unsafe.Sizeof(n))
	return n
}
func shellNotify(operation uintptr, n *notifyIconData) bool {
	ok, _, _ := shell32.NewProc("Shell_NotifyIconW").Call(operation, uintptr(unsafe.Pointer(n)))
	return ok != 0
}
func setShellText(dst []uint16, text string) {
	for i := range dst {
		dst[i] = 0
	}
	i := 0
	for _, r := range text {
		if r == 0 {
			continue
		}
		if r > 0xffff {
			if i+2 >= len(dst) {
				break
			}
			hi, lo := utf16.EncodeRune(r)
			dst[i], dst[i+1] = uint16(hi), uint16(lo)
			i += 2
		} else {
			if i+1 >= len(dst) {
				break
			}
			dst[i] = uint16(r)
			i++
		}
	}
}
func (a *window) startTray() bool {
	m, _, _ := user32.NewProc("RegisterWindowMessageW").Call(uintptr(unsafe.Pointer(wide("TaskbarCreated"))))
	a.tray.taskbarCreated = uint32(m)
	a.tray.enabled = true
	a.tray.icons = make(map[uint64]trayRoute)
	a.tray.observed = make(map[uint64]observedFiles)
	a.tray.newFiles(a.snap, false) // baseline idle tabs before Start() can deliver the first live sync
	return a.ensurePrimary(a.snap)
}
func (a *window) ensurePrimary(s desktop.Snapshot) bool {
	if len(a.tray.icons) > 0 {
		return true
	}
	if len(s.Accounts) == 0 {
		return false
	}
	_, ok := a.ensureIcon(s.Accounts[0])
	return ok
}
func (a *window) ensureIcon(ac desktop.Account) (trayRoute, bool) {
	if route, ok := a.tray.icons[ac.ID]; ok {
		return route, true
	}
	for {
		a.tray.nextIcon = a.tray.nextIcon%65535 + 1 // version 4 callbacks carry a 16-bit icon ID
		used := false
		for _, r := range a.tray.icons {
			if r.iconID == a.tray.nextIcon {
				used = true
			}
		}
		if !used {
			break
		}
	}
	route := trayRoute{a.tray.nextIcon, ac.DraftEpoch}
	n := a.iconData(route.iconID)
	n.Flags = 0x87 // MESSAGE | ICON | TIP | SHOWTIP
	n.Callback = trayMessage
	n.Icon = a.icon
	setShellText(n.Tip[:], "WX FileHelper")
	if !shellNotify(0, &n) {
		return trayRoute{}, false
	} // NIM_ADD
	n.Version = 4
	if !shellNotify(4, &n) {
		shellNotify(2, &n)
		return trayRoute{}, false
	} // NIM_SETVERSION
	a.tray.icons[ac.ID] = route
	return route, true
}
func (a *window) removeIcon(account uint64) {
	r := a.tray.icons[account]
	n := a.iconData(r.iconID)
	shellNotify(2, &n)
	delete(a.tray.icons, account)
}
func (a *window) removeTrayIcons() {
	for id := range a.tray.icons {
		a.removeIcon(id)
	}
}
func (a *window) recreateTray() {
	// Explorer just restarted; its prior icon handles and callbacks are gone.
	clear(a.tray.icons)
	if !a.ensurePrimary(a.snap) {
		if visible, _, _ := user32.NewProc("IsWindowVisible").Call(a.h); visible == 0 {
			a.restoreWindow(0)
		}
	}
}
func (a *window) hideToTray() {
	// Consume arrivals while still foreground before they can turn into stale toasts.
	a.refresh(false)
	if a.tray.enabled && a.ensurePrimary(a.snap) {
		showWindow.Call(a.h, 0)
		return
	}
	// Without a Shell notification area, keep a taskbar button as the return path.
	showWindow.Call(a.h, 6) // SW_MINIMIZE
}
func (a *window) restoreWindow(account uint64) {
	if account != 0 {
		s := a.c.Snapshot()
		for _, ac := range s.Accounts {
			if ac.ID == account && ac.DraftEpoch == a.tray.icons[account].epoch {
				_ = a.c.SetActive(account)
				break
			}
		}
	}
	show := uintptr(5)
	if minimized, _, _ := user32.NewProc("IsIconic").Call(a.h); minimized != 0 {
		show = 9
	} // SW_RESTORE
	showWindow.Call(a.h, show)
	user32.NewProc("SetForegroundWindow").Call(a.h)
	a.refresh(false)
}
func (a *window) requestExit() {
	a.refresh(false)
	if a.confirmClose() {
		user32.NewProc("DestroyWindow").Call(a.h)
	}
}
func (a *window) isBackground() bool {
	visible, _, _ := user32.NewProc("IsWindowVisible").Call(a.h)
	minimized, _, _ := user32.NewProc("IsIconic").Call(a.h)
	if visible == 0 || minimized != 0 {
		return true
	}
	foreground, _, _ := user32.NewProc("GetForegroundWindow").Call()
	var pid uint32
	user32.NewProc("GetWindowThreadProcessId").Call(foreground, uintptr(unsafe.Pointer(&pid)))
	return pid != uint32(os.Getpid())
}

// Keep an attachment-ID ledger for each signed-in session, independent of its
// bounded on-screen history. Never replay a tab's initial backlog or foreground
// arrivals when the window later goes to the tray.
func (t *trayState) newFiles(s desktop.Snapshot, background bool) []fileNotice {
	if t.observed == nil {
		t.observed = make(map[uint64]observedFiles)
	}
	live := make(map[uint64]bool, len(s.Accounts))
	var notices []fileNotice
	for _, ac := range s.Accounts {
		live[ac.ID] = true
		state, known := t.observed[ac.ID]
		initial := !known || state.epoch != ac.DraftEpoch
		if initial {
			state = observedFiles{epoch: ac.DraftEpoch, seen: make(map[string]bool)}
		}
		if !ac.SignedIn {
			t.observed[ac.ID] = state
			continue
		}
		notice := fileNotice{account: ac.ID}
		for _, m := range ac.Messages {
			switch m.Kind {
			case "file", "image", "video":
			default:
				continue
			}
			if m.ID == "" {
				continue
			}
			knownID := state.seen[m.ID] || state.seen[m.AlternateID] || state.seen[m.ClientID]
			// A sync echo can beat the HTTP acknowledgement of a local upload.
			// The controller reconciles its origin before clearing Busy.
			if !initial && !knownID && background && ac.Busy && m.Kind == "file" && !m.Outgoing {
				continue
			}
			for _, id := range []string{m.ID, m.AlternateID, m.ClientID} {
				state.remember(id)
			}
			if knownID || initial || !background || m.Outgoing {
				continue
			}
			if notice.count == 0 {
				notice.first = m.Text
			}
			notice.count++
		}
		if notice.count > 0 {
			notices = append(notices, notice)
		}
		t.observed[ac.ID] = state
	}
	for id := range t.observed {
		if !live[id] {
			delete(t.observed, id)
		}
	}
	return notices
}
func (a *window) observeTransfers(s desktop.Snapshot) {
	if !a.tray.enabled {
		return
	}
	for id, route := range a.tray.icons {
		valid := false
		for _, ac := range s.Accounts {
			if id == ac.ID && route.epoch == ac.DraftEpoch {
				valid = true
				break
			}
		}
		if !valid {
			a.removeIcon(id)
		}
	}
	a.ensurePrimary(s)
	for _, notice := range a.tray.newFiles(s, a.isBackground()) {
		for index, ac := range s.Accounts {
			if ac.ID != notice.account {
				continue
			}
			route, ok := a.ensureIcon(ac)
			if !ok {
				break
			}
			n := a.iconData(route.iconID)
			n.Flags = 0x10
			n.InfoFlags = 0x81 // NIF_INFO; NIIF_INFO | RESPECT_QUIET_TIME
			account := ac.Name
			if account == "" {
				account = fmt.Sprintf(a.tr("账号 %d", "Account %d"), index+1)
			}
			setShellText(n.Title[:], a.tr("收到文件", "File received")+" · "+account)
			body := notice.first
			if notice.count > 1 {
				body = fmt.Sprintf(a.tr("收到 %d 个附件，包括 %s", "Received %d attachments, including %s"), notice.count, body)
			}
			if body == "" {
				body = a.tr("点击查看文件", "Open to view the file")
			}
			setShellText(n.Info[:], body)
			shellNotify(1, &n)
			break
		}
	}
}

func (a *window) trayCallback(w, l uintptr) {
	event := uint32(l & 0xffff)
	id := uint32(l>>16) & 0xffff
	for account, r := range a.tray.icons {
		if r.iconID != id {
			continue
		}
		switch event {
		case 0x405:
			a.restoreWindow(account) // NIN_BALLOONUSERCLICK
		case 0x400, 0x401, 0x203:
			a.restoreWindow(0) // NIN_SELECT, NIN_KEYSELECT, double click
		case 0x7b:
			a.trayMenu(account, id, int32(int16(w)), int32(int16(w>>16))) // WM_CONTEXTMENU
		}
		return
	}
}
func (a *window) trayMenu(account uint64, id uint32, x, y int32) {
	menu, _, _ := user32.NewProc("CreatePopupMenu").Call()
	if menu == 0 {
		return
	}
	defer user32.NewProc("DestroyMenu").Call(menu)
	user32.NewProc("AppendMenuW").Call(menu, 0, 1, uintptr(unsafe.Pointer(wide(a.tr("打开文件传输助手", "Open WX FileHelper")))))
	user32.NewProc("AppendMenuW").Call(menu, 0, idExit, uintptr(unsafe.Pointer(wide(a.tr("退出程序", "Quit WX FileHelper")))))
	user32.NewProc("SetMenuDefaultItem").Call(menu, 1, 0)
	if x == -1 && y == -1 {
		var p point
		user32.NewProc("GetCursorPos").Call(uintptr(unsafe.Pointer(&p)))
		x, y = p.X, p.Y
	}
	user32.NewProc("SetForegroundWindow").Call(a.h)
	pick, _, _ := user32.NewProc("TrackPopupMenu").Call(menu, 0x100|0x2, uintptr(x), uintptr(y), 0, a.h, 0)
	user32.NewProc("PostMessageW").Call(a.h, 0, 0, 0)
	switch pick {
	case 1:
		a.restoreWindow(account)
	case idExit:
		a.requestExit()
	default:
		n := a.iconData(id)
		shellNotify(3, &n)
	}
}
