// SPDX-License-Identifier: AGPL-3.0-only
package desktop

import (
	"archive/zip"
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
)

// A cover is optional presentation metadata. It never affects whether the
// attachment itself can be sent or saved.
func IsPresentation(name string) bool { return strings.EqualFold(filepath.Ext(name), ".pptx") }

func presentationCover(path string) []byte {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > wechat.MaxFileSize {
		return nil
	}
	archive, err := zip.NewReader(file, info.Size())
	if err != nil {
		return nil
	}
	const maxThumbnail = 1 << 20
	for _, entry := range archive.File {
		switch strings.ToLower(entry.Name) {
		case "docprops/thumbnail.jpeg", "docprops/thumbnail.jpg", "docprops/thumbnail.png":
		default:
			continue
		}
		if entry.UncompressedSize64 == 0 || entry.UncompressedSize64 > maxThumbnail {
			continue
		}
		r, err := entry.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(r, maxThumbnail+1))
		r.Close()
		if err != nil || len(data) > maxThumbnail {
			continue
		}
		config, kind, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || (kind != "jpeg" && kind != "png") || config.Width < 1 || config.Height < 1 ||
			config.Width > 2048 || config.Height > 2048 || config.Width*config.Height > 2<<20 {
			continue
		}
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			continue
		}
		// Normalize before caching so a small on-screen card has a small, bounded footprint.
		w := min(config.Width, 416)
		h := max(1, w*config.Height/config.Width)
		if h > 234 {
			h = 234
			w = max(1, h*config.Width/config.Height)
		}
		thumb := image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				r, g, b, a := img.At(x*config.Width/w, y*config.Height/h).RGBA()
				thumb.SetRGBA(x, y, color.RGBA{uint8((r + 65535 - a) >> 8), uint8((g + 65535 - a) >> 8), uint8((b + 65535 - a) >> 8), 255})
			}
		}
		var out bytes.Buffer
		if jpeg.Encode(&out, thumb, &jpeg.Options{Quality: 85}) == nil {
			return out.Bytes()
		}
	}
	return nil
}

// Cover returns immutable JPEG bytes. A present, empty value means the file
// was inspected and contains no displayable thumbnail.
func (v Account) Cover(m wechat.Message) ([]byte, bool) {
	for _, id := range []string{m.ID, m.AlternateID, m.ClientID} {
		if b, ok := v.Covers[id]; ok && id != "" {
			return b, true
		}
	}
	return nil, false
}
func (a *account) rememberCover(id string, data []byte) {
	if a.Covers == nil {
		a.Covers = make(map[string][]byte)
	}
	if _, ok := a.Covers[id]; !ok {
		if len(a.coverOrder) == 16 {
			delete(a.Covers, a.coverOrder[0])
			a.coverOrder = slices.Delete(a.coverOrder, 0, 1)
		}
		a.coverOrder = append(a.coverOrder, id)
	}
	a.Covers[id] = data
}

// PreviewCover downloads only after an explicit request. The temporary copy
// is private to this operation and shares the lifetime of the account session.
func (c *Controller) PreviewCover(id uint64, messageID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	a := c.find(id)
	if a == nil {
		return errTab
	}
	if !a.SignedIn || a.session == nil {
		return errLogin
	}
	if a.Previewing != "" {
		return errBusy
	}
	for _, m := range a.Messages {
		if !hasServerID(m, messageID) || !IsPresentation(m.Text) || !m.CanDownload() ||
			m.Size <= 0 || m.Size > wechat.MaxFileSize {
			continue
		}
		if _, known := a.Cover(m); known {
			return nil
		}
		a.Previewing = m.ID
		a.Error = ""
		c.revision++
		go c.fetchCover(id, a.session, m)
		return nil
	}
	return errors.New("此课件没有可用的封面预览 / Cover preview unavailable for this presentation")
}
func (c *Controller) fetchCover(id uint64, s *session, m wechat.Message) {
	var data []byte
	dir, err := os.MkdirTemp("", "ultrafile-cover-")
	if err == nil {
		defer os.RemoveAll(dir)
		path := filepath.Join(dir, "presentation.pptx")
		err = s.client.Download(s.ctx, m, path)
		if err == nil && s.ctx.Err() == nil {
			data = presentationCover(path)
		}
		// Release the temporary file before publishing the completed preview.
		os.Remove(path)
	}
	if errors.Is(err, wechat.ErrSessionExpired) {
		c.expire(id, s, err)
		return
	}
	c.change(id, s, func(a *account) {
		a.Previewing = ""
		if err != nil {
			a.Error = err.Error()
			return
		}
		a.rememberCover(m.ID, data)
	})
}
