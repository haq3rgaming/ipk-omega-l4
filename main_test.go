package main

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"
)

func TestParsePorts(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr bool
	}{
		{name: "single", input: "22", want: []int{22}},
		{name: "list", input: "22,80,443", want: []int{22, 80, 443}},
		{name: "range", input: "1-3", want: []int{1, 2, 3}},
		{name: "mixed", input: "22,25-27", want: []int{22, 25, 26, 27}},
		{name: "invalid", input: "0", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parsePorts(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("unexpected length: got=%v want=%v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("unexpected ports: got=%v want=%v", got, tc.want)
				}
			}
		})
	}
}

func TestParsePortsErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "blank", input: "   "},
		{name: "empty element", input: "22,,80"},
		{name: "bad range", input: "1-2-3"},
		{name: "invalid range start", input: "a-2"},
		{name: "invalid range end", input: "1-b"},
		{name: "reversed range", input: "5-2"},
		{name: "out of bounds port", input: "65536"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parsePorts(tc.input); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestParseArgsHelp(t *testing.T) {
	cfg, err := parseArgs([]string{"--help"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.help {
		t.Fatalf("expected help flag to be true")
	}
}

func TestParseArgsInterfaceListingMode(t *testing.T) {
	cfg, err := parseArgs([]string{"-i"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.listIfacesOnly {
		t.Fatalf("expected listIfacesOnly to be true")
	}
}

func TestParseArgsValidScan(t *testing.T) {
	cfg, err := parseArgs([]string{"-u", "53", "localhost", "-i", "lo", "-w", "1500"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.iface != "lo" {
		t.Fatalf("unexpected interface: %s", cfg.iface)
	}
	if cfg.host != "localhost" {
		t.Fatalf("unexpected host: %s", cfg.host)
	}
	if len(cfg.udpPorts) != 1 || cfg.udpPorts[0] != 53 {
		t.Fatalf("unexpected udp ports: %v", cfg.udpPorts)
	}
	if cfg.timeout != 1500*time.Millisecond {
		t.Fatalf("unexpected timeout: %s", cfg.timeout)
	}
}

func TestParseArgsErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing interface", args: []string{"localhost", "-u", "53"}},
		{name: "missing host", args: []string{"-i", "lo", "-u", "53"}},
		{name: "missing ports", args: []string{"-i", "lo", "localhost"}},
		{name: "unknown arg", args: []string{"-i", "lo", "localhost", "-x"}},
		{name: "missing tcp value", args: []string{"-i", "lo", "localhost", "-t"}},
		{name: "missing udp value", args: []string{"-i", "lo", "localhost", "-u"}},
		{name: "missing timeout value", args: []string{"-i", "lo", "localhost", "-u", "53", "-w"}},
		{name: "bad timeout", args: []string{"-i", "lo", "localhost", "-u", "53", "-w", "0"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseArgs(tc.args); err == nil {
				t.Fatalf("expected error, got nil")
			}
		})
	}
}

func TestScanTCPOpen(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	state := scanTCP(context.Background(), net.ParseIP("127.0.0.1"), port, 200*time.Millisecond, nil)
	if state != "open" {
		t.Fatalf("unexpected TCP state: %s", state)
	}
}

func TestScanUDPOpen(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen packet failed: %v", err)
	}
	defer conn.Close()

	port := conn.LocalAddr().(*net.UDPAddr).Port
	state := scanUDP(context.Background(), net.ParseIP("127.0.0.1"), port, 200*time.Millisecond, nil)
	if state != "open" {
		t.Fatalf("unexpected UDP state: %s", state)
	}
}

func TestSelectLocalIP(t *testing.T) {
	v4Loopback := net.ParseIP("127.0.0.1")
	v4Other := net.ParseIP("192.0.2.10")
	v6Loopback := net.ParseIP("::1")
	v6Other := net.ParseIP("2001:db8::1")

	tests := []struct {
		name   string
		target net.IP
		v4     []net.IP
		v6     []net.IP
		want   net.IP
	}{
		{name: "ipv4 loopback prefers loopback", target: net.ParseIP("127.0.0.1"), v4: []net.IP{v4Other, v4Loopback}, want: v4Loopback},
		{name: "ipv4 non-loopback uses first v4", target: net.ParseIP("198.51.100.5"), v4: []net.IP{v4Other}, v6: []net.IP{v6Other}, want: v4Other},
		{name: "ipv6 non-loopback uses first v6", target: net.ParseIP("2001:db8::2"), v4: []net.IP{v4Other}, v6: []net.IP{v6Other}, want: v6Other},
		{name: "ipv6 loopback prefers loopback", target: net.ParseIP("::1"), v6: []net.IP{v6Other, v6Loopback}, want: v6Loopback},
		{name: "no matching local ip", target: net.ParseIP("127.0.0.1"), v4: []net.IP{v4Other}, want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := selectLocalIP(tc.target, tc.v4, tc.v6)
			if !got.Equal(tc.want) {
				t.Fatalf("unexpected local ip: got=%v want=%v", got, tc.want)
			}
		})
	}
}

func TestIsConnRefused(t *testing.T) {
	if !isConnRefused(syscall.ECONNREFUSED) {
		t.Fatalf("expected syscall.ECONNREFUSED to be recognized")
	}
	if !isConnRefused(errors.New("connectex: No connection could be made because the target machine actively refused it")) {
		t.Fatalf("expected refused text to be recognized")
	}
	if isConnRefused(errors.New("some other error")) {
		t.Fatalf("did not expect unrelated error to be treated as refused")
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestIsTimeoutErr(t *testing.T) {
	if !isTimeoutErr(timeoutError{}) {
		t.Fatalf("expected timeoutError to be recognized")
	}
	if isTimeoutErr(errors.New("some other error")) {
		t.Fatalf("did not expect unrelated error to be treated as timeout")
	}
}
