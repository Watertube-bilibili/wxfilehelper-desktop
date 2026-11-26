// SPDX-License-Identifier: AGPL-3.0-only
// Package nearby hosts user-initiated transfers on a selected LAN interface.
package nearby

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
)

const Chunk = 4 << 20
const MaxSize = 8 << 30

type File struct {
	Name   string
	Size   int64
	SHA256 string
	ID     string
}
type offer struct {
	File
	path string
}
type upload struct {
	File
	path    string
	done    map[int]bool
	writing map[int]bool
	touched time.Time
}
type Server struct {
	mu                sync.Mutex
	URL, Pin, Key     string
	dir, work, status string
	http              *http.Server
	shared            offer
	incoming          *upload
	pending           *request
	session           *request
	closed            bool
	stop              chan struct{}
	receipts          []string
}

func Addresses() []string {
	var out []string
	entries, _ := net.InterfaceAddrs()
	for _, e := range entries {
		p, err := netip.ParsePrefix(e.String())
		if err == nil && p.Addr().Is4() && p.Addr().IsPrivate() {
			out = append(out, p.Addr().String())
		}
	}
	return out
}
func token() string { var b [16]byte; rand.Read(b[:]); return hex.EncodeToString(b[:]) }
func Start(ip string, localSend bool, dir string) (*Server, error) {
	addr, err := netip.ParseAddr(ip)
	if err != nil || !(addr.IsPrivate() || addr.IsLoopback()) {
		return nil, errors.New("请选择局域网地址 / Choose a LAN address")
	}
	port := "0"
	if localSend {
		port = "53317"
	}
	l, err := net.Listen("tcp4", net.JoinHostPort(ip, port))
	if err != nil {
		return nil, err
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		l.Close()
		return nil, err
	}
	cert := x509.Certificate{SerialNumber: new(big.Int).SetBytes([]byte(token()[:16])), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(48 * time.Hour), IPAddresses: []net.IP{net.ParseIP(ip)}, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, &cert, &cert, &key.PublicKey, key)
	if err != nil {
		l.Close()
		return nil, err
	}
	work, err := os.MkdirTemp("", "ultrafile-")
	if err != nil {
		l.Close()
		return nil, err
	}
	fp := sha256.Sum256(der)
	s := &Server{URL: "https://" + l.Addr().String(), Pin: hex.EncodeToString(fp[:]), Key: token(), dir: dir, work: work, stop: make(chan struct{})}
	mux := http.NewServeMux()
	if localSend {
		s.localRoutes(mux)
	} else {
		mux.HandleFunc("/ultra/v1/", s.route)
	}
	s.http = &http.Server{Handler: mux, ReadHeaderTimeout: 8 * time.Second, ReadTimeout: 30 * time.Minute, WriteTimeout: 31 * time.Minute, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16384}
	tlsListener := tls.NewListener(l, &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}, MinVersion: tls.VersionTLS12})
	go s.http.Serve(tlsListener)
	if localSend && addr.IsPrivate() {
		go s.announce(net.ParseIP(ip))
	}
	return s, nil
}
func (s *Server) Close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.stop)
	if s.pending != nil {
		s.pending.decide(false)
	}
	s.mu.Unlock()
	s.http.Close()
	os.RemoveAll(s.work)
}
func (s *Server) Status() string { s.mu.Lock(); defer s.mu.Unlock(); return s.status }
func (s *Server) Receipts() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	files := s.receipts
	s.receipts = nil
	return files
}
func (s *Server) setStatus(v string) { s.mu.Lock(); s.status = v; s.mu.Unlock() }
func (s *Server) Pair() string       { return s.URL + "#" + s.Pin + "." + s.Key }
func (s *Server) Share(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > MaxSize {
		return errors.New("文件应小于 8 GiB / File limit 8 GiB")
	}
	tmp, err := os.CreateTemp(s.work, "share-")
	if err != nil {
		return err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(f, MaxSize+1))
	ce := tmp.Close()
	if err != nil || ce != nil || n != info.Size() {
		os.Remove(tmp.Name())
		return errors.New("读取文件失败或文件已变化 / File changed or could not be read")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	old := s.shared.path
	s.shared = offer{File{filepath.Base(path), n, hex.EncodeToString(h.Sum(nil)), token()}, tmp.Name()}
	s.status = "可接收 / Ready: " + s.shared.Name
	if old != "" {
		os.Remove(old)
	}
	return nil
}
func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte("Bearer "+s.Key)) != 1 {
		w.WriteHeader(403)
		return
	}
	switch r.Method + " " + r.URL.Path {
	case "GET /ultra/v1/file":
		s.mu.Lock()
		o := s.shared
		s.mu.Unlock()
		if o.ID == "" {
			w.WriteHeader(404)
			return
		}
		reply(w, o.File)
	case "GET /ultra/v1/chunk":
		s.mu.Lock()
		o := s.shared
		s.mu.Unlock()
		i, e := strconv.Atoi(r.URL.Query().Get("i"))
		if e != nil || i < 0 || int64(i) > MaxSize/Chunk || o.ID == "" || r.URL.Query().Get("id") != o.ID || int64(i)*Chunk >= o.Size {
			w.WriteHeader(404)
			return
		}
		f, e := os.Open(o.path)
		if e != nil {
			w.WriteHeader(409)
			return
		}
		defer f.Close()
		n := min(int64(Chunk), o.Size-int64(i)*Chunk)
		w.Header().Set("Content-Length", strconv.FormatInt(n, 10))
		io.CopyN(w, io.NewSectionReader(f, int64(i)*Chunk, n), n)
	case "POST /ultra/v1/start":
		var f File
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&f) != nil || !validFile(f) {
			w.WriteHeader(400)
			return
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if u := s.incoming; u != nil {
			if len(u.writing) > 0 || time.Since(u.touched) < 15*time.Minute {
				w.WriteHeader(409)
				return
			}
			os.Remove(u.path)
			s.incoming = nil
		}
		tmp, e := os.CreateTemp(s.work, "receive-")
		if e != nil {
			w.WriteHeader(500)
			return
		}
		tmp.Close()
		f.ID = token()
		s.incoming = &upload{f, tmp.Name(), map[int]bool{}, map[int]bool{}, time.Now()}
		reply(w, f)
	case "PUT /ultra/v1/chunk":
		s.writeChunk(w, r)
	case "POST /ultra/v1/finish", "DELETE /ultra/v1/upload":
		s.mu.Lock()
		u := s.incoming
		if u == nil || r.URL.Query().Get("id") != u.ID || len(u.writing) != 0 {
			s.mu.Unlock()
			w.WriteHeader(409)
			return
		}
		if r.Method == "DELETE" {
			s.incoming = nil
			s.mu.Unlock()
			os.Remove(u.path)
			w.WriteHeader(204)
			return
		}
		if int64(len(u.done)) != (u.Size+Chunk-1)/Chunk {
			s.mu.Unlock()
			w.WriteHeader(409)
			return
		}
		s.incoming = nil
		s.mu.Unlock()
		defer os.Remove(u.path)
		if e := s.commit(u.path, u.File); e != nil {
			http.Error(w, e.Error(), 422)
			return
		}
		w.WriteHeader(204)
	default:
		w.WriteHeader(404)
	}
}
func validFile(f File) bool {
	b, e := hex.DecodeString(f.SHA256)
	return f.Name != "" && len(f.Name) <= 1024 && f.Size >= 0 && f.Size <= MaxSize && e == nil && len(b) == 32
}
func (s *Server) writeChunk(w http.ResponseWriter, r *http.Request) {
	i, e := strconv.Atoi(r.URL.Query().Get("i"))
	s.mu.Lock()
	u := s.incoming
	if e != nil || i < 0 || int64(i) > MaxSize/Chunk || u == nil || r.URL.Query().Get("id") != u.ID || int64(i)*Chunk >= u.Size || u.writing[i] {
		s.mu.Unlock()
		w.WriteHeader(409)
		return
	}
	n := min(int64(Chunk), u.Size-int64(i)*Chunk)
	if r.ContentLength != n {
		s.mu.Unlock()
		w.WriteHeader(400)
		return
	}
	u.touched = time.Now()
	if u.done[i] {
		s.mu.Unlock()
		w.WriteHeader(204)
		return
	}
	u.writing[i] = true
	s.mu.Unlock()
	f, e := os.OpenFile(u.path, os.O_WRONLY, 0600)
	if e == nil {
		_, e = io.CopyN(io.NewOffsetWriter(f, int64(i)*Chunk), r.Body, n)
		ce := f.Close()
		if e == nil {
			e = ce
		}
	}
	s.mu.Lock()
	delete(u.writing, i)
	if e == nil {
		u.done[i] = true
	}
	s.mu.Unlock()
	if e != nil {
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(204)
}
func (s *Server) commit(path string, meta File) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()
	if err = os.MkdirAll(s.dir, 0700); err != nil {
		return err
	}
	// Give each transfer its own directory so an untrusted filename cannot overwrite another file.
	folder, err := os.MkdirTemp(s.dir, "received-")
	if err != nil {
		return err
	}
	dest := filepath.Join(folder, wechat.SafeFilename(meta.Name))
	f, err := os.CreateTemp(folder, ".partial-")
	if err != nil {
		os.Remove(folder)
		return err
	}
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, hash), src)
	ce := f.Close()
	if err != nil || ce != nil || n != meta.Size || (meta.SHA256 != "" && meta.SHA256 != hex.EncodeToString(hash.Sum(nil))) {
		os.Remove(f.Name())
		os.Remove(folder)
		return errors.New("文件校验失败 / Integrity check failed")
	}
	if err := os.Rename(f.Name(), dest); err != nil {
		os.Remove(f.Name())
		os.Remove(folder)
		return err
	}
	s.mu.Lock()
	s.status = "已保存 / Saved: " + dest
	s.receipts = append(s.receipts, filepath.Base(dest))
	s.mu.Unlock()
	return nil
}
