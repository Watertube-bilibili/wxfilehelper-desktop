// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"bytes"
	"image"
	"image/draw"
	"unsafe"
)

// The logo is artwork, so resample in premultiplied alpha at the actual display
// pixel size. QR images use a different renderer which preserves sharp modules.
type serviceIcon struct {
	image *image.RGBA
	sizes map[int][]byte
}

func (i *serviceIcon) load(data []byte) {
	src, _, e := image.Decode(bytes.NewReader(data))
	if e != nil {
		return
	}
	b := src.Bounds()
	i.image = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(i.image, i.image.Bounds(), src, b.Min, draw.Src)
	i.sizes = map[int][]byte{}
}
func (i *serviceIcon) drawOn(dc uintptr, r rect, bg uintptr) {
	n := int(min(r.Right-r.Left, r.Bottom-r.Top))
	if n < 1 || i.image == nil {
		return
	}
	pixels := i.sizes[n]
	if pixels == nil {
		pixels = make([]byte, n*n*4)
		w, h := i.image.Rect.Dx(), i.image.Rect.Dy()
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				var sum [4]int
				total := 0
				// Area-weighted resampling using integer subpixel coordinates.
				for sy := y * h / n; sy < ((y+1)*h+n-1)/n; sy++ {
					wy := min((y+1)*h, (sy+1)*n) - max(y*h, sy*n)
					for sx := x * w / n; sx < ((x+1)*w+n-1)/n; sx++ {
						wx := min((x+1)*w, (sx+1)*n) - max(x*w, sx*n)
						weight := wx * wy
						src := sy*i.image.Stride + 4*sx
						for c := 0; c < 4; c++ {
							sum[c] += int(i.image.Pix[src+c]) * weight
						}
						total += weight
					}
				}
				j := (y*n + x) * 4
				for c := 0; c < 4; c++ {
					pixels[j+c] = byte((sum[c] + total/2) / total)
				}
			}
		}
		if len(i.sizes) > 20 {
			i.sizes = map[int][]byte{}
		}
		i.sizes[n] = pixels
	}
	dst := make([]byte, len(pixels))
	for j := 0; j < len(pixels); j += 4 {
		alpha := uint32(255 - pixels[j+3])
		for c := 0; c < 3; c++ {
			b := uint32((bg >> uint(8*c)) & 255)
			dst[j+2-c] = byte(uint32(pixels[j+c]) + (b*alpha+127)/255)
		}
	}
	header := struct {
		Size          uint32
		Width, Height int32
		Planes, Depth uint16
		Rest          [6]uint32
	}{Size: 40, Width: int32(n), Height: -int32(n), Planes: 1, Depth: 32}
	x, y := int(r.Left)+(int(r.Right-r.Left)-n)/2, int(r.Top)+(int(r.Bottom-r.Top)-n)/2
	gdi32.NewProc("SetDIBitsToDevice").Call(dc, uintptr(x), uintptr(y), uintptr(n), uintptr(n), 0, 0, 0, uintptr(n), uintptr(unsafe.Pointer(&dst[0])), uintptr(unsafe.Pointer(&header)), 0)
}

func (a *window) updateWindowIcons() {
	for _, kind := range []uintptr{0, 1} {
		size := 16
		if kind == 1 {
			size = 32
		}
		n := a.px(size)
		h, _, _ := user32.NewProc("LoadImageW").Call(a.instance, 1, 1, uintptr(n), uintptr(n), 0x8000)
		if h != 0 {
			send.Call(a.h, 0x80, kind, h)
			if kind == 1 {
				a.icon = h
			}
		}
	}
}
