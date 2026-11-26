package nearby

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOutOfOrderRoundTrip(t *testing.T) {
	s, e := Start("127.0.0.1", false, t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	data := make([]byte, 2*Chunk+127)
	for i := range data {
		data[i] = byte((i/Chunk + i) % 251)
	}
	file := filepath.Join(t.TempDir(), "课件.pptx")
	if e = os.WriteFile(file, data, 0600); e != nil {
		t.Fatal(e)
	}
	if e = s.Share(file); e != nil {
		t.Fatal(e)
	}
	call := func(method, route, token string, body []byte, expect int) []byte {
		t.Helper()
		r := httptest.NewRequest(method, "/ultra/v1/"+route, bytes.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		s.route(w, r)
		if w.Code != expect {
			t.Fatalf("%s %s: got %d, want %d", method, route, w.Code, expect)
		}
		return w.Body.Bytes()
	}
	call("GET", "file", "wrong-key", nil, 403)
	var f File
	if e = json.Unmarshal(call("GET", "file", s.Key, nil, 200), &f); e != nil {
		t.Fatal(e)
	}
	hash := sha256.Sum256(data)
	if f.SHA256 != hex.EncodeToString(hash[:]) || f.Name != "课件.pptx" {
		t.Fatal(f)
	}
	for i := 0; i < 3; i++ {
		b := call("GET", fmt.Sprintf("chunk?id=%s&i=%d", f.ID, i), s.Key, nil, 200)
		if !bytes.Equal(b, data[i*Chunk:min(len(data), (i+1)*Chunk)]) {
			t.Fatal("download mismatch")
		}
	}
	upload := func(meta File) string {
		b, _ := json.Marshal(meta)
		var answer File
		if e = json.Unmarshal(call("POST", "start", s.Key, b, 200), &answer); e != nil {
			t.Fatal(e)
		}
		return answer.ID
	}
	id := upload(f)
	block := func(i int) {
		call("PUT", fmt.Sprintf("chunk?id=%s&i=%d", id, i), s.Key, data[i*Chunk:min(len(data), (i+1)*Chunk)], 204)
	}
	block(2)
	block(0)
	block(0)
	call("POST", "finish?id="+id, s.Key, nil, 409)
	block(1)
	call("POST", "finish?id="+id, s.Key, nil, 204)
	paths, _ := filepath.Glob(filepath.Join(s.dir, "*", "*"))
	if len(paths) != 1 {
		t.Fatal(paths)
	}
	read, e := os.ReadFile(paths[0])
	if e != nil || !bytes.Equal(data, read) {
		t.Fatal("upload mismatch", e)
	}
	f.SHA256 = strings.Repeat("0", 64)
	f.Size = 0
	id = upload(f)
	call("POST", "finish?id="+id, s.Key, nil, 422)
	paths, _ = filepath.Glob(filepath.Join(s.dir, "*", "*"))
	if len(paths) != 1 {
		t.Fatal("failed transfer left a file", paths)
	}
}

func TestLocalSendExchange(t *testing.T) {
	receiver, err := Start("127.0.0.1", true, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer receiver.Close()
	sender, err := Start("127.0.0.1", false, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer sender.Close()
	path := filepath.Join(t.TempDir(), "slides.pptx")
	if err = os.WriteFile(path, []byte("LocalSend v2 fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, accept := range []bool{false, true} {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		result := make(chan error, 1)
		go func() {
			result <- sender.SendLocal(ctx, "127.0.0.1", path, func(pin string) bool { return pin == receiver.Pin })
		}()
		for receiver.Pending() == "" && ctx.Err() == nil {
			time.Sleep(time.Millisecond * 5)
		}
		receiver.Decide(accept)
		err = <-result
		cancel()
		if (err == nil) != accept {
			t.Fatalf("accept %t, got %v", accept, err)
		}
	}
	paths, _ := filepath.Glob(filepath.Join(receiver.dir, "*", "slides.pptx"))
	if len(paths) != 1 {
		t.Fatal(paths)
	}
	b, e := os.ReadFile(paths[0])
	if e != nil || string(b) != "LocalSend v2 fixture" {
		t.Fatal("wrong file")
	}
	m := http.NewServeMux()
	receiver.localRoutes(m)
	request := httptest.NewRequest("POST", ls+"/upload?fileId=x&token=bad", nil)
	response := httptest.NewRecorder()
	m.ServeHTTP(response, request)
	raw, _ := io.ReadAll(response.Result().Body)
	if response.Code != 403 {
		t.Fatal(response.Code, string(raw))
	}
}
