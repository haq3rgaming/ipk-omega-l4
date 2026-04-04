package main

import (
	"net"
	"time"
)

const defaultTimeout = 1000 * time.Millisecond

// config holds the configuration for the scanner, including interface, ports, host, timeout, and flags for help and listing interfaces
type config struct {
	iface          string
	tcpPorts       []int
	udpPorts       []int
	host           string
	timeout        time.Duration
	help           bool
	listIfacesOnly bool
}

// scanTask represents a scanning task for a specific IP, port, protocol, and local IP
type scanTask struct {
	ip    net.IP
	port  int
	proto string
	local net.IP
}

// scanResult holds the result of a scan, including the IP, port, protocol, and output state (open/closed/filtered)
type scanResult struct {
	ip    string
	port  int
	proto string
	state string
}
