// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"context"
	"fmt"
	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/localsendui"
	"golang.org/x/sys/windows"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

type localResult struct {
	exe string
	err error
}
type localHost struct {
	job      windows.Handle
	loading  bool
	cancel   context.CancelFunc
	ready    chan localResult
	progress atomic.Int64
	cmd      *exec.Cmd
	exited   chan error
	child    uintptr
	deadline time.Time
	status   string
	last     string
	shown    bool
	bounds   rect
}

func (a *window) maybeOfficial() {
	if a.local.cmd == nil && !a.local.loading && a.local.status == "" && localsendui.Ready() {
		a.startOfficial()
	}
}
func ownLocalProcess(pid int) windows.Handle {
	job, e := windows.CreateJobObject(nil, nil)
	if e != nil {
		return 0
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	_, e = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info)))
	if e == nil {
		h, e := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(pid))
		if e == nil {
			defer windows.CloseHandle(h)
			if windows.AssignProcessToJobObject(job, h) == nil {
				return job
			}
		}
	}
	windows.CloseHandle(job)
	return 0
}
func (a *window) startOfficial() { a.selectOfficial("") }
func (a *window) selectOfficial(existing string) {
	h := &a.local
	if h.loading || h.cmd != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	h.cancel = cancel
	h.ready = make(chan localResult, 1)
	h.loading = true
	h.status = ""
	h.progress.Store(0)
	done := h.ready
	go func() {
		var path string
		var e error
		if existing != "" {
			path, e = localsendui.Select(ctx, existing)
		} else {
			path, e = localsendui.Prepare(ctx, func(total int64) { h.progress.Store(total) })
		}
		done <- localResult{path, e}
	}()
}
func (a *window) officialStatus() string {
	h := &a.local
	switch {
	case h.status != "":
		return h.status
	case h.child != 0:
		return "LocalSend " + localsendui.Version
	case h.cmd != nil:
		return a.tr("正在打开 LocalSend…", "Opening LocalSend…")
	case h.loading:
		return fmt.Sprintf(a.tr("正在准备官方组件 · %d%%", "Preparing official component · %d%%"), h.progress.Load()*100/localsendui.ArchiveBytes)
	}
	return a.tr("官方 Windows 客户端", "Official Windows client")
}

type runnerSearch struct {
	pid  uint32
	hwnd uintptr
}

var findRunnerCallback = syscall.NewCallback(func(w, param uintptr) uintptr {
	wanted := (*runnerSearch)(unsafe.Pointer(param))
	var pid uint32
	user32.NewProc("GetWindowThreadProcessId").Call(w, uintptr(unsafe.Pointer(&pid)))
	if pid != wanted.pid {
		return 1
	}
	buf := make([]uint16, 96)
	n, _, _ := user32.NewProc("GetClassNameW").Call(w, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n > 0 && syscall.UTF16ToString(buf) == "FLUTTER_RUNNER_WIN32_WINDOW" && getText(w) == "LocalSend" {
		wanted.hwnd = w
		return 0
	}
	return 1
})

func (a *window) pollOfficial() {
	h := &a.local
	if h.loading {
		select {
		case r := <-h.ready:
			h.loading = false
			if h.cancel != nil {
				h.cancel()
			}
			h.cancel = nil
			if r.err != nil {
				h.status = r.err.Error()
				break
			}
			cmd := exec.Command(r.exe)
			cmd.Dir = filepath.Dir(r.exe)
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if e := cmd.Start(); e != nil {
				h.status = e.Error()
				break
			}
			h.job = ownLocalProcess(cmd.Process.Pid)
			h.cmd = cmd
			h.deadline = time.Now().Add(25 * time.Second)
			h.exited = make(chan error, 1)
			done := h.exited
			go func() { done <- cmd.Wait() }()
		default:
		}
	}
	if h.cmd != nil {
		select {
		case e := <-h.exited:
			if h.job != 0 {
				windows.CloseHandle(h.job)
				h.job = 0
			}
			h.cmd = nil
			h.child = 0
			h.shown = false
			h.bounds = rect{}
			if e != nil {
				h.status = e.Error()
			} else {
				h.status = a.tr("LocalSend 已关闭。可重新打开。", "LocalSend closed. You can reopen it.")
			}
		default:
			if h.child == 0 {
				search := runnerSearch{pid: uint32(h.cmd.Process.Pid)}
				user32.NewProc("EnumWindows").Call(findRunnerCallback, uintptr(unsafe.Pointer(&search)))
				if search.hwnd != 0 {
					ch := search.hwnd
					long := user32.NewProc("GetWindowLongPtrW")
					set := user32.NewProc("SetWindowLongPtrW")
					style, _, _ := long.Call(ch, ^uintptr(15))
					ex, _, _ := long.Call(ch, ^uintptr(19))
					showWindow.Call(ch, 0)
					set.Call(ch, ^uintptr(15), (style&^0x80cf0000)|0x46000000)
					set.Call(ch, ^uintptr(19), ex&^0x40000)
					user32.NewProc("SetParent").Call(ch, a.h)
					parent, _, _ := user32.NewProc("GetParent").Call(ch)
					if parent == a.h {
						h.child = ch
						h.status = ""
						a.placeOfficial()
					} else {
						h.status = a.tr("无法载入原版窗口", "Could not attach the official window")
						a.stopOfficial()
					}
				} else if time.Now().After(h.deadline) {
					h.status = a.tr("原版窗口未就绪，请检查是否已有 LocalSend 在运行。", "LocalSend did not open. Check for another running instance.")
					a.stopOfficial()
				}
			}
		}
	}
	status := a.officialStatus()
	if h.last != status {
		h.last = status
		if a.tab(a.snap.Active).kind == idLocal {
			a.layout()
			redraw(a.h)
		}
	}
}
func (a *window) placeOfficial() {
	h := &a.local
	if h.child == 0 {
		return
	}
	t := a.tab(a.snap.Active)
	visible := t.kind == idLocal && !t.picker
	if visible {
		edge := a.sidebarWidth()
		b := a.box(edge, a.headerHeight, a.w-edge, a.hgt-a.headerHeight)
		if h.bounds != b || !h.shown {
			user32.NewProc("SetWindowPos").Call(h.child, 0, uintptr(b.Left), uintptr(b.Top), uintptr(b.Right-b.Left), uintptr(b.Bottom-b.Top), 0x34)
			h.bounds = b
		}
	}
	if visible != h.shown {
		if visible {
			showWindow.Call(h.child, 8)
		} else {
			showWindow.Call(h.child, 0)
		}
		h.shown = visible
	}
}
func (a *window) stopOfficial() {
	h := &a.local
	if h.cancel != nil {
		h.cancel()
		h.cancel = nil
	}
	if h.cmd == nil {
		return
	}
	// Close just the process we launched, even if its own tray preference intercepts WM_CLOSE.
	if h.child != 0 {
		user32.NewProc("PostMessageW").Call(h.child, 0x10, 0, 0)
	}
	cmd, done, job := h.cmd, h.exited, h.job
	h.job = 0
	go func() {
		if job != 0 {
			defer windows.CloseHandle(job)
		}
		select {
		case <-done:
		case <-time.After(4 * time.Second):
			cmd.Process.Kill()
		}
	}()
	h.cmd = nil
	h.child = 0
	h.shown = false
	h.bounds = rect{}
}
func (a *window) officialLayout(put positionControl, left, right, top, btn int) {
	if a.local.child != 0 {
		return
	}
	w := right - left
	a.caption("LocalSend", left, top+8, w, 42, a.titleFont, ink, singleLine)
	description := a.tr("原版的接收、发送和设置将在此标签中打开。", "Use the original Receive, Send and Settings screens in this tab.")
	if a.local.status != "" && a.local.cmd == nil {
		description = a.tr("如果组件无法打开，请解压官方 ZIP，再选择其中的 localsend_app.exe。", "If the component cannot open, extract the official ZIP and choose its localsend_app.exe.")
	}
	a.caption(description, left, top+55, w, 50, a.font, muted, 0x10)
	if a.local.loading || a.local.cmd != nil {
		return
	}
	title := a.tr("打开 LocalSend", "Open LocalSend")
	if !localsendui.Ready() {
		title = a.tr("获取官方组件 · 24.9 MiB", "Get official component · 24.9 MiB")
	}
	put(idOfficial, left, top+112, min(260, w), btn+4)
	setText(a.children[idOfficial], title)
	next := top + btn + 123
	put(idPickOfficial, left, next, min(w/2-3, 205), btn)
	setText(a.children[idPickOfficial], a.tr("选择已解压的原版", "Choose extracted app"))
	if !localsendui.Ready() {
		put(idGetZIP, left+min(w/2-3, 205)+6, next, min(w/2-4, 167), btn)
		setText(a.children[idGetZIP], a.tr("下载官方 ZIP", "Download official ZIP"))
	}
}
