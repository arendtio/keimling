//go:build linux

package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	portalBusName = "org.freedesktop.portal.Desktop"
	portalPath    = "/org/freedesktop/portal/desktop"
	portalCast    = "org.freedesktop.portal.ScreenCast"
)

// openScreenCast opens the desktop chooser for a screen or a window and
// returns a live PipeWire picture. WebKit's own capture dies while agreeing
// on a picture format, so this process reads the stream.
func openScreenCast(ctx context.Context) (*screenCast, error) {
	bus, err := dialSessionBus()
	if err != nil {
		return nil, err
	}
	name, err := bus.hello()
	if err != nil {
		bus.Close()
		return nil, err
	}
	handle := portalToken("keimling")
	sessionTok := portalToken("ksess")
	code, body, err := bus.portalRequest(ctx, name, portalCast, "CreateSession", "a{sv}", createSessionBody(handle, sessionTok), handle)
	if err != nil {
		bus.Close()
		return nil, err
	}
	if code == 1 {
		bus.Close()
		return nil, errScreenCancelled
	}
	if code != 0 {
		bus.Close()
		return nil, fmt.Errorf("screen share failed (%d)", code)
	}
	_, off, err := dbusUint32(body, 0)
	if err != nil {
		bus.Close()
		return nil, err
	}
	session, err := dbusDictString(body, off, "session_handle")
	if err != nil {
		bus.Close()
		return nil, err
	}
	fail := func(err error) (*screenCast, error) {
		_ = bus.closeSession(session)
		bus.Close()
		return nil, err
	}
	srcTok := portalToken("ksrc")
	code, _, err = bus.portalRequest(ctx, name, portalCast, "SelectSources", "oa{sv}", selectSourcesBody(session, srcTok), srcTok)
	if err != nil {
		return fail(err)
	}
	if code == 1 {
		return fail(errScreenCancelled)
	}
	if code != 0 {
		return fail(fmt.Errorf("screen share failed (%d)", code))
	}
	startTok := portalToken("kstart")
	code, body, err = bus.portalRequest(ctx, name, portalCast, "Start", "osa{sv}", startCastBody(session, startTok), startTok)
	if err != nil {
		return fail(err)
	}
	if code == 1 {
		return fail(errScreenCancelled)
	}
	if code != 0 {
		return fail(fmt.Errorf("screen share failed (%d)", code))
	}
	node, err := portalStreamNode(body)
	if err != nil {
		return fail(err)
	}
	fd, err := bus.openPipeWire(session)
	if err != nil {
		return fail(err)
	}
	cast := &screenCast{fd: fd}
	cast.run = func(ctx context.Context, write func([]byte) error) error {
		return streamPipeWire(ctx, cast.takeFD(), node, write)
	}
	cast.stop = func() {
		if fd := cast.takeFD(); fd >= 0 {
			_ = syscall.Close(fd)
		}
		_ = bus.c.SetReadDeadline(time.Now().Add(2 * time.Second))
		_ = bus.closeSession(session)
		bus.Close()
	}
	return cast, nil
}

func portalToken(prefix string) string {
	return prefix + strconv.FormatInt(time.Now().UnixNano(), 36)
}

func createSessionBody(handle, session string) []byte {
	var dict []byte
	dict = dbusDictEntry(dict, "handle_token", "s", dbusStringBytes(handle))
	dict = dbusDictEntry(dict, "session_handle_token", "s", dbusStringBytes(session))
	return appendDBusArray(nil, 8, dict)
}

func selectSourcesBody(session, token string) []byte {
	body := appendDBusString(nil, session)
	var dict []byte
	dict = dbusDictEntry(dict, "types", "u", dbusUint32Bytes(1|2))
	dict = dbusDictEntry(dict, "multiple", "b", dbusBoolBytes(false))
	dict = dbusDictEntry(dict, "cursor_mode", "u", dbusUint32Bytes(2))
	dict = dbusDictEntry(dict, "handle_token", "s", dbusStringBytes(token))
	return appendDBusArray(body, 8, dict)
}

func startCastBody(session, token string) []byte {
	body := appendDBusString(nil, session)
	body = appendDBusString(body, "")
	var dict []byte
	dict = dbusDictEntry(dict, "handle_token", "s", dbusStringBytes(token))
	return appendDBusArray(body, 8, dict)
}

func portalStreamNode(body []byte) (uint32, error) {
	_, off, err := dbusUint32(body, 0)
	if err != nil {
		return 0, err
	}
	if _, off, err = dbusUint32(body, off); err != nil {
		return 0, err
	}
	for off%8 != 0 && off < len(body) {
		off++
	}
	key, off, err := dbusString(body, off)
	if err != nil {
		return 0, err
	}
	sig, off, err := dbusSignature(body, off)
	if err != nil {
		return 0, err
	}
	if key != "streams" || !strings.HasPrefix(sig, "a(u") {
		return 0, fmt.Errorf("screen share returned no stream")
	}
	for off%4 != 0 {
		off++
	}
	if _, off, err = dbusUint32(body, off); err != nil {
		return 0, err
	}
	for off%8 != 0 && off < len(body) {
		off++
	}
	node, _, err := dbusUint32(body, off)
	if err != nil || node == 0 {
		return 0, fmt.Errorf("screen share returned no stream")
	}
	return node, nil
}

// screenPipeWireArgs reads the portal stream without pinning a pixel format or
// framerate. The queue sits after the converter, so a burst of frames is
// dropped before encoding instead of stacking up as delay. A queue in front
// of the converter keeps this source from leaving pause.
func screenPipeWireArgs(node uint32) []string {
	return []string{
		"gst-launch-1.0", "-q",
		"pipewiresrc", "fd=3", "path=" + strconv.FormatUint(uint64(node), 10), "do-timestamp=true",
		"!", "videoconvert",
		"!", "queue", "max-size-buffers=1", "max-size-bytes=0", "max-size-time=0", "leaky=downstream",
		"!", "jpegenc", "quality=70",
		"!", "fdsink", "fd=1", "sync=false",
	}
}

func streamPipeWire(ctx context.Context, fd int, node uint32, write func([]byte) error) error {
	if fd < 0 {
		return fmt.Errorf("screen share has no picture stream")
	}
	pw := os.NewFile(uintptr(fd), "pipewire")
	if _, err := exec.LookPath("gst-launch-1.0"); err != nil {
		pw.Close()
		return fmt.Errorf("screen share needs gstreamer")
	}
	args := screenPipeWireArgs(node)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.ExtraFiles = []*os.File{pw}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		pw.Close()
		return err
	}
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Start(); err != nil {
		pw.Close()
		return err
	}
	pw.Close()
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()
	sent := false
	readErr := consumeJPEGs(stdout, func(frame []byte) error {
		sent = true
		return write(frame)
	})
	if ctx.Err() != nil || errors.Is(readErr, context.Canceled) {
		return nil
	}
	if !sent {
		msg := strings.TrimSpace(errBuf.String())
		if len(msg) > 400 {
			msg = msg[len(msg)-400:]
		}
		if msg == "" && readErr != nil {
			msg = readErr.Error()
		}
		if msg == "" {
			msg = "no picture"
		}
		return fmt.Errorf("screen share produced no picture: %s", msg)
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return readErr
	}
	return nil
}

func (s *screenCast) takeFD() int {
	fd := s.fd
	s.fd = -1
	return fd
}

type dbusClient struct {
	c       *net.UnixConn
	serial  uint32
	pending []byte
	fds     []int
	signals []dbusMsg
}

type dbusMsg struct {
	kind    byte
	path    string
	member  string
	errName string
	serial  uint32
	reply   uint32
	sig     string
	fdCount uint32
	fds     []int
	body    []byte
}

func dialSessionBus() (*dbusClient, error) {
	addr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	if addr == "" {
		return nil, fmt.Errorf("no desktop session bus")
	}
	network, address, err := dbusUnixAddress(addr)
	if err != nil {
		return nil, err
	}
	nc, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	uc, ok := nc.(*net.UnixConn)
	if !ok {
		nc.Close()
		return nil, fmt.Errorf("session bus is not a unix socket")
	}
	if _, err := uc.Write([]byte{0}); err != nil {
		uc.Close()
		return nil, err
	}
	uid := strconv.Itoa(os.Getuid())
	if _, err := io.WriteString(uc, "AUTH EXTERNAL "+hex.EncodeToString([]byte(uid))+"\r\n"); err != nil {
		uc.Close()
		return nil, err
	}
	okLine, err := readUnixLine(uc)
	if err != nil {
		uc.Close()
		return nil, err
	}
	if !strings.HasPrefix(okLine, "OK ") {
		uc.Close()
		return nil, fmt.Errorf("session bus refused the connection")
	}
	// The picture stream arrives as a file descriptor. The bus drops the
	// connection when a caller asks for one without negotiating it.
	if _, err := io.WriteString(uc, "NEGOTIATE_UNIX_FD\r\n"); err != nil {
		uc.Close()
		return nil, err
	}
	agree, err := readUnixLine(uc)
	if err != nil {
		uc.Close()
		return nil, err
	}
	if !strings.HasPrefix(agree, "AGREE_UNIX_FD") {
		uc.Close()
		return nil, fmt.Errorf("session bus cannot pass a picture stream")
	}
	if _, err := io.WriteString(uc, "BEGIN\r\n"); err != nil {
		uc.Close()
		return nil, err
	}
	return &dbusClient{c: uc}, nil
}

func readUnixLine(c net.Conn) (string, error) {
	var b []byte
	one := []byte{0}
	for {
		n, err := c.Read(one)
		if n == 1 {
			b = append(b, one[0])
			if one[0] == '\n' {
				return string(b), nil
			}
		}
		if err != nil {
			return "", err
		}
	}
}

func dbusUnixAddress(addr string) (network, address string, err error) {
	for _, part := range strings.Split(addr, ",") {
		part = strings.TrimSpace(part)
		switch {
		case strings.HasPrefix(part, "unix:path="):
			return "unix", strings.TrimPrefix(part, "unix:path="), nil
		case strings.HasPrefix(part, "unix:abstract="):
			return "unix", "\x00" + strings.TrimPrefix(part, "unix:abstract="), nil
		}
		if i := strings.Index(part, "path="); strings.HasPrefix(part, "unix:") && i >= 0 {
			rest := part[i+len("path="):]
			if cut := strings.IndexByte(rest, ','); cut >= 0 {
				rest = rest[:cut]
			}
			return "unix", rest, nil
		}
	}
	return "", "", fmt.Errorf("session bus address is not a unix socket")
}

func (d *dbusClient) Close() error {
	closeFDs(d.fds)
	d.fds = nil
	for _, msg := range d.signals {
		closeFDs(msg.fds)
	}
	d.signals = nil
	if d.c == nil {
		return nil
	}
	return d.c.Close()
}

func (d *dbusClient) hello() (string, error) {
	sig, body, fds, err := d.call("org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus", "Hello", "", nil)
	closeFDs(fds)
	if err != nil {
		return "", err
	}
	if sig != "s" {
		return "", fmt.Errorf("unexpected bus name")
	}
	name, _, err := dbusString(body, 0)
	return name, err
}

func (d *dbusClient) addMatch(path string) error {
	rule := "type='signal',interface='org.freedesktop.portal.Request',member='Response',path='" + path + "'"
	_, _, fds, err := d.call("org.freedesktop.DBus", "/org/freedesktop/DBus", "org.freedesktop.DBus", "AddMatch", "s", dbusStringBytes(rule))
	closeFDs(fds)
	return err
}

func (d *dbusClient) portalRequest(ctx context.Context, unique, iface, member, sig string, body []byte, token string) (uint32, []byte, error) {
	reqPath := portalRequestPath(unique, token)
	if err := d.addMatch(reqPath); err != nil {
		return 0, nil, err
	}
	_, reply, fds, err := d.call(portalBusName, portalPath, iface, member, sig, body)
	closeFDs(fds)
	if err != nil {
		return 0, nil, err
	}
	got, _, err := dbusString(reply, 0)
	if err != nil {
		return 0, nil, err
	}
	if got != "" && got != reqPath {
		if err := d.addMatch(got); err != nil {
			return 0, nil, err
		}
		reqPath = got
	}
	return d.waitResults(ctx, reqPath)
}

func (d *dbusClient) waitResults(ctx context.Context, path string) (uint32, []byte, error) {
	for {
		if err := ctx.Err(); err != nil {
			return 0, nil, err
		}
		var msg dbusMsg
		if len(d.signals) > 0 {
			msg = d.signals[0]
			d.signals = d.signals[1:]
		} else {
			if deadline, ok := ctx.Deadline(); ok {
				_ = d.c.SetReadDeadline(deadline)
			}
			var err error
			msg, err = d.read()
			if err != nil {
				if ctx.Err() != nil {
					return 0, nil, ctx.Err()
				}
				return 0, nil, err
			}
		}
		if msg.kind != 4 || msg.member != "Response" || (msg.path != "" && msg.path != path) {
			closeFDs(msg.fds)
			continue
		}
		closeFDs(msg.fds)
		code, _, err := dbusUint32(msg.body, 0)
		if err != nil {
			return 0, nil, err
		}
		return code, msg.body, nil
	}
}

func (d *dbusClient) openPipeWire(session string) (int, error) {
	body := appendDBusString(nil, session)
	body = appendDBusArray(body, 8, nil)
	sig, reply, fds, err := d.call(portalBusName, portalPath, portalCast, "OpenPipeWireRemote", "oa{sv}", body)
	if err != nil {
		closeFDs(fds)
		return -1, err
	}
	if sig != "h" {
		closeFDs(fds)
		return -1, fmt.Errorf("screen share returned no picture stream")
	}
	idx, _, err := dbusUint32(reply, 0)
	if err != nil || int(idx) >= len(fds) {
		closeFDs(fds)
		return -1, fmt.Errorf("screen share returned no picture stream")
	}
	fd := fds[idx]
	fds[idx] = -1
	closeFDs(fds)
	return fd, nil
}

func (d *dbusClient) closeSession(path string) error {
	_, _, fds, err := d.call(portalBusName, path, "org.freedesktop.portal.Session", "Close", "", nil)
	closeFDs(fds)
	return err
}

func portalRequestPath(uniqueName, token string) string {
	sender := strings.TrimPrefix(uniqueName, ":")
	sender = strings.ReplaceAll(sender, ".", "_")
	return "/org/freedesktop/portal/desktop/request/" + sender + "/" + token
}

func (d *dbusClient) call(dest, path, iface, member, sig string, body []byte) (string, []byte, []int, error) {
	d.serial++
	serial := d.serial
	frame, err := dbusMethod(dest, path, iface, member, sig, serial, body)
	if err != nil {
		return "", nil, nil, err
	}
	_ = d.c.SetWriteDeadline(time.Now().Add(15 * time.Second))
	if _, err := d.c.Write(frame); err != nil {
		return "", nil, nil, err
	}
	_ = d.c.SetReadDeadline(time.Now().Add(15 * time.Second))
	for {
		msg, err := d.read()
		if err != nil {
			return "", nil, nil, err
		}
		if msg.reply != serial {
			if msg.kind == 4 {
				d.signals = append(d.signals, msg)
			} else {
				closeFDs(msg.fds)
			}
			continue
		}
		if msg.kind == 3 {
			closeFDs(msg.fds)
			text, _, _ := dbusString(msg.body, 0)
			if text == "" {
				text = msg.errName
			}
			if text == "" {
				text = "desktop portal call failed"
			}
			return "", nil, nil, fmt.Errorf("%s", text)
		}
		if msg.kind == 2 {
			return msg.sig, msg.body, msg.fds, nil
		}
		closeFDs(msg.fds)
	}
}

func (d *dbusClient) read() (dbusMsg, error) {
	tmp := make([]byte, 64*1024)
	for {
		if msg, rest, fds, ok, err := takeDBusMessage(d.pending, d.fds); ok || err != nil {
			d.pending = rest
			d.fds = fds
			return msg, err
		}
		n, got, err := d.recv(tmp)
		if n > 0 {
			d.pending = append(d.pending, tmp[:n]...)
			d.fds = append(d.fds, got...)
		}
		if err != nil && n == 0 {
			return dbusMsg{}, err
		}
	}
}

func (d *dbusClient) recv(dst []byte) (int, []int, error) {
	oob := make([]byte, syscall.CmsgSpace(16*4))
	n, oobn, flags, _, err := d.c.ReadMsgUnix(dst, oob)
	var fds []int
	if oobn > 0 {
		if msgs, perr := syscall.ParseSocketControlMessage(oob[:oobn]); perr == nil {
			for _, m := range msgs {
				got, ferr := syscall.ParseUnixRights(&m)
				if ferr == nil {
					for _, fd := range got {
						syscall.CloseOnExec(fd)
					}
					fds = append(fds, got...)
				}
			}
		}
	}
	if flags&syscall.MSG_CTRUNC != 0 {
		closeFDs(fds)
		return n, nil, fmt.Errorf("screen share picture stream was truncated")
	}
	if err != nil {
		return n, fds, err
	}
	if n == 0 {
		return 0, fds, io.EOF
	}
	return n, fds, nil
}

func takeDBusMessage(buf []byte, fds []int) (dbusMsg, []byte, []int, bool, error) {
	if len(buf) < 16 {
		return dbusMsg{}, buf, fds, false, nil
	}
	if buf[0] != 'l' || buf[3] != 1 {
		return dbusMsg{}, nil, fds, false, fmt.Errorf("unexpected session bus message")
	}
	bodyLen := int(binary.LittleEndian.Uint32(buf[4:8]))
	headerLen := int(binary.LittleEndian.Uint32(buf[12:16]))
	total := 16 + headerLen
	if pad := total % 8; pad != 0 {
		total += 8 - pad
	}
	total += bodyLen
	if total < 16 || len(buf) < total {
		return dbusMsg{}, buf, fds, false, nil
	}
	msg := dbusMsg{kind: buf[1], body: append([]byte(nil), buf[total-bodyLen:total]...)}
	if err := parseHeaderFields(buf[16:16+headerLen], &msg); err != nil {
		return dbusMsg{}, nil, fds, false, err
	}
	if msg.fdCount > 0 {
		if len(fds) < int(msg.fdCount) {
			return dbusMsg{}, nil, fds, false, fmt.Errorf("screen share picture stream is incomplete")
		}
		msg.fds = append([]int(nil), fds[:msg.fdCount]...)
		fds = fds[msg.fdCount:]
	}
	return msg, append([]byte(nil), buf[total:]...), fds, true, nil
}

func closeFDs(fds []int) {
	for _, fd := range fds {
		if fd >= 0 {
			_ = syscall.Close(fd)
		}
	}
}

func parseHeaderFields(header []byte, msg *dbusMsg) error {
	off := 0
	for off < len(header) {
		for off < len(header) && (off%8) != 0 {
			off++
		}
		if off >= len(header) {
			break
		}
		code := header[off]
		sig, next, err := dbusSignature(header, off+1)
		if err != nil {
			return err
		}
		switch sig {
		case "s", "o":
			val, n, err := dbusString(header, next)
			if err != nil {
				return err
			}
			off = n
			switch code {
			case 1:
				msg.path = val
			case 3:
				msg.member = val
			case 4:
				msg.errName = val
			}
		case "g":
			val, n, err := dbusSignature(header, next)
			if err != nil {
				return err
			}
			off = n
			if code == 8 {
				msg.sig = val
			}
		case "u":
			val, n, err := dbusUint32(header, next)
			if err != nil {
				return err
			}
			off = n
			switch code {
			case 5:
				msg.reply = val
			case 9:
				msg.fdCount = val
			}
		default:
			return fmt.Errorf("unsupported bus header %s", sig)
		}
	}
	return nil
}

func dbusMethod(dest, path, iface, member, sig string, serial uint32, body []byte) ([]byte, error) {
	var fields []byte
	fields = dbusHeaderField(fields, 6, "s", dbusStringBytes(dest))
	fields = dbusHeaderField(fields, 1, "o", dbusStringBytes(path))
	fields = dbusHeaderField(fields, 2, "s", dbusStringBytes(iface))
	fields = dbusHeaderField(fields, 3, "s", dbusStringBytes(member))
	if sig != "" {
		fields = dbusHeaderField(fields, 8, "g", dbusSignatureBytes(sig))
	}
	header := make([]byte, 16)
	header[0] = 'l'
	header[1] = 1
	header[3] = 1
	binary.LittleEndian.PutUint32(header[4:], uint32(len(body)))
	binary.LittleEndian.PutUint32(header[8:], serial)
	binary.LittleEndian.PutUint32(header[12:], uint32(len(fields)))
	header = append(header, fields...)
	for len(header)%8 != 0 {
		header = append(header, 0)
	}
	return append(header, body...), nil
}

func dbusHeaderField(buf []byte, code byte, sig string, value []byte) []byte {
	base := 16 + len(buf)
	for base%8 != 0 {
		buf = append(buf, 0)
		base++
	}
	buf = append(buf, code)
	buf = append(buf, dbusSignatureBytes(sig)...)
	base = 16 + len(buf)
	align := 4
	if sig == "g" {
		align = 1
	}
	for base%align != 0 {
		buf = append(buf, 0)
		base++
	}
	return append(buf, value...)
}

func dbusSignatureBytes(sig string) []byte {
	b := make([]byte, 0, len(sig)+2)
	b = append(b, byte(len(sig)))
	b = append(b, sig...)
	b = append(b, 0)
	return b
}

func dbusStringBytes(s string) []byte {
	b := make([]byte, 4+len(s)+1)
	binary.LittleEndian.PutUint32(b, uint32(len(s)))
	copy(b[4:], s)
	return b
}

func appendDBusString(b []byte, s string) []byte {
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return append(b, dbusStringBytes(s)...)
}

func dbusBoolBytes(v bool) []byte {
	b := make([]byte, 4)
	if v {
		binary.LittleEndian.PutUint32(b, 1)
	}
	return b
}

func dbusUint32Bytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return b
}

func appendDBusArray(body []byte, elemAlign int, content []byte) []byte {
	for len(body)%4 != 0 {
		body = append(body, 0)
	}
	n := make([]byte, 4)
	binary.LittleEndian.PutUint32(n, uint32(len(content)))
	body = append(body, n...)
	for len(body)%elemAlign != 0 {
		body = append(body, 0)
	}
	return append(body, content...)
}

func dbusDictEntry(buf []byte, key, sig string, value []byte) []byte {
	for len(buf)%8 != 0 {
		buf = append(buf, 0)
	}
	buf = append(buf, dbusStringBytes(key)...)
	buf = append(buf, dbusSignatureBytes(sig)...)
	base := len(buf)
	align := 4
	if sig == "g" {
		align = 1
	}
	for base%align != 0 {
		buf = append(buf, 0)
		base++
	}
	return append(buf, value...)
}

func dbusString(b []byte, off int) (string, int, error) {
	for off%4 != 0 && off < len(b) {
		off++
	}
	if off+4 > len(b) {
		return "", 0, io.ErrUnexpectedEOF
	}
	n := int(binary.LittleEndian.Uint32(b[off:]))
	off += 4
	if n < 0 || off+n >= len(b) {
		return "", 0, io.ErrUnexpectedEOF
	}
	s := string(b[off : off+n])
	return s, off + n + 1, nil
}

func dbusSignature(b []byte, off int) (string, int, error) {
	if off >= len(b) {
		return "", 0, io.ErrUnexpectedEOF
	}
	n := int(b[off])
	off++
	if off+n >= len(b) {
		return "", 0, io.ErrUnexpectedEOF
	}
	return string(b[off : off+n]), off + n + 1, nil
}

func dbusUint32(b []byte, off int) (uint32, int, error) {
	for off%4 != 0 && off < len(b) {
		off++
	}
	if off+4 > len(b) {
		return 0, 0, io.ErrUnexpectedEOF
	}
	return binary.LittleEndian.Uint32(b[off:]), off + 4, nil
}

func dbusDictString(b []byte, off int, want string) (string, error) {
	if _, next, err := dbusUint32(b, off); err != nil {
		return "", err
	} else {
		off = next
	}
	for off%8 != 0 && off < len(b) {
		off++
	}
	for off+8 < len(b) {
		start := off
		key, next, err := dbusString(b, off)
		if err != nil {
			return "", err
		}
		sig, next, err := dbusSignature(b, next)
		if err != nil {
			return "", err
		}
		if sig == "s" || sig == "o" {
			val, n, err := dbusString(b, next)
			if err != nil {
				return "", err
			}
			if key == want {
				return val, nil
			}
			off = n
		} else {
			return "", fmt.Errorf("unsupported portal value %s", sig)
		}
		if off <= start {
			break
		}
		for off%8 != 0 && off < len(b) {
			off++
		}
	}
	return "", fmt.Errorf("portal result has no %s", want)
}
