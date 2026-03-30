package main

import (
	"net"
	"time"
)

const defaultTimeout = 1000 * time.Millisecond

type config struct {
	iface          string
	tcpPorts       []int
	udpPorts       []int
	host           string
	timeout        time.Duration
	help           bool
	listIfacesOnly bool
}

type scanTask struct {
	ip    net.IP
	port  int
	proto string
	local net.IP
}

type scanResult struct {
	ip    string
	port  int
	proto string
	state string
}
