package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func realtimeWebSocketURL(base, model string) (string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = defaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		return "", fmt.Errorf("realtime websocket needs an http or https provider")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/realtime"
	q := u.Query()
	q.Set("model", model)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *appServer) handleRealtimeWS(w http.ResponseWriter, r *http.Request) {
	if s.agent == nil {
		http.Error(w, "no agent", http.StatusServiceUnavailable)
		return
	}
	if s.realtimeCallOpen() {
		http.Error(w, "a call is already open", http.StatusConflict)
		return
	}
	s.mu.Lock()
	busy := s.busy
	s.mu.Unlock()
	if busy {
		http.Error(w, "finish the current run before calling", http.StatusConflict)
		return
	}
	c, ep, err := s.agent.realtimeClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	wsURL, err := realtimeWebSocketURL(c.BaseURL, ep.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	ctx, cancelDial := context.WithTimeout(r.Context(), 20*time.Second)
	up, err := dialRealtimeWebSocket(ctx, wsURL, c.APIKey)
	cancelDial()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		up.Close()
		http.Error(w, "websocket unavailable", http.StatusInternalServerError)
		return
	}
	client, _, err := hj.Hijack()
	if err != nil {
		up.Close()
		return
	}
	if err := acceptWebSocket(client, r.Header.Get("Sec-WebSocket-Key")); err != nil {
		client.Close()
		up.Close()
		return
	}
	gen := s.beginRealtimeCall()
	proxyWebSocket(client, up, s.noteRealtimeClient, s.teeRealtimeAudio)
	s.endRealtimeCallGen(gen)
}

func dialRealtimeWebSocket(ctx context.Context, rawURL, apiKey string) (net.Conn, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	host := u.Host
	addr := host
	if _, _, err := net.SplitHostPort(host); err != nil {
		if u.Scheme == "wss" {
			addr = host + ":443"
		} else {
			addr = host + ":80"
		}
	}
	var conn net.Conn
	if u.Scheme == "wss" {
		conn, err = (&tls.Dialer{Config: &tls.Config{ServerName: u.Hostname()}}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		conn.Close()
		return nil, err
	}
	secKey := base64.StdEncoding.EncodeToString(key)
	path := u.RequestURI()
	if path == "" {
		path = "/"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "GET %s HTTP/1.1\r\n", path)
	fmt.Fprintf(&b, "Host: %s\r\n", host)
	b.WriteString("Upgrade: websocket\r\n")
	b.WriteString("Connection: Upgrade\r\n")
	fmt.Fprintf(&b, "Sec-WebSocket-Key: %s\r\n", secKey)
	b.WriteString("Sec-WebSocket-Version: 13\r\n")
	if strings.TrimSpace(apiKey) != "" {
		fmt.Fprintf(&b, "Authorization: Bearer %s\r\n", strings.TrimSpace(apiKey))
	}
	b.WriteString("\r\n")
	if _, err := io.WriteString(conn, b.String()); err != nil {
		conn.Close()
		return nil, err
	}
	br := bufio.NewReader(conn)
	status, err := br.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, err
	}
	if !strings.Contains(status, "101") {
		rest, _ := io.ReadAll(io.LimitReader(br, 1024))
		conn.Close()
		msg := strings.TrimSpace(status + string(rest))
		return nil, fmt.Errorf("realtime websocket: %s", truncate(msg, 300))
	}
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			conn.Close()
			return nil, err
		}
		if line == "\r\n" || line == "\n" {
			break
		}
	}
	if br.Buffered() > 0 {
		return &prefixConn{Conn: conn, r: br}, nil
	}
	return conn, nil
}

type prefixConn struct {
	net.Conn
	r *bufio.Reader
}

func (p *prefixConn) Read(b []byte) (int, error) { return p.r.Read(b) }

func acceptWebSocket(conn net.Conn, key string) error {
	sum := sha1.Sum([]byte(strings.TrimSpace(key) + wsGUID))
	accept := base64.StdEncoding.EncodeToString(sum[:])
	_, err := fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", accept)
	return err
}

type wsWriter struct {
	mu   sync.Mutex
	conn net.Conn
	mask bool
}

func (w *wsWriter) write(fin bool, opcode byte, payload []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return writeWSFrame(w.conn, fin, opcode, payload, w.mask)
}

func proxyWebSocket(client, upstream net.Conn, onClient, onUpstream func([]byte)) {
	defer client.Close()
	defer upstream.Close()
	toUp := &wsWriter{conn: upstream, mask: true}
	toClient := &wsWriter{conn: client, mask: false}
	done := make(chan struct{}, 2)
	go func() {
		copyWS(client, toClient, toUp, onClient)
		done <- struct{}{}
	}()
	go func() {
		copyWS(upstream, toUp, toClient, onUpstream)
		done <- struct{}{}
	}()
	<-done
	client.Close()
	upstream.Close()
	<-done
}

func copyWS(src net.Conn, back, forward *wsWriter, onText func([]byte)) {
	var text []byte
	texting := false
	for {
		fin, op, payload, err := readWSFrame(src)
		if err != nil {
			return
		}
		switch op {
		case 0x8:
			_ = forward.write(true, 0x8, payload)
			return
		case 0x9:
			_ = back.write(true, 0xA, payload)
		case 0xA:
		default:
			if onText != nil && (op == 0x1 || (op == 0 && texting)) {
				if op == 0x1 {
					text = append(text[:0], payload...)
					texting = true
				} else {
					text = append(text, payload...)
				}
				if fin {
					onText(append([]byte(nil), text...))
					text = text[:0]
					texting = false
				}
			} else if op != 0 {
				text = nil
				texting = false
			}
			if err := forward.write(fin, op, payload); err != nil {
				return
			}
		}
	}
}

// speakerPipe is one growing WAV response. A new listener replaces the previous one.
type speakerPipe struct {
	ch   chan []byte
	stop chan struct{}
}

func wavStreamHeader(rate, dataBytes int) []byte {
	h := make([]byte, 44)
	copy(h[0:], "RIFF")
	putLE32(h[4:], uint32(36+dataBytes))
	copy(h[8:], "WAVEfmt ")
	putLE32(h[16:], 16)
	putLE16(h[20:], 1)
	putLE16(h[22:], 1)
	putLE32(h[24:], uint32(rate))
	putLE32(h[28:], uint32(rate*2))
	putLE16(h[32:], 2)
	putLE16(h[34:], 16)
	copy(h[36:], "data")
	putLE32(h[40:], uint32(dataBytes))
	return h
}

func putLE16(b []byte, v uint16) { b[0] = byte(v); b[1] = byte(v >> 8) }
func putLE32(b []byte, v uint32) {
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
}

// handleRealtimeSpeaker is one WAV the window plays while bytes are appended.
// No Content-Length: this WebKit starts playback immediately and keeps going.
func (s *appServer) handleRealtimeSpeaker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "stream unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "audio/wav")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := w.Write(wavStreamHeader(24000, 24000*2*60*60)); err != nil {
		return
	}
	flusher.Flush()
	p := &speakerPipe{ch: make(chan []byte, 512), stop: make(chan struct{})}
	s.speakerMu.Lock()
	old := s.speaker
	s.speaker = p
	s.speakerMu.Unlock()
	if old != nil {
		close(old.stop)
	}
	defer func() {
		s.speakerMu.Lock()
		if s.speaker == p {
			s.speaker = nil
		}
		s.speakerMu.Unlock()
	}()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-p.stop:
			return
		case pcm := <-p.ch:
			if _, err := w.Write(pcm); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *appServer) writeSpeaker(pcm []byte) {
	if len(pcm) == 0 {
		return
	}
	b := append([]byte(nil), pcm...)
	s.speakerMu.Lock()
	p := s.speaker
	s.speakerMu.Unlock()
	if p == nil {
		return
	}
	select {
	case <-p.stop:
	case p.ch <- b:
	default:
	}
}

func (s *appServer) noteRealtimeClient(payload []byte) {
	var ev struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(payload, &ev) != nil || ev.Type != "response.cancel" {
		return
	}
	s.speakerMu.Lock()
	s.speakerDrop = true
	s.speakerMu.Unlock()
}

func (s *appServer) teeRealtimeAudio(payload []byte) {
	var ev struct {
		Type  string `json:"type"`
		Delta string `json:"delta"`
	}
	if json.Unmarshal(payload, &ev) != nil {
		return
	}
	switch ev.Type {
	case "response.created":
		s.speakerMu.Lock()
		s.speakerDrop = false
		s.speakerMu.Unlock()
		return
	case "input_audio_buffer.speech_started":
		// The window keeps playing bytes it already has. Drop the rest of this
		// reply until the next one starts.
		s.speakerMu.Lock()
		s.speakerDrop = true
		s.speakerMu.Unlock()
		return
	}
	s.speakerMu.Lock()
	drop := s.speakerDrop
	s.speakerMu.Unlock()
	if drop || (ev.Type != "response.output_audio.delta" && ev.Type != "response.audio.delta") {
		return
	}
	raw, err := base64.StdEncoding.DecodeString(ev.Delta)
	if err != nil || len(raw) < 2 {
		return
	}
	if len(raw)%2 == 1 {
		raw = raw[:len(raw)-1]
	}
	s.writeSpeaker(raw)
}

func readWSFrame(r io.Reader) (fin bool, opcode byte, payload []byte, err error) {
	var hdr [2]byte
	if _, err = io.ReadFull(r, hdr[:]); err != nil {
		return
	}
	fin = hdr[0]&0x80 != 0
	opcode = hdr[0] & 0x0f
	masked := hdr[1]&0x80 != 0
	n := uint64(hdr[1] & 0x7f)
	switch n {
	case 126:
		var ext [2]byte
		if _, err = io.ReadFull(r, ext[:]); err != nil {
			return
		}
		n = uint64(ext[0])<<8 | uint64(ext[1])
	case 127:
		var ext [8]byte
		if _, err = io.ReadFull(r, ext[:]); err != nil {
			return
		}
		n = 0
		for _, b := range ext {
			n = n<<8 | uint64(b)
		}
	}
	if n > 8<<20 {
		err = fmt.Errorf("websocket frame too large")
		return
	}
	var key [4]byte
	if masked {
		if _, err = io.ReadFull(r, key[:]); err != nil {
			return
		}
	}
	payload = make([]byte, n)
	if _, err = io.ReadFull(r, payload); err != nil {
		return
	}
	if masked {
		for i := range payload {
			payload[i] ^= key[i%4]
		}
	}
	return
}

func writeWSFrame(w io.Writer, fin bool, opcode byte, payload []byte, mask bool) error {
	head := make([]byte, 0, 14)
	b0 := opcode
	if fin {
		b0 |= 0x80
	}
	head = append(head, b0)
	n := len(payload)
	var maskBit byte
	if mask {
		maskBit = 0x80
	}
	switch {
	case n < 126:
		head = append(head, byte(n)|maskBit)
	case n <= 65535:
		head = append(head, 126|maskBit, byte(n>>8), byte(n))
	default:
		head = append(head, 127|maskBit)
		var ext [8]byte
		nn := uint64(n)
		for i := 7; i >= 0; i-- {
			ext[i] = byte(nn)
			nn >>= 8
		}
		head = append(head, ext[:]...)
	}
	var key [4]byte
	body := payload
	if mask {
		if _, err := rand.Read(key[:]); err != nil {
			return err
		}
		head = append(head, key[:]...)
		body = make([]byte, len(payload))
		for i := range payload {
			body[i] = payload[i] ^ key[i%4]
		}
	}
	if _, err := w.Write(head); err != nil {
		return err
	}
	_, err := w.Write(body)
	return err
}
