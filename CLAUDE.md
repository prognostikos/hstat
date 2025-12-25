# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build
go build -o hstat .
make build              # Uses devcontainer if go not installed locally

# Test
go test ./...           # All tests
go test -v ./parser     # Single package with verbose
go test -race ./...     # With race detector
go test -bench=. ./store # Benchmarks

# Lint (CI checks these)
go vet ./...
gofmt -w .              # Format code (required for CI)

# Install
make install            # Installs to ~/.local/bin
make install PREFIX=/usr/local
```

## Architecture

hstat is a TUI for monitoring Heroku router logs in real-time, built with Bubble Tea (Elm architecture).

### Data Flow

1. **main.go** - Reads log lines from stdin in a goroutine, parses them, sends `EntryMsg` to the Bubble Tea program. Opens `/dev/tty` separately for keyboard input since stdin is the log pipe.

2. **parser/** - Extracts fields from Heroku router log lines: status, service time, connect time, host, path, IP. Returns nil for non-router lines.

3. **store/** - Time-windowed data storage with pre-computed aggregates. Tracks counts per host/IP/status/path and maintains timing arrays for percentile calculations. Prunes old entries based on configured window. Excludes HTTP 101 (WebSocket) from timing stats.

4. **ui/** - Bubble Tea model with:
   - `model.go` - State struct, message types, `refreshData()` pulls from store
   - `update.go` - Key handlers, whois/ipinfo commands
   - `view.go` - Renders header, stats, status codes, hosts/IPs lists, paths (when filtered)
   - `styles.go` - Lipgloss styles

### Key Patterns

- **Filtering**: Setting `filter.Host` or `filter.IP` changes what `GetTopHosts/GetTopIPs/GetStatusCounts` return. When filtering by host, paths for that host are also shown.
- **Pruning**: Store maintains parallel arrays (entries, serviceTimes, connectTimes). The 101 status is tracked in entries but excluded from timing arrays.
- **Modal overlay**: Whois/ipinfo results display in a centered modal over dimmed background.
- **Stream monitoring**: Tracks `lastEntryTime` to warn when no data arrives for 30s, and `streamEnded` when stdin closes.
