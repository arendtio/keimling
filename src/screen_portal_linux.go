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
	"math"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	portalBusName = "org.freedesktop.portal.Desktop"
	portalPath    = "/org/freedesktop/portal/desktop"
	portalCast    = "org.freedesktop.portal.ScreenCast"
	portalRemote  = "org.freedesktop.portal.RemoteDesktop"
)

func screenShareChooser() (string, []screenTarget) {
	return "portal", nil
}

// openScreenCast opens the desktop chooser for a screen or a window and
// returns a live PipeWire picture. WebKit's own capture dies while agreeing
// on a picture format, so this process reads the stream. input asks the same
// dialog for the keyboard and mouse as well.
func openScreenCast(ctx context.Context, input bool, target string) (*screenCast, error) {
	_ = target
	bus, err := dialSessionBus()
	if err != nil {
		return nil, err
	}
	name, err := bus.hello()
	if err != nil {
		bus.Close()
		return nil, err
	}
	owner := portalCast
	if input {
		owner = portalRemote
	}
	handle := portalToken("keimling")
	sessionTok := portalToken("ksess")
	code, body, err := bus.portalRequest(ctx, name, owner, "CreateSession", "a{sv}", createSessionBody(handle, sessionTok), handle)
	if err != nil {
		bus.Close()
		if input {
			return nil, portalInputErr(err)
		}
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
	sourceTypes, restore := uint32(1|2), ""
	if input {
		sourceTypes, restore = screenShareRecall()
	}
	code, _, err = bus.portalRequest(ctx, name, portalCast, "SelectSources", "oa{sv}", selectSourcesBodyTypes(session, srcTok, sourceTypes), srcTok)
	if err != nil {
		return fail(err)
	}
	if code == 1 {
		return fail(errScreenCancelled)
	}
	if code != 0 {
		return fail(fmt.Errorf("screen share failed (%d)", code))
	}
	if input {
		devTok := portalToken("kdev")
		code, _, err = bus.portalRequest(ctx, name, portalRemote, "SelectDevices", "oa{sv}", selectDevicesBodyRestore(session, devTok, restore), devTok)
		if err != nil {
			return fail(portalInputErr(err))
		}
		if code == 1 {
			return fail(errScreenCancelled)
		}
		if code != 0 {
			return fail(fmt.Errorf("keyboard and mouse were not allowed (%d)", code))
		}
	}
	startTok := portalToken("kstart")
	startIface := portalCast
	if input {
		startIface = portalRemote
	}
	code, body, err = bus.portalRequest(ctx, name, startIface, "Start", "osa{sv}", startCastBody(session, startTok), startTok)
	if err != nil {
		return fail(err)
	}
	if code == 1 {
		return fail(errScreenCancelled)
	}
	if code != 0 {
		return fail(fmt.Errorf("screen share failed (%d)", code))
	}
	if input {
		if dev, ok := portalResultUint(body, "devices"); ok && dev&3 == 0 {
			return fail(fmt.Errorf("keyboard and mouse were not allowed"))
		}
	}
	node, source, err := portalStream(body)
	if err != nil {
		return fail(err)
	}
	screenShareRemember(source, "")
	if input {
		if tok, ok := portalResultString(body, "restore_token"); ok {
			screenShareRemember(0, tok)
		}
	}
	fd, err := bus.openPipeWire(session)
	if err != nil {
		return fail(err)
	}
	var ctrl *screenControl
	if input {
		ctrl = linuxScreenControl(bus, session, node)
	}
	cast := &screenCast{fd: fd, control: ctrl}
	cast.run = func(ctx context.Context, write func([]byte) error) error {
		return streamPipeWire(ctx, cast.takeFD(), node, write)
	}
	cast.stop = func() {
		if fd := cast.takeFD(); fd >= 0 {
			_ = syscall.Close(fd)
		}
		bus.wake()
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
	return selectSourcesBodyTypes(session, token, 1|2)
}

func selectSourcesBodyTypes(session, token string, types uint32) []byte {
	if types != 1 && types != 2 {
		types = 1 | 2
	}
	body := appendDBusString(nil, session)
	var dict []byte
	dict = dbusDictEntry(dict, "types", "u", dbusUint32Bytes(types))
	dict = dbusDictEntry(dict, "multiple", "b", dbusBoolBytes(false))
	dict = dbusDictEntry(dict, "cursor_mode", "u", dbusUint32Bytes(2))
	dict = dbusDictEntry(dict, "handle_token", "s", dbusStringBytes(token))
	return appendDBusArray(body, 8, dict)
}

func selectDevicesBody(session, token string) []byte {
	return selectDevicesBodyRestore(session, token, "")
}

func selectDevicesBodyRestore(session, token, restore string) []byte {
	body := appendDBusString(nil, session)
	var dict []byte
	dict = dbusDictEntry(dict, "types", "u", dbusUint32Bytes(1|2))
	dict = dbusDictEntry(dict, "persist_mode", "u", dbusUint32Bytes(1))
	if strings.TrimSpace(restore) != "" {
		dict = dbusDictEntry(dict, "restore_token", "s", dbusStringBytes(strings.TrimSpace(restore)))
	}
	dict = dbusDictEntry(dict, "handle_token", "s", dbusStringBytes(token))
	return appendDBusArray(body, 8, dict)
}

var screenShareMemory struct {
	mu      sync.Mutex
	source  uint32
	restore string
}

func screenShareRemember(source uint32, restore string) {
	screenShareMemory.mu.Lock()
	defer screenShareMemory.mu.Unlock()
	if source == 1 || source == 2 {
		screenShareMemory.source = source
	}
	if strings.TrimSpace(restore) != "" {
		screenShareMemory.restore = strings.TrimSpace(restore)
	}
}

func screenShareRecall() (uint32, string) {
	screenShareMemory.mu.Lock()
	defer screenShareMemory.mu.Unlock()
	types := screenShareMemory.source
	if types != 1 && types != 2 {
		types = 1 | 2
	}
	return types, screenShareMemory.restore
}

func startCastBody(session, token string) []byte {
	body := appendDBusString(nil, session)
	body = appendDBusString(body, "")
	var dict []byte
	dict = dbusDictEntry(dict, "handle_token", "s", dbusStringBytes(token))
	return appendDBusArray(body, 8, dict)
}

func portalStreamNode(body []byte) (uint32, error) {
	node, _, err := portalStream(body)
	return node, err
}

func portalStream(body []byte) (uint32, uint32, error) {
	_, off, err := dbusUint32(body, 0)
	if err != nil {
		return 0, 0, err
	}
	n, off, err := dbusUint32(body, off)
	if err != nil {
		return 0, 0, err
	}
	for off%8 != 0 && off < len(body) {
		off++
	}
	end := off + int(n)
	if end > len(body) {
		end = len(body)
	}
	for off+8 < end {
		key, next, err := dbusString(body, off)
		if err != nil {
			break
		}
		sig, next, err := dbusSignature(body, next)
		if err != nil {
			break
		}
		if key == "streams" && strings.HasPrefix(sig, "a(u") {
			node, source, err := streamNodeAt(body, next)
			return node, source, err
		}
		next, err = skipDBusValue(body, next, sig)
		if err != nil {
			return 0, 0, fmt.Errorf("screen share returned no stream")
		}
		off = next
		for off%8 != 0 && off < end {
			off++
		}
	}
	return 0, 0, fmt.Errorf("screen share returned no stream")
}

func streamNodeAt(body []byte, off int) (uint32, uint32, error) {
	for off%4 != 0 && off < len(body) {
		off++
	}
	_, off, err := dbusUint32(body, off)
	if err != nil {
		return 0, 0, fmt.Errorf("screen share returned no stream")
	}
	for off%8 != 0 && off < len(body) {
		off++
	}
	node, off, err := dbusUint32(body, off)
	if err != nil || node == 0 {
		return 0, 0, fmt.Errorf("screen share returned no stream")
	}
	return node, streamSourceType(body, off), nil
}

func streamSourceType(body []byte, off int) uint32 {
	n, off, err := dbusUint32(body, off)
	if err != nil || n == 0 {
		return 0
	}
	for off%8 != 0 && off < len(body) {
		off++
	}
	end := off + int(n)
	if end > len(body) {
		end = len(body)
	}
	for off+8 < end {
		key, next, err := dbusString(body, off)
		if err != nil {
			return 0
		}
		sig, next, err := dbusSignature(body, next)
		if err != nil {
			return 0
		}
		if key == "source_type" && sig == "u" {
			v, _, err := dbusUint32(body, next)
			if err != nil {
				return 0
			}
			return v
		}
		next, err = skipDBusValue(body, next, sig)
		if err != nil {
			return 0
		}
		off = next
		for off%8 != 0 && off < end {
			off++
		}
	}
	return 0
}

func skipDBusValue(b []byte, off int, sig string) (int, error) {
	if sig == "" || off > len(b) {
		return 0, io.ErrUnexpectedEOF
	}
	switch sig[0] {
	case 'y':
		if off >= len(b) {
			return 0, io.ErrUnexpectedEOF
		}
		return off + 1, nil
	case 'b', 'i', 'u', 'h':
		_, off, err := dbusUint32(b, off)
		return off, err
	case 'n', 'q':
		for off%2 != 0 && off < len(b) {
			off++
		}
		if off+2 > len(b) {
			return 0, io.ErrUnexpectedEOF
		}
		return off + 2, nil
	case 's', 'o':
		_, off, err := dbusString(b, off)
		return off, err
	case 'g':
		if off >= len(b) {
			return 0, io.ErrUnexpectedEOF
		}
		n := int(b[off])
		off += 1 + n + 1
		if off > len(b) {
			return 0, io.ErrUnexpectedEOF
		}
		return off, nil
	case 'd', 'x', 't':
		for off%8 != 0 && off < len(b) {
			off++
		}
		if off+8 > len(b) {
			return 0, io.ErrUnexpectedEOF
		}
		return off + 8, nil
	case 'a':
		return dbusSkipArray(b, off, sig[1:])
	default:
		return 0, fmt.Errorf("unsupported portal value %s", sig)
	}
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
	mu      sync.Mutex
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
	d.mu.Lock()
	defer d.mu.Unlock()
	closeFDs(d.fds)
	d.fds = nil
	for _, msg := range d.signals {
		closeFDs(msg.fds)
	}
	d.signals = nil
	if d.c == nil {
		return nil
	}
	err := d.c.Close()
	d.c = nil
	return err
}

func (d *dbusClient) wake() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.c != nil {
		_ = d.c.SetReadDeadline(time.Now().Add(2 * time.Second))
	}
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
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.c == nil {
		return "", nil, nil, fmt.Errorf("screen share stopped")
	}
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

func portalInputErr(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, "UnknownMethod") || strings.Contains(msg, "UnknownInterface") {
		return fmt.Errorf("this desktop cannot share keyboard and mouse")
	}
	return err
}

func appendDBusUint32(b []byte, v uint32) []byte {
	for len(b)%4 != 0 {
		b = append(b, 0)
	}
	return append(b, dbusUint32Bytes(v)...)
}

func appendDBusInt32(b []byte, v int32) []byte {
	return appendDBusUint32(b, uint32(v))
}

func appendDBusDouble(b []byte, v float64) []byte {
	for len(b)%8 != 0 {
		b = append(b, 0)
	}
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(v))
	return append(b, buf[:]...)
}

func pointerMoveBody(session string, stream uint32, x, y float64) []byte {
	body := appendDBusString(nil, session)
	body = appendDBusArray(body, 8, nil)
	body = appendDBusUint32(body, stream)
	body = appendDBusDouble(body, x)
	body = appendDBusDouble(body, y)
	return body
}

func pointerButtonBody(session string, button int32, down bool) []byte {
	body := appendDBusString(nil, session)
	body = appendDBusArray(body, 8, nil)
	body = appendDBusInt32(body, button)
	state := uint32(0)
	if down {
		state = 1
	}
	return appendDBusUint32(body, state)
}

func keysymBody(session string, sym int32, down bool) []byte {
	return pointerButtonBody(session, sym, down)
}

func (d *dbusClient) notify(member, sig string, body []byte) error {
	_, _, fds, err := d.call(portalBusName, portalPath, portalRemote, member, sig, body)
	closeFDs(fds)
	return err
}

func (d *dbusClient) pointerMove(session string, stream uint32, x, y int) error {
	return d.notify("NotifyPointerMotionAbsolute", "oa{sv}udd", pointerMoveBody(session, stream, float64(x), float64(y)))
}

func (d *dbusClient) pointerButton(session string, button int32, down bool) error {
	return d.notify("NotifyPointerButton", "oa{sv}iu", pointerButtonBody(session, button, down))
}

func (d *dbusClient) pointerClick(session string, stream uint32, x, y int, button string) error {
	code := int32(272)
	switch button {
	case "right":
		code = 273
	case "middle":
		code = 274
	}
	if err := d.pointerMove(session, stream, x, y); err != nil {
		return err
	}
	if err := d.pointerButton(session, code, true); err != nil {
		return err
	}
	return d.pointerButton(session, code, false)
}

func (d *dbusClient) pointerScroll(session string, dy int) error {
	body := appendDBusString(nil, session)
	body = appendDBusArray(body, 8, nil)
	body = appendDBusUint32(body, 0)
	body = appendDBusInt32(body, int32(dy))
	err := d.notify("NotifyPointerAxisDiscrete", "oa{sv}ui", body)
	if err == nil || !strings.Contains(err.Error(), "UnknownMethod") {
		return err
	}
	body = appendDBusString(nil, session)
	body = appendDBusArray(body, 8, nil)
	body = appendDBusDouble(body, 0)
	body = appendDBusDouble(body, float64(dy))
	return d.notify("NotifyPointerAxis", "oa{sv}dd", body)
}

func (d *dbusClient) keyTap(session string, sym int32) error {
	if err := d.notify("NotifyKeyboardKeysym", "oa{sv}iu", keysymBody(session, sym, true)); err != nil {
		return err
	}
	return d.notify("NotifyKeyboardKeysym", "oa{sv}iu", keysymBody(session, sym, false))
}

func linuxScreenControl(bus *dbusClient, session string, node uint32) *screenControl {
	ctrl := &screenControl{}
	ready := func() error {
		w, h := ctrl.size()
		if w < 1 || h < 1 {
			return fmt.Errorf("the picture is not ready")
		}
		return nil
	}
	ctrl.click = func(x, y float64, button string) error {
		if err := ready(); err != nil {
			return err
		}
		w, h := ctrl.size()
		return bus.pointerClick(session, node, screenPixel(x, w), screenPixel(y, h), button)
	}
	ctrl.scroll = func(x, y float64, dy int) error {
		if err := ready(); err != nil {
			return err
		}
		w, h := ctrl.size()
		if err := bus.pointerMove(session, node, screenPixel(x, w), screenPixel(y, h)); err != nil {
			return err
		}
		return bus.pointerScroll(session, dy)
	}
	ctrl.key = func(name string) error {
		sym, ok := linuxKeysym(name)
		if !ok {
			return fmt.Errorf("unknown key")
		}
		return bus.keyTap(session, sym)
	}
	ctrl.text = func(s string) error {
		for _, r := range s {
			sym, ok := keysymForRune(r)
			if !ok {
				continue
			}
			if err := bus.keyTap(session, sym); err != nil {
				return err
			}
		}
		return nil
	}
	return ctrl
}

func portalResultString(body []byte, want string) (string, bool) {
	_, off, err := dbusUint32(body, 0)
	if err != nil {
		return "", false
	}
	n, off, err := dbusUint32(body, off)
	if err != nil {
		return "", false
	}
	for off%8 != 0 && off < len(body) {
		off++
	}
	end := off + int(n)
	if end > len(body) {
		end = len(body)
	}
	for off+8 < end {
		key, next, err := dbusString(body, off)
		if err != nil {
			return "", false
		}
		sig, next, err := dbusSignature(body, next)
		if err != nil {
			return "", false
		}
		if key == want && sig == "s" {
			v, _, err := dbusString(body, next)
			if err != nil || strings.TrimSpace(v) == "" {
				return "", false
			}
			return v, true
		}
		next, err = skipDBusValue(body, next, sig)
		if err != nil {
			return "", false
		}
		off = next
		for off%8 != 0 && off < end {
			off++
		}
	}
	return "", false
}

func portalResultUint(body []byte, want string) (uint32, bool) {
	_, off, err := dbusUint32(body, 0)
	if err != nil {
		return 0, false
	}
	if _, off, err = dbusUint32(body, off); err != nil {
		return 0, false
	}
	for off%8 != 0 && off < len(body) {
		off++
	}
	for off+8 < len(body) {
		key, next, err := dbusString(body, off)
		if err != nil {
			return 0, false
		}
		sig, next, err := dbusSignature(body, next)
		if err != nil {
			return 0, false
		}
		switch {
		case sig == "u" || sig == "b":
			v, n, err := dbusUint32(body, next)
			if err != nil {
				return 0, false
			}
			if key == want && sig == "u" {
				return v, true
			}
			off = n
		case sig == "s" || sig == "o":
			_, n, err := dbusString(body, next)
			if err != nil {
				return 0, false
			}
			off = n
		case strings.HasPrefix(sig, "a") && len(sig) > 1:
			n, err := dbusSkipArray(body, next, sig[1:])
			if err != nil {
				return 0, false
			}
			off = n
		default:
			return 0, false
		}
		for off%8 != 0 && off < len(body) {
			off++
		}
	}
	return 0, false
}

func dbusSkipArray(b []byte, off int, elemSig string) (int, error) {
	n, off, err := dbusUint32(b, off)
	if err != nil {
		return 0, err
	}
	align := 4
	if elemSig != "" {
		switch elemSig[0] {
		case 'y', 'g':
			align = 1
		case 'n', 'q':
			align = 2
		case '(', '{', 'd', 'x', 't':
			align = 8
		}
	}
	for off%align != 0 && off < len(b) {
		off++
	}
	off += int(n)
	if off > len(b) {
		return 0, io.ErrUnexpectedEOF
	}
	return off, nil
}
