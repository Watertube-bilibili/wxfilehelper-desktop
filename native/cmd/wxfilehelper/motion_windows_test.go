// SPDX-License-Identifier: AGPL-3.0-only
package main

import (
	"testing"
	"time"
)

func TestInterruptibleMotion(t *testing.T) {
	now := time.Unix(0, 0)
	var v transition
	v.target(52, now, 0)
	v.target(196, now, 180*time.Millisecond)
	mid := now.Add(60 * time.Millisecond)
	before := v.value(mid)
	if before <= 52 || before >= 196 {
		t.Fatal("missing intermediate frame")
	}
	v.target(52, mid, 180*time.Millisecond)
	if v.value(mid) != before || v.value(mid.Add(70*time.Millisecond)) >= before || v.value(mid.Add(time.Second)) != 52 {
		t.Fatal("reversing should continue from the displayed position")
	}
	v.target(196, mid, 0)
	if v.value(mid) != 196 {
		t.Fatal("reduced motion should complete immediately")
	}
}
