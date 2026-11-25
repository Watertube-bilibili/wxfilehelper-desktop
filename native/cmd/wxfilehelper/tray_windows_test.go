// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"fmt"
	"os"
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/desktop"
	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
)

func TestNotificationsFollowOriginVisibilityAndSession(t *testing.T) {
	state := trayState{}
	old := wechat.Message{ID: "old", Kind: "file", Text: "before.pdf"}
	s := desktop.Snapshot{Accounts: []desktop.Account{{ID: 1, SignedIn: true, Messages: []wechat.Message{old}}, {ID: 2, SignedIn: true}}}
	if n := state.newFiles(s, true); len(n) != 0 {
		t.Fatalf("notified initial history: %+v", n)
	}
	s.Accounts[0].Messages = append(s.Accounts[0].Messages,
		wechat.Message{ID: "text", Kind: "text", Text: "hello"},
		wechat.Message{ID: "sent", Kind: "file", Outgoing: true},
		wechat.Message{ID: "new", AlternateID: "alias", Kind: "file", Text: "new.pdf"},
		wechat.Message{ID: "photo", Kind: "image", Text: "photo.jpg"})
	s.Accounts[1].Messages = []wechat.Message{{ID: "new", Kind: "file", Text: "other.pdf"}}
	ns := state.newFiles(s, true)
	if len(ns) != 2 || ns[0] != (fileNotice{1, "new.pdf", 2}) || ns[1] != (fileNotice{2, "other.pdf", 1}) {
		t.Fatalf("bad batch: %+v", ns)
	}
	s.Accounts[0].Messages = []wechat.Message{{ID: "alias", Kind: "file"}, {ID: "visible", Kind: "file"}}
	if n := state.newFiles(s, false); len(n) != 0 {
		t.Fatal(n)
	}
	if n := state.newFiles(s, true); len(n) != 0 {
		t.Fatal("foreground/replay", n)
	}
	s.Accounts[1].DraftEpoch++
	s.Accounts[1].Messages = []wechat.Message{{ID: "restored", Kind: "file"}}
	if n := state.newFiles(s, true); len(n) != 0 {
		t.Fatal("old session", n)
	}
	s.Accounts = s.Accounts[:1]
	state.newFiles(s, true)
	if len(state.observed) != 1 {
		t.Fatal("closed account retained")
	}
}

func TestNotificationsWaitForLocalSendReceipt(t *testing.T) {
	state := trayState{}
	s := desktop.Snapshot{Accounts: []desktop.Account{{ID: 1, SignedIn: true}}}
	state.newFiles(s, true)
	ac := &s.Accounts[0]
	ac.Busy = true
	ac.Messages = []wechat.Message{{ID: "server-echo", Kind: "file"}, {ID: "phone", Kind: "file", Text: "phone.pdf"}}
	if n := state.newFiles(s, true); len(n) != 0 {
		t.Fatal(n)
	}
	ac.Messages[0].Outgoing = true
	ac.Busy = false
	if n := state.newFiles(s, true); len(n) != 1 || n[0].count != 1 || n[0].first != "phone.pdf" {
		t.Fatal(n)
	}
}

func TestFirstLiveSyncAndNotificationMemoryBound(t *testing.T) {
	state := trayState{}
	s := desktop.Snapshot{Accounts: []desktop.Account{{ID: 1}}}
	state.newFiles(s, false)
	s.Accounts[0].SignedIn = true
	s.Accounts[0].Messages = []wechat.Message{{ID: "first", Kind: "file", Text: "first.pdf"}}
	if n := state.newFiles(s, true); len(n) != 1 || n[0].first != "first.pdf" {
		t.Fatalf("lost first live sync: %v", n)
	}
	ledger := state.observed[1]
	for i := 0; i < maxNotificationIDs*2; i++ {
		ledger.remember(fmt.Sprint(i))
	}
	if len(ledger.seen) != maxNotificationIDs || len(ledger.order) != maxNotificationIDs || ledger.seen["first"] {
		t.Fatal("unbounded notification ledger")
	}
}

func TestNotificationUnicodeAndLayout(t *testing.T) {
	var small [5]uint16
	setShellText(small[:], "中😀文end")
	if got := syscall.UTF16ToString(small[:]); got != "中😀文" {
		t.Fatal(got)
	}
	setShellText(small[:], "ABC😀")
	if got := syscall.UTF16ToString(small[:]); got != "ABC" {
		t.Fatal(got)
	}
	if runtime.GOARCH == "amd64" && unsafe.Sizeof(notifyIconData{}) != 976 {
		t.Fatal("NOTIFYICONDATAW layout")
	}
}

// Opt in on a Windows desktop with Explorer. CI service sessions have no tray.
func TestExplorerTrayLifecycle(t *testing.T) {
	if os.Getenv("WXFILEHELPER_TEST_TRAY") != "1" {
		t.Skip("requires an interactive Windows notification area")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err := withWindow(func(a *window) error {
		if !a.startTray() {
			t.Fatal("Shell_NotifyIconW could not register the tray icon")
		}
		first := a.snap.Active
		second, err := a.c.AddAccount()
		if err != nil {
			return err
		}
		a.refresh(true)
		if _, ok := a.ensureIcon(a.account); !ok {
			t.Fatal("second account icon could not be registered")
		}
		r := a.tray.icons[second]
		a.switchTo(first)
		showWindow.Call(a.h, 5)
		send.Call(a.h, 0x10, 0, 0)
		visible := func() bool { v, _, _ := user32.NewProc("IsWindowVisible").Call(a.h); return v != 0 }
		if visible() || len(a.c.Snapshot().Accounts) != 2 {
			t.Fatal("close did not keep both accounts in background")
		}
		callback := uintptr(r.iconID)<<16 | 0x405
		send.Call(a.h, trayMessage, 0, callback)
		if !visible() || a.c.Snapshot().Active != second {
			t.Fatal("notification did not restore its account")
		}
		a.c.RemoveAccount(second)
		a.refresh(false)
		a.hideToTray()
		send.Call(a.h, trayMessage, 0, callback)
		if visible() || a.c.Snapshot().Active != first {
			t.Fatal("stale callback activated a different account")
		}
		a.removeTrayIcons() // simulate Explorer discarding its previous registrations
		a.recreateTray()
		if len(a.tray.icons) != 1 {
			t.Fatal("Explorer recovery failed")
		}
		a.requestExit()
		if alive, _, _ := user32.NewProc("IsWindow").Call(a.h); alive != 0 || len(a.tray.icons) != 0 {
			t.Fatal("explicit exit left a window or tray icon")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
