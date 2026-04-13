package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func scanTCP(ctx context.Context, ip net.IP, port int, timeout time.Duration, localIP net.IP) string {
	debugf("scanTCP: target=%s:%d", ip.String(), port)

	if ip4 := ip.To4(); ip4 == nil {
		debugf("scanTCP: %s:%d filtered (tcp syn scan currently supports ipv4 only)", ip.String(), port)
		return "filtered"
	}

	state, err := scanTCPSYNv4(ctx, ip.To4(), port, timeout, localIP)
	if err != nil {
		debugf("scanTCP: %s:%d filtered (syn scan failed: %v)", ip.String(), port, err)
		return "filtered"
	}
	debugf("scanTCP: %s:%d %s (syn scan)", ip.String(), port, state)
	return state
}

func scanTCPSYNv4(ctx context.Context, targetIP net.IP, targetPort int, timeout time.Duration, localIP net.IP) (string, error) {
	if targetIP == nil || targetIP.To4() == nil {
		return "", errors.New("target is not a valid IPv4 address")
	}

	srcIP, err := resolveSourceIPv4(targetIP, localIP)
	if err != nil {
		return "", err
	}

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
	if err != nil {
		return "", err
	}
	defer syscall.Close(fd)

	if err := syscall.Bind(fd, &syscall.SockaddrInet4{Addr: ipTo4Array(srcIP)}); err != nil {
		return "", err
	}

	tv := syscall.NsecToTimeval(timeout.Nanoseconds())
	if err := syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv); err != nil {
		return "", err
	}

	srcPort, seq, err := randomProbeValues()
	if err != nil {
		return "", err
	}

	tcpHeader := makeTCPHeader(srcIP, targetIP, srcPort, uint16(targetPort), seq)
	if tcpHeader == nil {
		return "", errors.New("failed to build tcp syn segment")
	}
	if err := syscall.Sendto(fd, tcpHeader, 0, &syscall.SockaddrInet4{Addr: ipTo4Array(targetIP), Port: targetPort}); err != nil {
		return "", err
	}

	deadline := time.Now().Add(timeout)
	buf := make([]byte, 1500)
	for {
		if err := ctx.Err(); err != nil {
			return "filtered", nil
		}
		if time.Now().After(deadline) {
			return "filtered", nil
		}

		n, _, err := syscall.Recvfrom(fd, buf, 0)
		if err != nil {
			if errors.Is(err, syscall.EINTR) {
				continue
			}
			if errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK) {
				return "filtered", nil
			}
			return "", err
		}

		flags, ok := parseTCPFlagsIPv4(buf[:n], targetIP, srcIP, uint16(targetPort), srcPort)
		if !ok {
			continue
		}

		if flags&tcpFlagRST != 0 {
			return "closed", nil
		}
		if flags&(tcpFlagSYN|tcpFlagACK) == (tcpFlagSYN | tcpFlagACK) {
			return "open", nil
		}
	}
}

const (
	tcpFlagSYN = 0x02
	tcpFlagRST = 0x04
	tcpFlagACK = 0x10
)

func resolveSourceIPv4(targetIP net.IP, localIP net.IP) (net.IP, error) {
	if localIP != nil {
		if ip4 := localIP.To4(); ip4 != nil {
			return ip4, nil
		}
		return nil, errors.New("local IP is not IPv4")
	}

	conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: targetIP, Port: 53})
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok || addr.IP == nil || addr.IP.To4() == nil {
		return nil, errors.New("unable to determine local IPv4 address")
	}
	return addr.IP.To4(), nil
}

func randomProbeValues() (uint16, uint32, error) {
	var b [6]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return 0, 0, err
	}

	srcPort := uint16(32768 + (binary.BigEndian.Uint16(b[0:2]) % (65535 - 32768)))
	seq := binary.BigEndian.Uint32(b[2:6])
	return srcPort, seq, nil
}

func makeTCPHeader(srcIP net.IP, dstIP net.IP, srcPort uint16, dstPort uint16, seq uint32) []byte {
	ip4 := &layers.IPv4{SrcIP: srcIP.To4(), DstIP: dstIP.To4(), Protocol: layers.IPProtocolTCP}
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
		Seq:     seq,
		SYN:     true,
		Window:  64240,
	}
	_ = tcp.SetNetworkLayerForChecksum(ip4)

	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	if err := gopacket.SerializeLayers(buf, opts, tcp); err != nil {
		return nil
	}
	return buf.Bytes()
}

func parseTCPFlagsIPv4(pkt []byte, srcIP net.IP, dstIP net.IP, srcPort uint16, dstPort uint16) (byte, bool) {
	packet := gopacket.NewPacket(pkt, layers.LayerTypeIPv4, gopacket.NoCopy)
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if ipLayer == nil || tcpLayer == nil {
		return 0, false
	}

	ipv4, ok := ipLayer.(*layers.IPv4)
	if !ok {
		return 0, false
	}
	tcp, ok := tcpLayer.(*layers.TCP)
	if !ok {
		return 0, false
	}

	if !ipv4.SrcIP.Equal(srcIP) || !ipv4.DstIP.Equal(dstIP) {
		return 0, false
	}
	if uint16(tcp.SrcPort) != srcPort || uint16(tcp.DstPort) != dstPort {
		return 0, false
	}

	var flags byte
	if tcp.SYN {
		flags |= tcpFlagSYN
	}
	if tcp.RST {
		flags |= tcpFlagRST
	}
	if tcp.ACK {
		flags |= tcpFlagACK
	}
	return flags, true
}

func ipTo4Array(ip net.IP) [4]byte {
	ip4 := ip.To4()
	if ip4 == nil {
		return [4]byte{}
	}
	var out [4]byte
	copy(out[:], ip4)
	return out
}

func scanUDP(ctx context.Context, ip net.IP, port int, timeout time.Duration, localIP net.IP) string {
	debugf("scanUDP: target=%s:%d", ip.String(), port)
	dialer := net.Dialer{Timeout: timeout} // Create a net.Dialer with the specified timeout for UDP
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

	// Set a deadline for the connection to ensure we don't wait indefinitely
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return "open"
	}

	// Send empty payload to the target UDP port
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
	// Attempt to read from the connection
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

func isTimeoutErr(err error) bool { // Check if the error is a timeout error
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func isConnRefused(err error) bool { // Check if the error is a connection refused error
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "actively refused") ||
		strings.Contains(msg, "port unreachable")
}
