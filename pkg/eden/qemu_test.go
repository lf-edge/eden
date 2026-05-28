package eden

import (
	"net"
	"strconv"
	"strings"
	"testing"
)

func TestCheckPortFree_Free(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("setup listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("setup close: %v", err)
	}
	if err := checkPortFree("127.0.0.1", port, "test role"); err != nil {
		t.Fatalf("checkPortFree on free port: %v", err)
	}
}

func TestCheckPortFree_InUseAndReleased(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("setup listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	// Listener still open: checkPortFree should report it in use with
	// the role, address, and actionable hint in the message.
	err = checkPortFree("127.0.0.1", port, "EVE console (telnet)")
	if err == nil {
		ln.Close()
		t.Fatalf("checkPortFree on bound port: want error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "EVE console (telnet)") {
		t.Errorf("error missing role: %q", msg)
	}
	if !strings.Contains(msg, "127.0.0.1:"+strconv.Itoa(port)) {
		t.Errorf("error missing address: %q", msg)
	}
	if !strings.Contains(msg, "sudo ss -tlnp") {
		t.Errorf("error missing actionable hint: %q", msg)
	}

	// Close the listener — the port is now free.
	if err := ln.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	// checkPortFree must now succeed: the conflict has cleared.
	if err := checkPortFree("127.0.0.1", port, "EVE console (telnet)"); err != nil {
		t.Fatalf("checkPortFree after release: %v", err)
	}

	// And checkPortFree's own probe listener must not have lingered:
	// a fresh net.Listen on the same port has to succeed immediately.
	// (If checkPortFree leaked its socket, this would fail with EADDRINUSE.)
	ln2, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		t.Fatalf("net.Listen after successful checkPortFree: %v", err)
	}
	ln2.Close()
}
