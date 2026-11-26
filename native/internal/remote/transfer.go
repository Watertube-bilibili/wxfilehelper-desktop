// SPDX-License-Identifier: AGPL-3.0-only
// Package remote exchanges files over a reliable, ordered WebRTC data channel.
// PeerServer forwards SDP and ICE; file bytes travel on the negotiated UDP path.
package remote

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Watertube-bilibili/wxfilehelper-desktop/native/internal/wechat"
	"github.com/coder/websocket"
	"github.com/pion/ice/v4"
	"github.com/pion/webrtc/v4"
)

const Chunk = 16384
const MaxSize int64 = 8 << 30
const label = "ultra-file-v1"

var codeRE = regexp.MustCompile("^[a-z2-7]{12}$")

func newCode() string {
	var b [8]byte
	rand.Read(b[:])
	return strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])[:12])
}

type Status struct {
	Code, State, Name, Error, Saved string
	Size, Done                      int64
}
type control struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
	Size int64  `json:"size"`
	Hash string `json:"hash,omitempty"`
}
type payload struct {
	Type          string                     `json:"type"`
	ID            string                     `json:"connectionId"`
	SDP           *webrtc.SessionDescription `json:"sdp,omitempty"`
	Candidate     *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
	Label         string                     `json:"label,omitempty"`
	Serialization string                     `json:"serialization,omitempty"`
	Reliable      bool                       `json:"reliable,omitempty"`
}
type signal struct {
	Type    string  `json:"type"`
	Dst     string  `json:"dst,omitempty"`
	Src     string  `json:"src,omitempty"`
	Payload payload `json:"payload,omitempty"`
}
type Session struct {
	mu                 sync.Mutex
	ctx                context.Context
	cancel             context.CancelFunc
	sendPath, peer, id string
	state              Status
	sock               *websocket.Conn
	pc                 *webrtc.PeerConnection
	dc                 *webrtc.DataChannel
	candidates         []webrtc.ICECandidateInit
	file               *os.File
	temp, target       string
	sum                hash.Hash
}

func Start(target, path string) (*Session, error) {
	s := &Session{state: Status{Code: newCode(), State: "signaling"}, sendPath: path}
	if path != "" {
		target = strings.ToLower(strings.TrimSpace(target))
		if !codeRE.MatchString(target) {
			return nil, errors.New("请输入接收方的 12 位配对码 / Enter a 12-character receiver code")
		}
		info, e := os.Stat(path)
		if e != nil {
			return nil, e
		}
		if !info.Mode().IsRegular() || info.Size() > MaxSize {
			return nil, errors.New("单个文件最大 8 GiB / Maximum file size is 8 GiB")
		}
		s.peer = "ultrafile-" + target
		s.id = newCode()
		s.state.Name = info.Name()
		s.state.Size = info.Size()
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	go s.run()
	return s, nil
}
func (s *Session) Snapshot() Status { s.mu.Lock(); defer s.mu.Unlock(); return s.state }
func (s *Session) Close() {
	s.mu.Lock()
	dc := s.dc
	st := s.state.State
	s.mu.Unlock()
	if dc != nil && dc.ReadyState() == webrtc.DataChannelStateOpen && st != "complete" {
		s.sendControl(control{Type: "cancel"})
	}
	s.finish("cancelled", nil)
}
func (s *Session) finish(state string, e error) {
	s.mu.Lock()
	if s.ctx.Err() != nil {
		s.mu.Unlock()
		return
	}
	if s.state.State == "complete" {
		state = "complete"
		e = nil
	}
	s.state.State = state
	if e != nil {
		s.state.Error = e.Error()
	}
	s.cancel()
	pc, ws := s.pc, s.sock
	s.pc = nil
	s.sock = nil
	if s.file != nil {
		s.file.Close()
		s.file = nil
	}
	if s.temp != "" {
		os.Remove(s.temp)
		s.temp = ""
	}
	s.mu.Unlock()
	// Pion callbacks may run on a channel's read goroutine.
	if pc != nil {
		go pc.Close()
	}
	if ws != nil {
		go ws.CloseNow()
	}
}
func (s *Session) fail(e error) {
	if e != nil {
		s.finish("error", e)
	}
}
func (s *Session) signal(kind string, p payload) error {
	s.mu.Lock()
	ws, dst := s.sock, s.peer
	s.mu.Unlock()
	if ws == nil {
		return errors.New("signaling closed")
	}
	b, e := json.Marshal(signal{Type: kind, Dst: dst, Payload: p})
	if e != nil {
		return e
	}
	ctx, cancel := context.WithTimeout(s.ctx, 8*time.Second)
	defer cancel()
	return ws.Write(ctx, websocket.MessageText, b)
}
func (s *Session) run() {
	q := url.Values{"key": {"peerjs"}, "id": {"ultrafile-" + s.state.Code}, "token": {newCode()}, "version": {"1.5.5"}}
	ctx, cancel := context.WithTimeout(s.ctx, 20*time.Second)
	ws, _, err := websocket.Dial(ctx, "wss://0.peerjs.com/peerjs?"+q.Encode(), nil)
	cancel()
	if err != nil {
		s.fail(err)
		return
	}
	s.mu.Lock()
	if s.ctx.Err() != nil {
		s.mu.Unlock()
		ws.CloseNow()
		return
	}
	s.sock = ws
	s.mu.Unlock()
	ws.SetReadLimit(128 << 10)
	go func() {
		tick := time.NewTicker(5 * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-tick.C:
				if e := s.signal("HEARTBEAT", payload{}); e != nil {
					return
				}
			}
		}
	}()
	for {
		_, b, e := ws.Read(s.ctx)
		if e != nil {
			s.mu.Lock()
			linked := s.dc != nil && s.dc.ReadyState() == webrtc.DataChannelStateOpen
			s.mu.Unlock()
			if !linked {
				s.fail(e)
			}
			return
		}
		var v signal
		if json.Unmarshal(b, &v) != nil {
			continue
		}
		if e = s.onSignal(v); e != nil {
			s.fail(e)
			return
		}
	}
}
func (s *Session) onSignal(v signal) error {
	s.mu.Lock()
	peer, id, pc := s.peer, s.id, s.pc
	s.mu.Unlock()
	switch v.Type {
	case "OPEN":
		if s.sendPath == "" {
			s.mu.Lock()
			s.state.State = "waiting"
			s.mu.Unlock()
			return nil
		}
		pc, e := s.makePeer()
		if e != nil {
			return e
		}
		dc, e := pc.CreateDataChannel(label, nil)
		if e != nil {
			return e
		}
		s.bind(dc)
		offer, e := pc.CreateOffer(nil)
		if e != nil {
			return e
		}
		return s.description(pc, offer, "OFFER")
	case "OFFER":
		if s.sendPath != "" || pc != nil || !strings.HasPrefix(v.Src, "ultrafile-") || !codeRE.MatchString(strings.TrimPrefix(v.Src, "ultrafile-")) || v.Payload.Label != label || v.Payload.ID == "" || v.Payload.SDP == nil {
			return nil
		}
		s.mu.Lock()
		s.peer = v.Src
		s.id = v.Payload.ID
		s.mu.Unlock()
		pc, e := s.makePeer()
		if e != nil {
			return e
		}
		pc.OnDataChannel(func(dc *webrtc.DataChannel) {
			if dc.Label() == label {
				s.bind(dc)
			}
		})
		if e = s.setRemote(pc, *v.Payload.SDP); e != nil {
			return e
		}
		answer, e := pc.CreateAnswer(nil)
		if e != nil {
			return e
		}
		return s.description(pc, answer, "ANSWER")
	case "ANSWER":
		if pc != nil && v.Src == peer && v.Payload.ID == id && v.Payload.SDP != nil && pc.RemoteDescription() == nil {
			return s.setRemote(pc, *v.Payload.SDP)
		}
	case "CANDIDATE":
		if pc != nil && v.Src == peer && v.Payload.ID == id && v.Payload.Candidate != nil {
			if pc.RemoteDescription() != nil {
				return pc.AddICECandidate(*v.Payload.Candidate)
			}
			if len(s.candidates) < 64 {
				s.candidates = append(s.candidates, *v.Payload.Candidate)
			}
		}
	case "EXPIRE", "ID-TAKEN", "ERROR", "INVALID-KEY":
		return errors.New("配对码不可用，请让接收方重新连接 / Pairing code unavailable")
	}
	return nil
}
func (s *Session) setRemote(pc *webrtc.PeerConnection, desc webrtc.SessionDescription) error {
	if err := pc.SetRemoteDescription(desc); err != nil {
		return err
	}
	for _, v := range s.candidates {
		if e := pc.AddICECandidate(v); e != nil {
			return e
		}
	}
	s.candidates = nil
	return nil
}
func (s *Session) makePeer() (*webrtc.PeerConnection, error) {
	var settings webrtc.SettingEngine
	settings.SetNetworkTypes([]webrtc.NetworkType{webrtc.NetworkTypeUDP4, webrtc.NetworkTypeUDP6})
	settings.SetICEMulticastDNSMode(ice.MulticastDNSModeQueryAndGather)
	pc, e := webrtc.NewAPI(webrtc.WithSettingEngine(settings)).NewPeerConnection(webrtc.Configuration{ICEServers: []webrtc.ICEServer{{URLs: []string{"stun:stun.cloudflare.com:3478"}}}})
	if e != nil {
		return nil, e
	}
	s.mu.Lock()
	if s.ctx.Err() != nil {
		s.mu.Unlock()
		pc.Close()
		return nil, context.Canceled
	}
	s.pc = pc
	s.state.State = "connecting"
	s.mu.Unlock()
	pc.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		if state == webrtc.PeerConnectionStateFailed {
			s.fail(errors.New("无法建立 UDP 直连，请切换网络或使用同一 Wi-Fi / UDP connection failed"))
		}
	})
	go func() {
		select {
		case <-time.After(40 * time.Second):
			if s.Snapshot().State == "connecting" {
				s.fail(errors.New("UDP 连接超时 / UDP connection timed out"))
			}
		case <-s.ctx.Done():
		}
	}()
	return pc, nil
}
func (s *Session) description(pc *webrtc.PeerConnection, d webrtc.SessionDescription, kind string) error {
	ready := webrtc.GatheringCompletePromise(pc)
	if e := pc.SetLocalDescription(d); e != nil {
		return e
	}
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
	case <-s.ctx.Done():
		return s.ctx.Err()
	}
	return s.signal(kind, payload{Type: "data", ID: s.id, SDP: pc.LocalDescription(), Label: label, Serialization: "raw", Reliable: true})
}
func (s *Session) bind(dc *webrtc.DataChannel) {
	s.mu.Lock()
	s.dc = dc
	if s.sendPath == "" && s.state.State == "connecting" {
		s.state.State = "connected"
	}
	s.mu.Unlock()
	dc.OnClose(func() {
		st := s.Snapshot().State
		if st == "complete" {
			s.finish("complete", nil)
		} else if st != "cancelled" {
			s.fail(errors.New("连接已关闭 / Connection closed"))
		}
	})
	dc.OnError(s.fail)
	dc.OnOpen(func() {
		s.mu.Lock()
		if s.sendPath != "" {
			s.state.State = "offered"
		} else if s.state.State == "connecting" {
			s.state.State = "connected"
		}
		st := s.state
		s.mu.Unlock()
		if s.sendPath != "" {
			s.fail(s.sendControl(control{Type: "offer", Name: st.Name, Size: st.Size}))
		}
	})
	dc.OnMessage(func(m webrtc.DataChannelMessage) {
		if !m.IsString {
			s.fail(s.writeChunk(m.Data))
			return
		}
		if len(m.Data) > 4096 {
			s.fail(errors.New("invalid control data"))
			return
		}
		var v control
		if json.Unmarshal(m.Data, &v) != nil {
			return
		}
		s.fail(s.onControl(v))
	})
}
func (s *Session) sendControl(v control) error {
	b, _ := json.Marshal(v)
	s.mu.Lock()
	dc := s.dc
	s.mu.Unlock()
	if dc == nil {
		return errors.New("channel unavailable")
	}
	return dc.SendText(string(b))
}
func (s *Session) onControl(v control) error {
	s.mu.Lock()
	st := s.state
	switch v.Type {
	case "offer":
		if s.sendPath != "" || st.State != "connected" || v.Size < 0 || v.Size > MaxSize || v.Name == "" {
			s.mu.Unlock()
			return errors.New("invalid incoming file")
		}
		s.state.Name = wechat.SafeFilename(v.Name)
		s.state.Size = v.Size
		s.state.State = "incoming"
		s.mu.Unlock()
		return nil
	case "accept":
		if s.sendPath == "" || st.State != "offered" {
			s.mu.Unlock()
			return errors.New("unexpected acceptance")
		}
		s.state.State = "sending"
		s.mu.Unlock()
		go func() { s.fail(s.sendFile()) }()
		return nil
	case "finish":
		if st.State != "receiving" || s.file == nil || st.Done != st.Size || hex.EncodeToString(s.sum.Sum(nil)) != v.Hash {
			s.mu.Unlock()
			return errors.New("文件校验失败 / File integrity check failed")
		}
		e := s.file.Close()
		s.file = nil
		if e == nil {
			e = os.Rename(s.temp, s.target)
		}
		if e == nil {
			s.temp = ""
			s.state.Saved = s.target
			s.state.State = "complete"
		}
		s.mu.Unlock()
		if e != nil {
			return e
		}
		e = s.sendControl(control{Type: "ack", Hash: v.Hash})
		time.AfterFunc(3*time.Second, func() { s.finish("complete", nil) })
		return e
	case "ack":
		if st.State == "confirming" && s.sum != nil && v.Hash == hex.EncodeToString(s.sum.Sum(nil)) {
			s.mu.Unlock()
			s.finish("complete", nil)
			return nil
		}
	case "cancel":
		s.mu.Unlock()
		s.finish("cancelled", nil)
		return nil
	}
	s.mu.Unlock()
	return nil
}
func (s *Session) Accept(path string) error {
	s.mu.Lock()
	if s.state.State != "incoming" {
		s.mu.Unlock()
		return errors.New("no pending file")
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".ultrafile-*.part")
	if e != nil {
		s.mu.Unlock()
		return e
	}
	s.file = f
	s.temp = f.Name()
	s.target = path
	s.sum = sha256.New()
	s.state.State = "receiving"
	s.mu.Unlock()
	e = s.sendControl(control{Type: "accept"})
	s.fail(e)
	return e
}
func (s *Session) writeChunk(b []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file == nil || len(b) > Chunk || s.state.Done+int64(len(b)) > s.state.Size {
		return errors.New("invalid UDP file chunk")
	}
	n, e := s.file.Write(b)
	if e != nil {
		return e
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	s.sum.Write(b)
	s.state.Done += int64(n)
	return nil
}
func (s *Session) sendFile() error {
	f, e := os.Open(s.sendPath)
	if e != nil {
		return e
	}
	defer f.Close()
	s.mu.Lock()
	dc := s.dc
	size := s.state.Size
	s.mu.Unlock()
	sum := sha256.New()
	buf := make([]byte, Chunk)
	var total int64
	for {
		if e = s.ctx.Err(); e != nil {
			return e
		}
		n, err := f.Read(buf)
		if n > 0 {
			for dc.BufferedAmount() > 512<<10 {
				select {
				case <-s.ctx.Done():
					return s.ctx.Err()
				case <-time.After(5 * time.Millisecond):
				}
			}
			total += int64(n)
			if total > size {
				return errors.New("file changed during transfer")
			}
			if e = dc.Send(buf[:n]); e != nil {
				return e
			}
			sum.Write(buf[:n])
			s.mu.Lock()
			s.state.Done = total
			s.mu.Unlock()
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if total != size {
		return fmt.Errorf("file changed: %d / %d", total, size)
	}
	s.mu.Lock()
	s.state.State = "confirming"
	s.sum = sum
	s.mu.Unlock()
	return s.sendControl(control{Type: "finish", Hash: hex.EncodeToString(sum.Sum(nil))})
}
