// SPDX-License-Identifier: AGPL-3.0-only
package remote

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTerminalReceiptAndPartialCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	dir := t.TempDir()
	target := filepath.Join(dir, "existing.zip")
	if e := os.WriteFile(target, []byte("original"), 0600); e != nil {
		t.Fatal(e)
	}
	f, e := os.CreateTemp(dir, ".partial-")
	if e != nil {
		t.Fatal(e)
	}
	s := &Session{ctx: ctx, cancel: cancel, file: f, temp: f.Name(), target: target, sum: sha256.New(), state: Status{State: "receiving", Size: 3}}
	if e = s.writeChunk([]byte("abc")); e != nil {
		t.Fatal(e)
	}
	if e = s.onControl(control{Type: "finish", Hash: "wrong"}); e == nil {
		t.Fatal("corrupt transfer accepted")
	}
	s.fail(e)
	if b, e := os.ReadFile(target); e != nil || string(b) != "original" {
		t.Fatal("failed transfer changed destination")
	}
	if _, e = os.Stat(f.Name()); !os.IsNotExist(e) {
		t.Fatal("failed transfer kept temporary file")
	}
	ctx, cancel = context.WithCancel(context.Background())
	received := &Session{ctx: ctx, cancel: cancel, state: Status{State: "complete", Saved: target, Done: 3}}
	received.fail(errors.New("peer closed after ACK"))
	received.Close()
	if st := received.Snapshot(); st.State != "complete" || st.Error != "" || st.Saved != target || st.Done != 3 {
		t.Fatalf("committed receipt rolled back: %+v", st)
	}
}
