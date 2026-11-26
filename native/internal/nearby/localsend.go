// SPDX-License-Identifier: AGPL-3.0-only
package nearby

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const ls = "/api/localsend/v2"

func (s *Server) announce(ip net.IP) {
	conn, err := net.DialUDP("udp4", &net.UDPAddr{IP: ip}, &net.UDPAddr{IP: net.IPv4(224, 0, 0, 167), Port: 53317})
	if err != nil {
		return
	}
	defer conn.Close()
	advertisement := struct {
		device
		Announce bool `json:"announce"`
	}{s.info(), true}
	data, err := json.Marshal(advertisement)
	if err != nil {
		return
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		conn.Write(data)
		select {
		case <-s.stop:
			return
		case <-ticker.C:
		}
	}
}

type device struct {
	Alias       string `json:"alias"`
	Version     string `json:"version"`
	Fingerprint string `json:"fingerprint"`
	DeviceType  string `json:"deviceType"`
	Protocol    string `json:"protocol"`
	Port        int    `json:"port"`
}
type lsFile struct {
	ID   string `json:"id"`
	Name string `json:"fileName"`
	Size int64  `json:"size"`
	Type string `json:"fileType"`
	Hash string `json:"sha256,omitempty"`
}
type batch struct {
	Info  device            `json:"info"`
	Files map[string]lsFile `json:"files"`
}
type approval struct {
	ID    string            `json:"sessionId"`
	Files map[string]string `json:"files"`
}
type request struct {
	batch
	host     string
	auth     approval
	decision chan bool
	once     sync.Once
	expires  time.Time
	busy     map[string]bool
}

func (r *request) decide(v bool) { r.once.Do(func() { r.decision <- v }) }
func (s *Server) info() device   { return device{"Ultra File", "2.1", s.Pin, "desktop", "https", 53317} }
func (s *Server) Pending() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending == nil {
		return ""
	}
	var names []string
	for _, f := range s.pending.Files {
		names = append(names, f.Name)
	}
	return s.pending.host + " → " + strings.Join(names, ", ")
}
func (s *Server) Decide(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pending != nil {
		s.pending.decide(v)
	}
}
func (s *Server) localRoutes(m *http.ServeMux) {
	m.HandleFunc("GET "+ls+"/info", func(w http.ResponseWriter, r *http.Request) { reply(w, s.info()) })
	m.HandleFunc("POST "+ls+"/register", func(w http.ResponseWriter, r *http.Request) { reply(w, s.info()) })
	m.HandleFunc("POST "+ls+"/prepare-upload", func(w http.ResponseWriter, r *http.Request) {
		var b batch
		if json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&b) != nil || len(b.Files) == 0 || len(b.Files) > 64 {
			w.WriteHeader(400)
			return
		}
		var total int64
		for id, f := range b.Files {
			if id == "" || id != f.ID || f.Name == "" || f.Size < 0 || f.Size > MaxSize || len(f.Name) > 1024 {
				w.WriteHeader(400)
				return
			}
			total += f.Size
		}
		if total > MaxSize {
			w.WriteHeader(413)
			return
		}
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		q := &request{batch: b, host: host, decision: make(chan bool, 1), busy: map[string]bool{}, auth: approval{token(), map[string]string{}}}
		s.mu.Lock()
		if s.closed || s.pending != nil || (s.session != nil && (time.Now().Before(s.session.expires) || len(s.session.busy) > 0)) {
			s.mu.Unlock()
			w.WriteHeader(429)
			return
		}
		s.pending = q
		s.mu.Unlock()
		accept := false
		select {
		case accept = <-q.decision:
		case <-time.After(90 * time.Second):
		case <-r.Context().Done():
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		s.pending = nil
		if !accept || s.closed {
			w.WriteHeader(403)
			return
		}
		for id := range b.Files {
			q.auth.Files[id] = token()
		}
		q.expires = time.Now().Add(30 * time.Minute)
		s.session = q
		reply(w, q.auth)
	})
	m.HandleFunc("POST "+ls+"/upload", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query()
		id := p.Get("fileId")
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		s.mu.Lock()
		q := s.session
		if q == nil || q.host != host || time.Now().After(q.expires) || q.auth.ID != p.Get("sessionId") || q.auth.Files[id] == "" || q.auth.Files[id] != p.Get("token") || q.busy[id] {
			s.mu.Unlock()
			w.WriteHeader(403)
			return
		}
		f := q.Files[id]
		q.busy[id] = true
		s.mu.Unlock()
		good := false
		defer func() {
			s.mu.Lock()
			delete(q.busy, id)
			if good {
				delete(q.auth.Files, id)
				if len(q.auth.Files) == 0 && s.session == q {
					s.session = nil
				}
			}
			s.mu.Unlock()
		}()
		tmp, e := os.CreateTemp(s.work, "ls-")
		if e != nil {
			w.WriteHeader(500)
			return
		}
		defer os.Remove(tmp.Name())
		n, e := io.Copy(tmp, io.LimitReader(r.Body, f.Size+1))
		ce := tmp.Close()
		if e != nil || ce != nil || n != f.Size {
			w.WriteHeader(400)
			return
		}
		if e = s.commit(tmp.Name(), File{Name: f.Name, Size: f.Size, SHA256: strings.ToLower(f.Hash)}); e != nil {
			w.WriteHeader(422)
			return
		}
		good = true
		w.WriteHeader(200)
	})
	m.HandleFunc("POST "+ls+"/cancel", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		defer s.mu.Unlock()
		q := s.session
		host, _, _ := net.SplitHostPort(r.RemoteAddr)
		if q == nil || q.host != host || r.URL.Query().Get("sessionId") != q.auth.ID {
			w.WriteHeader(403)
			return
		}
		s.session = nil
		w.WriteHeader(200)
	})
}

// SendLocal uses an explicitly entered LAN address. User confirmation binds
// this transfer to the presented certificate before any filename is sent.
func (s *Server) SendLocal(ctx context.Context, addr, path string, trust func(string) bool) error {
	ip, e := netip.ParseAddr(addr)
	if e != nil || !(ip.IsPrivate() || ip.IsLoopback()) {
		return errors.New("请输入局域网 IP / Enter a LAN IP address")
	}
	host := net.JoinHostPort(ip.String(), "53317")
	var pin string
	dial := net.Dialer{Timeout: 6 * time.Second}
	conn, e := tls.DialWithDialer(&dial, "tcp", host, &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionTLS12})
	if e != nil {
		return e
	}
	cert := conn.ConnectionState().PeerCertificates[0]
	conn.Close()
	sum := sha256.Sum256(cert.Raw)
	pin = hex.EncodeToString(sum[:])
	if !trust(pin) {
		return context.Canceled
	}
	tr := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true, VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
		if len(raw) > 0 && sha256.Sum256(raw[0]) == sum {
			return nil
		}
		return errors.New("certificate changed")
	}}}
	defer tr.CloseIdleConnections()
	client := http.Client{Transport: tr, Timeout: 30 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	file, e := os.Open(path)
	if e != nil {
		return e
	}
	defer file.Close()
	info, e := file.Stat()
	if e != nil {
		return e
	}
	if !info.Mode().IsRegular() || info.Size() > MaxSize {
		return errors.New("invalid file")
	}
	id := token()
	b := batch{s.info(), map[string]lsFile{id: {ID: id, Name: filepath.Base(path), Size: info.Size(), Type: "application/octet-stream"}}}
	body, _ := json.Marshal(b)
	base := "https://" + host + ls
	post := func(endpoint string, data io.Reader, size int64, mime string) (*http.Response, error) {
		req, e := http.NewRequestWithContext(ctx, "POST", base+endpoint, data)
		if e != nil {
			return nil, e
		}
		req.ContentLength = size
		req.Header.Set("Content-Type", mime)
		resp, e := client.Do(req)
		if e != nil {
			return nil, e
		}
		if resp.StatusCode/100 != 2 {
			resp.Body.Close()
			return nil, fmt.Errorf("LocalSend: HTTP %d", resp.StatusCode)
		}
		return resp, nil
	}
	r, e := post("/prepare-upload", bytes.NewReader(body), int64(len(body)), "application/json")
	if e != nil {
		return e
	}
	var a approval
	e = json.NewDecoder(io.LimitReader(r.Body, 65536)).Decode(&a)
	r.Body.Close()
	if e != nil || a.ID == "" || a.Files[id] == "" {
		return errors.New("接收端未接受 / Transfer not accepted")
	}
	query := url.Values{"sessionId": {a.ID}, "fileId": {id}, "token": {a.Files[id]}}
	r, e = post("/upload?"+query.Encode(), file, info.Size(), "application/octet-stream")
	if e == nil {
		r.Body.Close()
		s.setStatus("已发送 / Sent: " + info.Name())
	}
	return e
}
