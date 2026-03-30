package main

import (
	"context"
	"net"
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
