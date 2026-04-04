package main

import (
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

func parseArgs(args []string) (config, error) {
	cfg := config{timeout: defaultTimeout}
	ifaceWithoutValue := false
	debugf("parseArgs: raw args=%v", args)

	for i := 0; i < len(args); i++ { // Iterate over command-line arguments and update config struct
		arg := args[i]
		switch arg {
		case "-h", "--help":
			cfg.help = true
		case "-i":
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				cfg.iface = args[i+1]
				i++
			} else {
				ifaceWithoutValue = true
			}
		case "-t":
			if i+1 >= len(args) {
				return cfg, errors.New("missing value for -t")
			}
			ports, err := parsePorts(args[i+1]) // Parse TCP ports, returns slice of ints and error if any
			if err != nil {
				return cfg, fmt.Errorf("invalid TCP ports: %w", err)
			}
			cfg.tcpPorts = ports
			i++
		case "-u":
			if i+1 >= len(args) {
				return cfg, errors.New("missing value for -u")
			}
			ports, err := parsePorts(args[i+1]) // Parse UDP ports, returns slice of ints and error if any
			if err != nil {
				return cfg, fmt.Errorf("invalid UDP ports: %w", err)
			}
			cfg.udpPorts = ports
			i++
		case "-w":
			if i+1 >= len(args) {
				return cfg, errors.New("missing value for -w")
			}
			ms, err := strconv.Atoi(args[i+1]) // Parse timeout value in milliseconds, returns int and error if any
			if err != nil || ms <= 0 {
				return cfg, errors.New("-w must be a positive integer in milliseconds")
			}
			cfg.timeout = time.Duration(ms) * time.Millisecond
			i++
		default:
			if strings.HasPrefix(arg, "-") {
				return cfg, fmt.Errorf("unknown argument: %s", arg)
			}
			if cfg.host != "" {
				return cfg, errors.New("multiple host arguments provided")
			}
			cfg.host = arg
		}
	}

	if cfg.help { // If help flag is set, print usage and exit
		debugf("parseArgs: help requested")
		return cfg, nil
	}

	// If -i is provided without a value and no other required fields are set, assume interface listing mode
	if ifaceWithoutValue && cfg.iface == "" && cfg.host == "" && len(cfg.tcpPorts) == 0 && len(cfg.udpPorts) == 0 && cfg.timeout == defaultTimeout {
		cfg.listIfacesOnly = true
		debugf("parseArgs: interface list mode selected")
		return cfg, nil
	}

	if cfg.iface == "" {
		return cfg, errors.New("-i INTERFACE is required")
	}

	if cfg.host == "" {
		return cfg, errors.New("HOST is required")
	}

	if len(cfg.tcpPorts) == 0 && len(cfg.udpPorts) == 0 {
		return cfg, errors.New("at least one of -t or -u must be provided")
	}

	debugf("parseArgs: iface=%s host=%s tcpPorts=%d udpPorts=%d timeout=%s", cfg.iface, cfg.host, len(cfg.tcpPorts), len(cfg.udpPorts), cfg.timeout)

	return cfg, nil
}

func parsePorts(raw string) ([]int, error) {
	if strings.TrimSpace(raw) == "" { // Check for empty port list
		return nil, errors.New("empty port list")
	}

	parts := strings.Split(raw, ",") // Split raw port string by commas to handle multiple ports and ranges
	seen := map[int]struct{}{}       // Use a map to track seen ports and avoid duplicates

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			return nil, errors.New("empty port element")
		}

		if strings.Contains(p, "-") { // Check for port range specified with a hyphen, e.g. "20-25"
			bounds := strings.Split(p, "-")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid range: %s", p)
			}

			start, err := strconv.Atoi(strings.TrimSpace(bounds[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid range start: %s", p)
			}
			end, err := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid range end: %s", p)
			}

			if start < 1 || end > 65535 || start > end {
				return nil, fmt.Errorf("range out of bounds: %s", p)
			}

			for port := start; port <= end; port++ {
				seen[port] = struct{}{}
			}
			continue
		}

		port, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid port: %s", p)
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("port out of bounds: %d", port)
		}
		seen[port] = struct{}{}
	}

	// Combine seen ports into a slice and sort them for consistent output, however due to the async nature of scanning, the order of output may not be guaranteed
	ports := make([]int, 0, len(seen))
	for port := range seen {
		ports = append(ports, port)
	}
	sort.Ints(ports)
	return ports, nil
}

func printUsage() { // Print usage information for the command-line tool
	fmt.Println("Usage:")
	fmt.Println("  ./ipk-L4-scan -i INTERFACE [-u PORTS] [-t PORTS] HOST [-w TIMEOUT] [-h | --help]")
	fmt.Println("  ./ipk-L4-scan -i")
	fmt.Println("  ./ipk-L4-scan -h")
	fmt.Println("  ./ipk-L4-scan --help")
}

func printActiveInterfaces() error { // List all active network interfaces on the system, returns error if any occurs
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		fmt.Println(iface.Name)
	}
	return nil
}
