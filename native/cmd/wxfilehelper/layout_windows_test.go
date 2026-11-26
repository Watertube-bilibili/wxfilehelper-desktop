// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"runtime"
	"testing"
	"unsafe"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/desktop"
	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
)

// Exercise native HWNDs for signed-in layouts without making a login or network request.
func TestWindowsTransferLayouts(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	err := withWindow(func(a *window) error {
		s := a.c.Snapshot()
		a.tab(s.Active).kind = idWeChat
		s.Accounts[0].SignedIn = true
		s.Accounts[0].Status = "online"
		s.Accounts[0].Messages = []wechat.Message{{ID: "attachment", Kind: "file", MediaID: "test-media", Sender: "phone", Text: "slides.pptx", Size: 4096}}
		s.Accounts[0].Files = []desktop.QueuedFile{{Name: "slides.pdf", Size: 65536}}
		for _, size := range []struct {
			w, h int
			rail bool
		}{{590, 430, false}, {1024, 700, true}, {1920, 1080, true}} {
			var client, outer rect
			user32.NewProc("GetClientRect").Call(a.h, uintptr(unsafe.Pointer(&client)))
			user32.NewProc("GetWindowRect").Call(a.h, uintptr(unsafe.Pointer(&outer)))
			width := a.px(size.w) + int(outer.Right-outer.Left-client.Right)
			height := a.px(size.h) + int(outer.Bottom-outer.Top-client.Bottom)
			user32.NewProc("SetWindowPos").Call(a.h, 0, 0, 0, uintptr(width), uintptr(height), 0x16)
			a.display(s, true)
			if a.rail != size.rail {
				t.Fatalf("incorrect sidebar mode at %dx%d", size.w, size.h)
			}
			for _, id := range []int{idSend, idChoose, idEditor, idHistory, idQueue, idRemoveFile} {
				b, ok := a.placements[id]
				if !ok || b.Left < 0 || b.Top < 0 || b.Right > int32(a.px(a.w)+1) || b.Bottom > int32(a.px(a.hgt)+1) {
					t.Fatalf("control %d outside %dx%d: %+v", id, size.w, size.h, b)
				}
				for other, v := range a.placements {
					if id != other && b.Left < v.Right && b.Right > v.Left && b.Top < v.Bottom && b.Bottom > v.Top {
						t.Fatalf("controls %d and %d overlap at %dx%d", id, other, size.w, size.h)
					}
				}
			}
		}
		h := a.children[idHistory]
		send.Call(h, 0x186, 0, 0)
		if m, ok := a.selectedMessage(); !ok || m.ID != "attachment" {
			t.Fatal("file selection did not resolve to its message")
		}
		a.enableActions()
		if ok, _, _ := user32.NewProc("IsWindowEnabled").Call(a.children[idSave]); ok == 0 {
			t.Fatal("Save as disabled for selected incoming PPTX")
		}
		a.layout()
		if _, ok := a.placements[idCover]; !ok {
			t.Fatal("no on-demand preview action for the incoming PPTX")
		}
		a.tab(s.Active).view = idTransfers
		a.display(s, true)
		if _, ok := a.placements[idHistory]; ok {
			t.Fatal("history overlaps the queue view")
		}
		if _, ok := a.placements[idQueue]; !ok {
			t.Fatal("selected tab lost its queued attachments")
		}
		a.tab(s.Active).view = idFiles
		s.Accounts[0].Messages = append(s.Accounts[0].Messages, wechat.Message{ID: "text", Kind: "text", Text: "draft comparison"})
		a.display(s, true)
		if len(a.timeline) != 1 || a.timeline[0].message.ID != "attachment" {
			t.Fatal("file inbox should filter text")
		}
		a.command(idNavLocal)
		tabID := a.c.Snapshot().Active
		if tabID == s.Active || a.tab(tabID).kind != idLocal || a.tab(s.Active).view != idFiles {
			t.Fatal("navigating to LocalSend lost the original tab or its view")
		}
		a.command(idNavWeChat)
		if a.c.Snapshot().Active != s.Active || len(a.c.Snapshot().Accounts) != 2 {
			t.Fatal("sidebar did not reuse the open WeChat tab")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
