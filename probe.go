package main

import (
	"context"
	"errors"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func scanTCP(ctx context.Context, ip net.IP, port int, timeout time.Duration, localIP net.IP) string {
	debugf("scanTCP: target=%s:%d", ip.String(), port)
	first := doTCPDial(ctx, ip, port, timeout, localIP)
	if first == nil {
		debugf("scanTCP: %s:%d open", ip.String(), port)
		return "open"
	}

	if isConnRefused(first) {
		debugf("scanTCP: %s:%d closed (refused)", ip.String(), port)
		return "closed"
	}

	if isTimeoutErr(first) {
		debugf("scanTCP: %s:%d timeout, retrying", ip.String(), port)
		second := doTCPDial(ctx, ip, port, timeout, localIP)
		if second == nil {
			debugf("scanTCP: %s:%d open on retry", ip.String(), port)
			return "open"
		}
		if isConnRefused(second) {
			debugf("scanTCP: %s:%d closed on retry", ip.String(), port)
			return "closed"
		}
		if isTimeoutErr(second) {
			debugf("scanTCP: %s:%d filtered (timeout twice)", ip.String(), port)
			return "filtered"
		}
	}

	debugf("scanTCP: %s:%d filtered (default)", ip.String(), port)
	return "filtered"
}

func doTCPDial(ctx context.Context, ip net.IP, port int, timeout time.Duration, localIP net.IP) error {
	dialer := net.Dialer{Timeout: timeout}
	if localIP != nil {
		dialer.LocalAddr = &net.TCPAddr{IP: localIP}
	}

	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(ip.String(), strconv.Itoa(port)))
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}

func scanUDP(ctx context.Context, ip net.IP, port int, timeout time.Duration, localIP net.IP) string {
	debugf("scanUDP: target=%s:%d", ip.String(), port)
	dialer := net.Dialer{Timeout: timeout}
	if localIP != nil {
		dialer.LocalAddr = &net.UDPAddr{IP: localIP}
	}

	conn, err := dialer.DialContext(ctx, "udp", net.JoinHostPort(ip.String(), strconv.Itoa(port)))
	if err != nil {
		if isConnRefused(err) {
			debugf("scanUDP: %s:%d closed (refused on dial)", ip.String(), port)
			return "closed"
		}
		if isTimeoutErr(err) {
			debugf("scanUDP: %s:%d open (dial timeout)", ip.String(), port)
			return "open"
		}
		debugf("scanUDP: %s:%d open (dial error treated as open): %v", ip.String(), port, err)
		return "open"
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return "open"
	}

	_, err = conn.Write([]byte{0x00})
	if err != nil {
		if isConnRefused(err) {
			debugf("scanUDP: %s:%d closed (write refused)", ip.String(), port)
			return "closed"
		}
		if isTimeoutErr(err) {
			debugf("scanUDP: %s:%d open (write timeout)", ip.String(), port)
			return "open"
		}
		debugf("scanUDP: %s:%d open (write error treated as open): %v", ip.String(), port, err)
		return "open"
	}

	buf := make([]byte, 1)
	_, err = conn.Read(buf)
	if err != nil {
		if isConnRefused(err) {
			debugf("scanUDP: %s:%d closed (read refused)", ip.String(), port)
			return "closed"
		}
		if isTimeoutErr(err) {
			debugf("scanUDP: %s:%d open (read timeout)", ip.String(), port)
			return "open"
		}
		debugf("scanUDP: %s:%d open (read error treated as open): %v", ip.String(), port, err)
		return "open"
	}

	debugf("scanUDP: %s:%d open (data received)", ip.String(), port)
	return "open"
}

func isTimeoutErr(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func isConnRefused(err error) bool {
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "actively refused") ||
		strings.Contains(msg, "port unreachable")
}
