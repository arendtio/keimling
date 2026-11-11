//go:build linux

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"math"
	"strings"
	"testing"
)

func TestScreenCastBodiesMatchPortal(t *testing.T) {
	session := "/org/freedesktop/portal/desktop/session/1_2/sessionTOK"
	cases := []struct {
		name string
		got  []byte
		want string
	}{
		{"create", createSessionBody("handleTOK", "sessionTOK"), "53000000000000000c00000068616e646c655f746f6b656e000173000900000068616e646c65544f4b000000000000001400000073657373696f6e5f68616e646c655f746f6b656e000173000a00000073657373696f6e544f4b00"},
		{"select", selectSourcesBody(session, "srcTOK"), "360000002f6f72672f667265656465736b746f702f706f7274616c2f6465736b746f702f73657373696f6e2f315f322f73657373696f6e544f4b000067000000050000007479706573000175000000000300000000000000080000006d756c7469706c650001620000000000000000000b000000637572736f725f6d6f64650001750000020000000c00000068616e646c655f746f6b656e0001730006000000737263544f4b00"},
		{"start", startCastBody(session, "startTOK"), "360000002f6f72672f667265656465736b746f702f706f7274616c2f6465736b746f702f73657373696f6e2f315f322f73657373696f6e544f4b00000000000000000000210000000c00000068616e646c655f746f6b656e00017300080000007374617274544f4b00"},
	}
	for _, tc := range cases {
		got := hex.EncodeToString(tc.got)
		if got != tc.want {
			t.Errorf("%s\n got %s\nwant %s", tc.name, got, tc.want)
		}
	}
}

func TestLinuxChooserIsThePortal(t *testing.T) {
	kind, targets := screenShareChooser()
	if kind != "portal" || len(targets) != 0 {
		t.Fatalf("%s %v", kind, targets)
	}
}

func TestSelectDevicesAndPointerBody(t *testing.T) {
	session := "/org/freedesktop/portal/desktop/session/1_2/sessionTOK"
	body := selectDevicesBody(session, "devTOK")
	got, _, err := dbusString(body, 0)
	if err != nil || got != session {
		t.Fatalf("session %q %v", got, err)
	}
	if !strings.Contains(string(body), "devTOK") {
		t.Fatal("missing token")
	}
	move := pointerMoveBody(session, 7, 3, 4)
	path, off, err := dbusString(move, 0)
	if err != nil || path != session {
		t.Fatalf("move session %q %v", path, err)
	}
	_, off, err = dbusUint32(move, off)
	if err != nil {
		t.Fatal(err)
	}
	for off%8 != 0 {
		off++
	}
	node, off, err := dbusUint32(move, off)
	if err != nil || node != 7 {
		t.Fatalf("node %d %v", node, err)
	}
	for off%8 != 0 {
		off++
	}
	x := math.Float64frombits(binary.LittleEndian.Uint64(move[off : off+8]))
	if x != 3 {
		t.Fatalf("x %v", x)
	}
}

func TestPortalDevicesBit(t *testing.T) {
	// response code 0, dict with devices = 3 (keyboard|pointer)
	var dict []byte
	dict = dbusDictEntry(dict, "devices", "u", dbusUint32Bytes(3))
	body := appendDBusUint32(nil, 0)
	body = appendDBusArray(body, 8, dict)
	got, ok := portalResultUint(body, "devices")
	if !ok || got != 3 {
		t.Fatalf("devices %d %v", got, ok)
	}
}

func TestPortalStreamNode(t *testing.T) {
	raw, err := hex.DecodeString("00000000280000000700000073747265616d730009612875617b73767d29000008000000000000002a00000000000000")
	if err != nil {
		t.Fatal(err)
	}
	node, err := portalStreamNode(raw)
	if err != nil || node != 42 {
		t.Fatalf("node %d %v", node, err)
	}
}

func TestPortalStreamNodeAfterDevices(t *testing.T) {
	var entries []byte
	entries = dbusDictEntry(entries, "devices", "u", dbusUint32Bytes(1|2))
	for len(entries)%8 != 0 {
		entries = append(entries, 0)
	}
	entries = append(entries, streamsDictEntry(42)...)
	body := appendDBusUint32(nil, 0)
	body = appendDBusArray(body, 8, entries)
	node, err := portalStreamNode(body)
	if err != nil || node != 42 {
		t.Fatalf("node %d %v", node, err)
	}
}

// streamsDictEntry builds one a{sv} entry whose value is a single PipeWire node.
// The buffer it is appended to must already be 8-aligned, as dict contents are.
func TestWindowShareStaysAWindow(t *testing.T) {
	props := dbusDictEntry(nil, "source_type", "u", dbusUint32Bytes(2))
	body := appendDBusUint32(nil, 0)
	body = appendDBusArray(body, 8, streamsDictEntryProps(42, props))
	node, source, err := portalStream(body)
	if err != nil || node != 42 || source != 2 {
		t.Fatalf("node %d source %d %v", node, source, err)
	}
	screenShareMemory.mu.Lock()
	screenShareMemory.source = 0
	screenShareMemory.restore = ""
	screenShareMemory.mu.Unlock()
	screenShareRemember(source, "")
	types, restore := screenShareRecall()
	if types != 2 || restore != "" {
		t.Fatalf("recall %d %q", types, restore)
	}
	session := "/org/freedesktop/portal/desktop/session/1_2/sessionTOK"
	selected := selectSourcesBodyTypes(session, "srcTOK", types)
	if got := dictUintAfterPath(t, selected, "types"); got != 2 {
		t.Fatalf("types %d", got)
	}
	var reply []byte
	reply = dbusDictEntry(reply, "devices", "u", dbusUint32Bytes(3))
	for len(reply)%8 != 0 {
		reply = append(reply, 0)
	}
	reply = dbusDictEntry(reply, "restore_token", "s", dbusStringBytes("winTOK"))
	started := appendDBusUint32(nil, 0)
	started = appendDBusArray(started, 8, reply)
	tok, ok := portalResultString(started, "restore_token")
	if !ok || tok != "winTOK" {
		t.Fatalf("token %q %v", tok, ok)
	}
	screenShareRemember(0, tok)
	_, restore = screenShareRecall()
	devices := selectDevicesBodyRestore(session, "devTOK", restore)
	if !strings.Contains(string(devices), "winTOK") || !strings.Contains(string(devices), "persist_mode") {
		t.Fatal("restore was not offered")
	}
	screenShareMemory.mu.Lock()
	screenShareMemory.source = 0
	screenShareMemory.restore = ""
	screenShareMemory.mu.Unlock()
}

func dictUintAfterPath(t *testing.T, body []byte, want string) uint32 {
	t.Helper()
	_, off, err := dbusString(body, 0)
	if err != nil {
		t.Fatal(err)
	}
	n, off, err := dbusUint32(body, off)
	if err != nil {
		t.Fatal(err)
	}
	for off%8 != 0 {
		off++
	}
	end := off + int(n)
	for off+8 < end {
		key, next, err := dbusString(body, off)
		if err != nil {
			t.Fatal(err)
		}
		sig, next, err := dbusSignature(body, next)
		if err != nil {
			t.Fatal(err)
		}
		if key == want && sig == "u" {
			v, _, err := dbusUint32(body, next)
			if err != nil {
				t.Fatal(err)
			}
			return v
		}
		next, err = skipDBusValue(body, next, sig)
		if err != nil {
			t.Fatal(err)
		}
		off = next
		for off%8 != 0 && off < end {
			off++
		}
	}
	t.Fatalf("missing %s", want)
	return 0
}

func streamsDictEntry(node uint32) []byte {
	return streamsDictEntryProps(node, nil)
}

func streamsDictEntryProps(node uint32, props []byte) []byte {
	buf := dbusDictEntry(nil, "streams", "a(ua{sv})", nil)
	// Replace the empty value with a real array. The signature ends just before that value.
	sig := dbusSignatureBytes("a(ua{sv})")
	at := bytes.Index(buf, sig)
	if at < 0 {
		return buf
	}
	buf = buf[:at+len(sig)]
	for len(buf)%4 != 0 {
		buf = append(buf, 0)
	}
	lenAt := len(buf)
	buf = append(buf, 0, 0, 0, 0)
	for len(buf)%8 != 0 {
		buf = append(buf, 0)
	}
	dataAt := len(buf)
	buf = appendDBusUint32(buf, node)
	buf = appendDBusArray(buf, 8, props)
	binary.LittleEndian.PutUint32(buf[lenAt:lenAt+4], uint32(len(buf)-dataAt))
	return buf
}

func TestPortalResponseURI(t *testing.T) {
	raw, err := hex.DecodeString("00000000300000000300000075726900017300001f00000066696c653a2f2f2f746d702f6b65696d6c696e672d73637265656e2e706e6700")
	if err != nil {
		t.Fatal(err)
	}
	_, off, err := dbusUint32(raw, 0)
	if err != nil {
		t.Fatal(err)
	}
	uri, err := dbusDictString(raw, off, "uri")
	if err != nil || uri != "file:///tmp/keimling-screen.png" {
		t.Fatalf("uri %q %v", uri, err)
	}
}

func TestSessionBusReadsPortalVersion(t *testing.T) {
	bus, err := dialSessionBus()
	if err != nil {
		t.Skip(err)
	}
	defer bus.Close()
	if _, err := bus.hello(); err != nil {
		t.Fatal(err)
	}
	body := appendDBusString(nil, "org.freedesktop.portal.ScreenCast")
	body = appendDBusString(body, "version")
	sig, reply, fds, err := bus.call(
		"org.freedesktop.portal.Desktop",
		"/org/freedesktop/portal/desktop",
		"org.freedesktop.DBus.Properties",
		"Get",
		"ss",
		body,
	)
	closeFDs(fds)
	if err != nil {
		t.Fatal(err)
	}
	if sig != "v" {
		t.Fatalf("signature %s", sig)
	}
	vsig, off, err := dbusSignature(reply, 0)
	if err != nil || vsig != "u" {
		t.Fatalf("variant %s %v", vsig, err)
	}
	ver, _, err := dbusUint32(reply, off)
	if err != nil || ver == 0 {
		t.Fatalf("version %d %v", ver, err)
	}
}

func TestScreenPipeWireArgsLeaveFormatToTheDesktop(t *testing.T) {
	joined := strings.Join(screenPipeWireArgs(42), " ")
	if strings.Contains(joined, "capsfilter") || strings.Contains(joined, "videorate") || strings.Contains(joined, "framerate=") || strings.Contains(joined, "keepalive-time") {
		t.Fatalf("pipeline forces a format or repeats a held frame: %s", joined)
	}
	convert := strings.Index(joined, "videoconvert")
	queue := strings.Index(joined, "queue")
	encode := strings.Index(joined, "jpegenc")
	if convert < 0 || queue < convert || encode < queue {
		t.Fatalf("stale frames must be dropped after convert and before encode: %s", joined)
	}
	for _, part := range []string{"path=42", "leaky=downstream", "max-size-buffers=1"} {
		if !strings.Contains(joined, part) {
			t.Fatalf("pipeline missing %s: %s", part, joined)
		}
	}
}
