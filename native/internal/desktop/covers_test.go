// SPDX-License-Identifier: AGPL-3.0-only
package desktop

import (
	"archive/zip"
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
)

type deckSession struct {
	mockSession
	deck    []byte
	targets chan string
}

func (s *deckSession) Download(ctx context.Context, m wechat.Message, target string) error {
	select {
	case s.targets <- target:
	case <-ctx.Done():
		return ctx.Err()
	}
	return os.WriteFile(target, s.deck, 0600)
}

func testPresentation(t *testing.T, thumbnail bool) []byte {
	t.Helper()
	var deck bytes.Buffer
	pack := zip.NewWriter(&deck)
	part, err := pack.Create("ppt/presentation.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write([]byte("<presentation/>")); err != nil {
		t.Fatal(err)
	}
	if thumbnail {
		part, err = pack.Create("docProps/thumbnail.png")
		if err != nil {
			t.Fatal(err)
		}
		img := image.NewRGBA(image.Rect(0, 0, 640, 360))
		for y := range img.Bounds().Dy() {
			for x := range img.Bounds().Dx() {
				img.SetRGBA(x, y, color.RGBA{34, 104, 72, 255})
			}
		}
		if err = png.Encode(part, img); err != nil {
			t.Fatal(err)
		}
	}
	if err = pack.Close(); err != nil {
		t.Fatal(err)
	}
	return deck.Bytes()
}

func TestPresentationCoverFromSession(t *testing.T) {
	s := &deckSession{mockSession: mockSession{id: "phone"}, deck: testPresentation(t, true), targets: make(chan string, 3)}
	c := newTestController(t, s)
	id := c.Snapshot().Active
	if err := c.Start(id); err != nil {
		t.Fatal(err)
	}
	awaitView(t, c, id, func(a Account) bool { return a.SignedIn })
	m := wechat.Message{ID: "server", AlternateID: "received", Sender: "phone", MediaID: "token", Kind: "file", Text: "课件.PPTX", Size: int64(len(s.deck))}
	c.mu.Lock()
	appendMessage(c.find(id), m, true)
	c.mu.Unlock()
	if err := c.PreviewCover(id, "received"); err != nil {
		t.Fatal(err)
	}
	result := awaitView(t, c, id, func(a Account) bool { _, ok := a.Cover(m); return ok && a.Previewing == "" })
	b, _ := result.Cover(m)
	thumb, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil || thumb.Bounds().Dx() != 416 || thumb.Bounds().Dy() != 234 {
		t.Fatalf("unexpected thumbnail: %v", err)
	}
	r, g, _, _ := thumb.At(1, 1).RGBA()
	if g <= r*2 {
		t.Fatal("preview lost the original slide color")
	}
	if path := <-s.targets; path == "" {
		t.Fatal("missing temporary target")
	} else if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("temporary presentation still present: %v", err)
	}
	if err = c.PreviewCover(id, m.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-s.targets:
		t.Fatal("cached cover downloaded twice")
	default:
	}
	other, err := c.AddAccount()
	if err != nil {
		t.Fatal(err)
	}
	if err = c.PreviewCover(other, m.ID); err == nil {
		t.Fatal("preview crossed account boundary")
	}
	if err = c.SignOut(id); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Snapshot().Accounts[0].Cover(m); ok {
		t.Fatal("cover survived sign out")
	}

	path := filepath.Join(t.TempDir(), "plain.pptx")
	if err = os.WriteFile(path, testPresentation(t, false), 0600); err != nil {
		t.Fatal(err)
	}
	if got := presentationCover(path); len(got) != 0 {
		t.Fatal("unavailable thumbnail should leave an ordinary card")
	}
}
