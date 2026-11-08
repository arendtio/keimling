//go:build linux

package main

import (
	"encoding/hex"
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
