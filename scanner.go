package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func runScanner(cfg config) error {
	debugf("runScanner: starting iface=%s host=%s timeout=%s", cfg.iface, cfg.host, cfg.timeout)
	v4Local, v6Local, err := interfaceIPs(cfg.iface) // Get local IPv4 and IPv6 addresses for the specified interface
	if err != nil {
		return err // If error occurs while retrieving interface IPs, return the error
	}
	debugf("runScanner: local addresses v4=%d v6=%d", len(v4Local), len(v6Local))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ips, err := resolveHost(ctx, cfg.host)
	if err != nil {
		return err
	}
	debugf("runScanner: resolved %d IPs", len(ips))

	tasks := make(chan scanTask, 512) // Buffered channel to hold scan tasks, allowing workers to process tasks concurrently without blocking the main goroutine
	var outMu sync.Mutex              // Mutex to synchronize output to prevent interleaving of lines and ensure thread-safe writes to stdout

	workers := 200 // Number of concurrent worker goroutines to perform scanning
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-tasks:
					if !ok {
						return
					}
					state := "filtered"
					if task.proto == "tcp" {
						state = scanTCP(ctx, task.ip, task.port, cfg.timeout, task.local)
					} else {
						state = scanUDP(ctx, task.ip, task.port, cfg.timeout, task.local)
					}

					outMu.Lock()
					fmt.Printf("%s %d %s %s\n", task.ip.String(), task.port, task.proto, state)
					outMu.Unlock()
				}
			}
		}()
	}

	submit := func(ip net.IP, port int, proto string) bool {
		local := selectLocalIP(ip, v4Local, v6Local)
		select {
		case <-ctx.Done():
			return false
		case tasks <- scanTask{ip: ip, port: port, proto: proto, local: local}:
			return true
		}
	}

outer: // Iterate over each resolved IP address and each specified port, submitting scan tasks to the workers
	for _, ip := range ips {
		for _, port := range cfg.tcpPorts {
			select {
			case <-ctx.Done():
				break outer
			default:
				if !submit(ip, port, "tcp") {
					break outer
				}
			}
		}

		for _, port := range cfg.udpPorts {
			select {
			case <-ctx.Done():
				break outer
			default:
				if !submit(ip, port, "udp") {
					break outer
				}
			}
		}
	}

	close(tasks)
	wg.Wait()

	if ctx.Err() != nil { // Interrupt signal received eg. Ctrl+C
		debugf("runScanner: interrupted by signal")
		return nil
	}
	debugf("runScanner: finished")

	return nil
}

func resolveHost(ctx context.Context, host string) ([]net.IP, error) {
	debugf("resolveHost: host=%s", host)
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host) // Resolve the host to a list of IP addresses using the default resolver
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, errors.New("host resolved to zero addresses")
	}
	return ips, nil
}

// Get local IPv4 and IPv6 addresses for the specified interface
// Ensure the interface is active and has usable addresses, otherwise return an error
func interfaceIPs(ifaceName string) ([]net.IP, []net.IP, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, nil, fmt.Errorf("interface %s not found", ifaceName)
	}
	if iface.Flags&net.FlagUp == 0 {
		return nil, nil, fmt.Errorf("interface %s is not active", ifaceName)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, nil, err
	}

	v4 := make([]net.IP, 0)
	v6 := make([]net.IP, 0)
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipNet.IP
		if ip == nil {
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			v4 = append(v4, ip4)
		} else {
			v6 = append(v6, ip)
		}
	}

	if len(v4) == 0 && len(v6) == 0 {
		return nil, nil, fmt.Errorf("interface %s has no usable addresses", ifaceName)
	}

	return v4, v6, nil
}

func selectLocalIP(target net.IP, v4 []net.IP, v6 []net.IP) net.IP {
	if target.IsLoopback() { // If the target IP is a loopback address, prefer local loopback addresses for scanning
		if target.To4() != nil {
			for _, ip := range v4 {
				if ip.IsLoopback() {
					return ip
				}
			}
			return nil
		}

		for _, ip := range v6 {
			if ip.IsLoopback() {
				return ip
			}
		}
		return nil
	}

	if target.To4() != nil {
		if len(v4) > 0 {
			return v4[0]
		}
		return nil
	}

	if len(v6) > 0 {
		return v6[0]
	}
	return nil
}
