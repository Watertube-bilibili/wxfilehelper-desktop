// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"math"
	"syscall"
	"time"
	"unsafe"
)

type transition struct {
	from, to    float64
	start       time.Time
	duration    time.Duration
	initialized bool
}

func (v *transition) value(now time.Time) float64 {
	if v.duration == 0 || !now.Before(v.start.Add(v.duration)) {
		return v.to
	}
	t := max(0, min(1, float64(now.Sub(v.start))/float64(v.duration)))
	return v.from + (v.to-v.from)*(1-math.Pow(1-t, 3))
}
func (v *transition) target(to float64, now time.Time, d time.Duration) bool {
	if !v.initialized {
		v.from, v.to = to, to
		v.initialized = true
		return false
	}
	if v.to == to {
		return false
	}
	v.from = v.value(now)
	v.to = to
	v.start = now
	v.duration = d
	return d > 0
}
func (v *transition) moving(now time.Time) bool {
	return v.duration > 0 && now.Before(v.start.Add(v.duration))
}

type motionState struct {
	disabled, armed  bool
	rail, tabX, tabW transition
	hover            map[int]*transition
}

func (a *window) readMotion() {
	var on int32 = 1
	user32.NewProc("SystemParametersInfoW").Call(0x1042, 0, uintptr(unsafe.Pointer(&on)), 0)
	if on == 0 {
		a.motion.disabled = true
		a.motion.rail.duration = 0
		a.motion.tabX.duration = 0
		a.motion.tabW.duration = 0
		for _, v := range a.motion.hover {
			v.duration = 0
		}
	} else {
		a.motion.disabled = false
	}
}
func (a *window) animate(v *transition, to float64, d time.Duration) {
	if a.motion.disabled {
		d = 0
	}
	if v.target(to, time.Now(), d) && !a.motion.armed {
		user32.NewProc("SetTimer").Call(a.h, 9, 16, 0)
		a.motion.armed = true
	}
}
func (a *window) motionFrame() {
	now := time.Now()
	active := false
	if a.motion.rail.initialized && a.motion.rail.duration != 0 {
		active = a.motion.rail.moving(now)
		a.layout()
		redraw(a.h)
		if !active {
			a.motion.rail.duration = 0
		}
	}
	if a.motion.tabX.duration != 0 || a.motion.tabW.duration != 0 {
		redraw(a.children[idTabs])
		if a.motion.tabX.moving(now) || a.motion.tabW.moving(now) {
			active = true
		} else {
			a.motion.tabX.duration = 0
			a.motion.tabW.duration = 0
		}
	}
	for id, v := range a.motion.hover {
		if v.duration != 0 {
			redraw(a.children[id])
			if v.moving(now) {
				active = true
			} else {
				v.duration = 0
			}
		}
	}
	if !active {
		a.motion.armed = false
		user32.NewProc("KillTimer").Call(a.h, 9)
	}
}
func (a *window) hoverButton(id int) {
	if id == a.hot {
		return
	}
	old := a.hot
	a.hot = id
	if a.motion.hover == nil {
		a.motion.hover = make(map[int]*transition)
	}
	for _, i := range []int{old, id} {
		if i == 0 {
			continue
		}
		v := a.motion.hover[i]
		if v == nil {
			v = &transition{initialized: true}
			a.motion.hover[i] = v
		}
		dest := float64(0)
		if i == id {
			dest = 1
		}
		a.animate(v, dest, 115*time.Millisecond)
		if a.children[i] != 0 {
			redraw(a.children[i])
		}
	}
}
func blend(from, to uintptr, f float64) uintptr {
	var shade uintptr
	for shift := uint(0); shift < 24; shift += 8 {
		x := float64(from >> shift & 255)
		y := float64(to >> shift & 255)
		shade |= uintptr(math.Round(x+(y-x)*f)) << shift
	}
	return shade
}

var hoverCallback uintptr

func init() { hoverCallback = syscall.NewCallback(hoverProc) }
func hoverProc(h uintptr, msg uint32, w, l, sub, id uintptr) uintptr {
	if ui != nil && ui.ready {
		if msg == 0x200 && ui.hot != int(id) {
			ui.hoverButton(int(id))
			t := struct {
				Size, Flags uint32
				H           uintptr
				Time        uint32
			}{Flags: 2, H: h}
			t.Size = uint32(unsafe.Sizeof(t))
			user32.NewProc("TrackMouseEvent").Call(uintptr(unsafe.Pointer(&t)))
		}
		if msg == 0x2a3 && ui.hot == int(id) {
			ui.hoverButton(0)
		}
	}
	if msg == 0x82 {
		comctl32.NewProc("RemoveWindowSubclass").Call(h, hoverCallback, sub)
	}
	v, _, _ := comctl32.NewProc("DefSubclassProc").Call(h, uintptr(msg), w, l)
	return v
}
func (a *window) tabUnderline(dc uintptr, h uintptr, r rect) {
	idx, _, _ := send.Call(h, 0x130b, 0, 0)
	if int(idx) < 0 {
		return
	}
	var b rect
	if ok, _, _ := send.Call(h, 0x130a, idx, uintptr(unsafe.Pointer(&b))); ok == 0 {
		return
	}
	a.animate(&a.motion.tabX, float64(b.Left+int32(a.px(1))), 170*time.Millisecond)
	a.animate(&a.motion.tabW, float64(b.Right-b.Left-int32(a.px(3))), 170*time.Millisecond)
	now := time.Now()
	x := int32(math.Round(a.motion.tabX.value(now)))
	w := int32(math.Round(a.motion.tabW.value(now)))
	y := r.Bottom - int32(a.px(1))
	fill(dc, rect{x, y - int32(a.px(2)), x + w, y}, accent)
}
