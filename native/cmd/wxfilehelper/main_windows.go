// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"strings"
	"syscall"
	"unsafe"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/desktop"
)

const (
	idTabs = 100 + iota
	idAdd
	idMenu
	idChat
	idFiles
	idDownloads
	idUpdate
	idAbout
	idRefresh
	idHistory
	idSave
	idCopy
	idQueue
	idRemoveFile
	idEditor
	idChoose
	idSend
	idStop
	idNotice
	idLanguage
	idSignout
	idCloseTab
	idToggleFilter
	idExit
)

type window struct {
	h, instance, icon                                   uintptr
	children                                            map[int]uintptr
	font, smallFont, titleFont, brandFont               uintptr
	c                                                   *desktop.Controller
	snap                                                desktop.Snapshot
	account                                             desktop.Account
	ready, english, rail, filesOnly, manualUpdate       bool
	drafts                                              map[uint64]string
	pending                                             map[uint64]pendingDraft
	epochs                                              map[uint64]uint64
	historyRows, queueRows, tabTexts                    []string
	messageIDs                                          []string
	dpi, w, hgt                                         int
	qr                                                  qrBitmap
	qrBounds, qrCard, composer, queueSurface, statusDot rect
	statusInk                                           uintptr
	headerHeight, hot, historyRow                       int
	labels                                              []caption
	placements                                          map[int]rect
	contentLeft, contentRight, contentTop               int
	tray                                                trayState
	lastDir                                             string
}

var ui *window

func main() {
	runtime.LockOSThread()
	smoke := slices.Contains(os.Args[1:], "--smoke-test")
	err := run(smoke)
	if smoke {
		if report := os.Getenv("WXFILEHELPER_SMOKE_REPORT"); report != "" {
			result := "OK\n"
			if err != nil {
				result = err.Error() + "\n"
			}
			if writeErr := os.WriteFile(report, []byte(result), 0600); writeErr != nil {
				err = writeErr
			}
		}
	}
	if err != nil {
		if smoke {
			fmt.Fprintln(os.Stderr, err)
		} else {
			dialog(0, err.Error(), "WX FileHelper", 0x10)
		}
		os.Exit(1)
	}
}

func run(smoke bool) error {
	if smoke {
		return withWindow((*window).smokeTest)
	}
	background := slices.Contains(os.Args[1:], "--background")
	event, _, status := kernel32.NewProc("CreateEventW").Call(0, 0, 0, uintptr(unsafe.Pointer(wide(`Local\WXFileHelper.Native.Activate`))))
	if event != 0 {
		defer kernel32.NewProc("CloseHandle").Call(event)
		if status == syscall.Errno(183) { // Another instance owns the activation event.
			if !background {
				kernel32.NewProc("SetEvent").Call(event)
			}
			return nil
		}
	}
	return withWindow(func(a *window) error {
		a.tray.activation = event
		return a.eventLoop(background)
	})
}

func withWindow(use func(*window) error) error {
	setDPIAwareness()
	controls := struct{ Size, Classes uint32 }{8, 0x000000ff}
	if ok, _, err := comctl32.NewProc("InitCommonControlsEx").Call(uintptr(unsafe.Pointer(&controls))); ok == 0 {
		return fmt.Errorf("Common controls: %w", err)
	}
	instance, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	icon, _, _ := user32.NewProc("LoadIconW").Call(instance, 1)
	if icon == 0 {
		if path, err := os.Executable(); err == nil {
			shell32.NewProc("ExtractIconExW").Call(uintptr(unsafe.Pointer(wide(path))), 0, uintptr(unsafe.Pointer(&icon)), 0, 1)
		}
	}
	if icon == 0 {
		icon, _, _ = user32.NewProc("LoadIconW").Call(0, 32512)
	}
	cursor, _, _ := user32.NewProc("LoadCursorW").Call(0, 32512)
	className := wide("WXFileHelper.Native")
	class := windowClass{Style: 3, Proc: syscall.NewCallback(windowProc), Instance: instance, Icon: icon, Cursor: cursor, Name: className, IconSmall: icon}
	class.Size = uint32(unsafe.Sizeof(class))
	if ok, _, err := user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&class))); ok == 0 {
		return fmt.Errorf("Register window: %w", err)
	}
	c := desktop.New()
	defer user32.NewProc("UnregisterClassW").Call(uintptr(unsafe.Pointer(className)), instance)
	defer c.Close()
	lang, _, _ := kernel32.NewProc("GetUserDefaultUILanguage").Call()
	ui = &window{instance: instance, icon: icon, c: c, children: map[int]uintptr{}, drafts: map[uint64]string{}, pending: map[uint64]pendingDraft{}, epochs: map[uint64]uint64{}, english: lang&0x3ff != 4, dpi: 96}
	var work rect
	user32.NewProc("SystemParametersInfoW").Call(48, 0, uintptr(unsafe.Pointer(&work)), 0)
	systemDPI := getDPI(0)
	w := min(1024*systemDPI/96, max(450, int(work.Right-work.Left)-20))
	h := min(700*systemDPI/96, max(360, int(work.Bottom-work.Top)-20))
	hwnd, _, err := createWindow.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(wide(ui.tr("文件传输助手", "File Transfer")+" · WX FileHelper"))),
		0x02cf0000, uintptr(int(work.Left)+(int(work.Right-work.Left)-w)/2), uintptr(int(work.Top)+(int(work.Bottom-work.Top)-h)/2), uintptr(w), uintptr(h), 0, 0, instance, 0)
	if hwnd == 0 {
		return fmt.Errorf("Create window: %w", err)
	}
	ui.h = hwnd
	defer func() {
		if valid, _, _ := user32.NewProc("IsWindow").Call(hwnd); valid != 0 {
			user32.NewProc("DestroyWindow").Call(hwnd)
		}
	}()
	ui.dpi = getDPI(hwnd)
	if err := ui.createChildren(); err != nil {
		return err
	}
	ui.setFonts()
	ui.ready = true
	shell32.NewProc("DragAcceptFiles").Call(hwnd, 1)
	ui.refresh(true)
	return use(ui)
}

func (a *window) eventLoop(background bool) error {
	hwnd, c := a.h, a.c
	hasTray := a.startTray()
	if !background || !hasTray {
		showWindow.Call(hwnd, 5)
		user32.NewProc("UpdateWindow").Call(hwnd)
	}
	user32.NewProc("SetTimer").Call(hwnd, 1, 200, 0)
	c.StartUpdateChecks()
	a.fail(c.Start(a.snap.Active))
	var m message
	for {
		ok, _, err := user32.NewProc("GetMessageW").Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ok) == -1 {
			return err
		}
		if ok == 0 {
			break
		}
		if m.ID == 0x100 && a.key(m) {
			continue
		}
		if m.ID == 0x100 && m.W == 9 {
			if consumed, _, _ := user32.NewProc("IsDialogMessageW").Call(hwnd, uintptr(unsafe.Pointer(&m))); consumed != 0 {
				continue
			}
		}
		user32.NewProc("TranslateMessage").Call(uintptr(unsafe.Pointer(&m)))
		user32.NewProc("DispatchMessageW").Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}

func (a *window) smokeTest() error {
	first := a.snap.Active
	second, err := a.c.AddAccount()
	if err != nil {
		return err
	}
	a.refresh(true)
	if err := a.c.SetActive(first); err != nil {
		return err
	}
	a.refresh(true)
	tabs := a.children[idTabs]
	count, _, _ := send.Call(tabs, 0x1304, 0, 0)
	selected, _, _ := send.Call(tabs, 0x130b, 0, 0)
	if count != 2 || selected != 0 || first == second || a.account.ID != first {
		return fmt.Errorf("native tab initialization failed")
	}
	send.Call(tabs, 0x130c, 1, 0)
	n := notification{Window: tabs, ID: idTabs, Code: -551}
	send.Call(a.h, 0x4e, idTabs, uintptr(unsafe.Pointer(&n)))
	if a.c.Snapshot().Active != second {
		return fmt.Errorf("native tab selection did not switch accounts")
	}
	if yes, _, _ := user32.NewProc("IsWindowEnabled").Call(a.children[idSend]); yes != 0 {
		return fmt.Errorf("signed-out send enabled")
	}
	return nil
}

func windowProc(h uintptr, msg uint32, w, l uintptr) uintptr {
	if a := ui; a != nil && a.ready {
		if a.tray.taskbarCreated != 0 && msg == a.tray.taskbarCreated {
			a.recreateTray()
			return 0
		}
		switch msg {
		case trayMessage:
			a.trayCallback(w, l)
			return 0
		case 0x5:
			if w != 1 {
				a.layout()
				redraw(h)
			}
			return 0 // WM_SIZE
		case 0x14:
			return 1 // painted in WM_PAINT
		case 0xf:
			a.paint()
			return 0
		case 0x318: // WM_PRINTCLIENT used by Windows previews and printing
			var bounds rect
			user32.NewProc("GetClientRect").Call(h, uintptr(unsafe.Pointer(&bounds)))
			a.drawSurface(w, bounds)
			return 0
		case 0x24:
			m := (*minMaxInfo)(unsafe.Pointer(l))
			m.MinTrack = point{int32(a.px(540)), int32(a.px(400))}
			return 0
		case 0x2e0:
			a.dpi = int(w & 0xffff)
			a.historyRow = 0
			r := (*rect)(unsafe.Pointer(l))
			user32.NewProc("SetWindowPos").Call(h, 0, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 0x14)
			a.setFonts()
			a.layout()
			redraw(h)
			return 0
		case 0x113:
			if a.tray.activation != 0 {
				if signal, _, _ := kernel32.NewProc("WaitForSingleObject").Call(a.tray.activation, 0); signal == 0 {
					a.restoreWindow(0)
				}
			}
			if visible, _, _ := user32.NewProc("IsWindowVisible").Call(h); visible != 0 {
				a.refreshHover()
			}
			a.refresh(false)
			return 0
		case 0x133, 0x134: // WM_CTLCOLOREDIT / WM_CTLCOLORLISTBOX
			bg := white
			if l == a.children[idHistory] || l == a.children[idQueue] {
				bg = base
			}
			setTextColor.Call(w, ink)
			gdi32.NewProc("SetBkColor").Call(w, bg)
			gdi32.NewProc("SetDCBrushColor").Call(w, bg)
			brush, _, _ := gdi32.NewProc("GetStockObject").Call(18) // DC_BRUSH takes the DC's brush color.
			return brush
		case 0x111:
			id, code := int(w&0xffff), int((w>>16)&0xffff)
			if id == idEditor && code == 0x300 {
				a.drafts[a.snap.Active] = getText(a.children[idEditor])
				a.enableActions()
				return 0
			}
			if id == idEditor && (code == 0x100 || code == 0x200) {
				invalidateRect.Call(h, uintptr(unsafe.Pointer(&a.composer)), 0)
				return 0
			}
			if (id == idHistory || id == idQueue) && code == 1 {
				a.enableActions()
				return 0
			}
			if id == idHistory && code == 2 {
				a.openMessage()
				return 0
			}
			if code == 0 {
				a.command(id)
				return 0
			}
		case 0x4e:
			n := (*notification)(unsafe.Pointer(l))
			if n.ID == idTabs && n.Code == -551 {
				idx, _, _ := send.Call(a.children[idTabs], 0x130b, 0, 0)
				if int(idx) >= 0 && int(idx) < len(a.snap.Accounts) {
					a.switchTo(a.snap.Accounts[idx].ID)
				}
				return 0
			}
		case 0x2b:
			item := (*drawItem)(unsafe.Pointer(l))
			switch item.ID {
			case idHistory, idQueue:
				a.drawListRow(item)
			case idTabs:
				a.drawTab(item)
			default:
				a.drawButton(item)
			}
			return 1
		case 0x233:
			paths, err := droppedFiles(w)
			shell32.NewProc("DragFinish").Call(w)
			if err == nil && len(paths) > 0 {
				err = a.c.Queue(a.snap.Active, paths)
			}
			a.fail(err)
			a.refresh(false)
			return 0
		case 0x10:
			a.hideToTray()
			return 0
		case 0x16: // WM_ENDSESSION (sign-out, shutdown or Restart Manager)
			if w != 0 {
				user32.NewProc("DestroyWindow").Call(h)
			}
			return 0
		case 0x2:
			a.ready = false
			a.removeTrayIcons()
			user32.NewProc("KillTimer").Call(h, 1)
			a.freeFonts()
			user32.NewProc("PostQuitMessage").Call(0)
			return 0
		}
	}
	r, _, _ := defWindowProc.Call(h, uintptr(msg), w, l)
	return r
}
func (a *window) tr(zh, en string) string {
	if a.english {
		return en
	}
	return zh
}
func (a *window) px(v int) int { return (v*a.dpi + 48) / 96 }
func (a *window) fail(err error) {
	if err != nil {
		dialog(a.h, err.Error(), a.tr("文件传输助手", "File Transfer"), 0x30)
	}
}
func (a *window) confirmClose() bool {
	for _, s := range a.snap.Accounts {
		if s.Busy || s.Downloading || len(s.Files) > 0 || strings.TrimSpace(a.drafts[s.ID]) != "" {
			if visible, _, _ := user32.NewProc("IsWindowVisible").Call(a.h); visible == 0 {
				a.restoreWindow(0)
			}
			return dialog(a.h, a.tr("还有传输或未发送的内容。现在退出？", "You have a transfer or unsent items. Exit now?"), "WX FileHelper", 0x124) == 6
		}
	}
	return true
}
func (a *window) switchTo(id uint64) { a.fail(a.c.SetActive(id)); a.refresh(false) }

func (a *window) refreshHover() {
	var cursor point
	next := 0
	if ok, _, _ := user32.NewProc("GetCursorPos").Call(uintptr(unsafe.Pointer(&cursor))); ok != 0 {
		coords := uintptr(uint32(cursor.X)) | uintptr(uint32(cursor.Y))<<32
		control, _, _ := user32.NewProc("WindowFromPoint").Call(coords)
		for id, child := range a.children {
			if control == child {
				next = id
				break
			}
		}
	}
	if next == a.hot {
		return
	}
	old := a.hot
	a.hot = next
	for _, id := range []int{old, next} {
		if control := a.children[id]; control != 0 {
			redraw(control)
		}
	}
}
