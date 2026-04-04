# ipk-L4-scan

`ipk-L4-scan` is a Go command-line tool for scanning TCP and UDP ports on a target host using a selected network interface. It resolves the host, chooses a suitable local address from the requested interface, and reports each port as `open`, `closed`, or `filtered` depending on the probe outcome.

## Build And Run

Build the binary with the Go toolchain:

```bash
go build -o ipk-L4-scan .
```

or

```bash
go build .
```

Run the scanner directly after building:

```bash
./ipk-L4-scan -i INTERFACE -t PORTS [-u PORTS] HOST [-w TIMEOUT]
```

Examples:

```bash
./ipk-L4-scan -i eth0 -t 22,80,443 localhost
./ipk-L4-scan -i wlan0 -u 53 192.0.2.10
./ipk-L4-scan -i lo
```

The last form lists active interfaces only. Help is available with `-h` or `--help`.

The repository also includes a Makefile for POSIX-compatible environments:

```bash
make
make test
make clean
```

Packing the project for submission only works on POSIX shells due to the use of `zip`, `find` and shell syntax. If your environment supports it, you can create a submission archive with:

```bash
make pack
```

## Implemented Features And Behavior

The scanner supports these command-line options:

- `-i INTERFACE` selects the network interface used for outgoing probes.
- `-t PORTS` scans TCP ports, accepting single values, comma-separated lists, and ranges such as `22,80,443` or `1-1024`.
- `-u PORTS` scans UDP ports with the same port syntax.
- `-w TIMEOUT` sets the probe timeout in milliseconds.
- `-h` and `--help` print usage information.

If `-i` is provided without a value and no other scan arguments are present, the program switches to interface-list mode and prints active interfaces.

The implementation resolves the target host to all available IP addresses and scans each address against the requested TCP and UDP port sets. TCP scanning uses a dial-based probe and classifies results as:

- `open` when the connection succeeds.
- `closed` when the connection is refused.
- `filtered` when the connection times out twice or returns an unexpected failure.

UDP scanning uses a best-effort probe and classifies results as:

- `closed` when the platform reports a refusal.
- `open` when the probe succeeds or when the response is ambiguous.

Port parsing removes duplicates and sorts the final port list before scanning.

## Design Decisions

The code separates command-line parsing, probing, and orchestration into small functions so each part can be tested independently.

The scanner uses a fixed worker pool to keep concurrency bounded while still allowing many probes to run in parallel.

Local source address selection is interface-aware: the program prefers a loopback address for loopback targets and otherwise chooses the first address that matches the target IP family.

UDP probing is intentionally conservative. Unlike TCP, UDP does not provide a reliable connection handshake, so the code treats many ambiguous outcomes as `open` instead of making a false claim that a service is closed.

## Testing

Reproduce the verification used for this repository with these commands:

```bash
go test .
go build -o ipk-L4-scan .
```

Observed result in this workspace:

- `go test .` passed with 37 tests.
- `go build -o ipk-L4-scan .` completed successfully.

The test coverage includes port parsing, argument validation, scan-state helpers, and selected TCP/UDP behavior using local loopback sockets.

## Known Limitations

- UDP results are heuristic by nature and may report `open` for ambiguous network failures.
- Interface listing only shows interfaces that are currently up.
- Scan output depends on local network state, host resolution, and mainly firewall behavior.
- The Makefile assumes a POSIX shell environment; on Windows, `go build` and `go test` are the most portable commands.

## References And Sources

- Project source code in this repository.
- Go standard library documentation for `context`, `net`, `os`, `os/signal`, `sync`, `syscall`, and `time`.