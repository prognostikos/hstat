# hstat

Real-time Heroku router log monitor with interactive filtering.

## Usage

```bash
heroku logs --tail -a myapp | hstat
```

Or with options:
```bash
heroku logs --tail -a myapp | hstat --window 10m --top 20 --refresh 1s
```

Can also read from files:
```bash
hstat < router.log
```

## Options

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--window` | `-w` | `5m` | Percentile window (`5m`, `10m`, `1h`, or `all`) |
| `--top` | `-n` | `15` | Number of hosts/IPs to show |
| `--refresh` | `-r` | `1s` | Screen refresh interval |
| `--version` | `-v` | - | Show version and exit |
| `--help` | `-h` | - | Usage info |

## Key Bindings

### Navigation
| Key | Action |
|-----|--------|
| `Tab` / `l` | Next section |
| `Shift+Tab` / `h` | Previous section |
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `g` | Jump to top |
| `G` | Jump to bottom |

### Actions
| Key | Action |
|-----|--------|
| `Enter` | Filter by selected host/IP |
| `w` | Whois lookup (when IP selected) |
| `i` | IP info lookup via ipinfo.io (when IP selected) |
| `Esc` | Clear filter (or quit if no filter) |
| `q` / `Ctrl+C` | Quit |
| `?` | Toggle help |

## Features

- Real-time response time percentiles (p50, p95, p99)
- Connect time stats
- HTTP status code breakdown with color coding
- Top hosts by request count
- Top IPs by request count
- Interactive filtering: select a host to see its IPs/statuses, or an IP to see its hosts/statuses
- IP lookup via `whois` command or ipinfo.io API (modal overlay)
- Adaptive layout (single column < 100 cols, two columns >= 100 cols)
- Time-windowed data (configurable, default 5 minutes)

## Installation

Install to `~/.local/bin` (ensure it's in your PATH):
```bash
make install
```

Or specify a different location:
```bash
make install PREFIX=/usr/local
```

Uninstall:
```bash
make uninstall
```

## Building

```bash
make build
# or
go build -o hstat .
```

Cross-compile for macOS:
```bash
GOOS=darwin GOARCH=arm64 go build -o hstat .
```

## Testing

Run all tests:
```bash
go test ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

Run tests for a specific package:
```bash
go test -v ./parser
go test -v ./store
go test -v ./ui
```

Run benchmarks:
```bash
go test -bench=. ./store
```

## Project Structure

```
hstat/
├── main.go           # Entry point, stdin reading, signal handling
├── parser/
│   ├── parser.go     # Heroku router log parsing
│   └── parser_test.go
├── store/
│   ├── store.go      # Time-windowed data storage and aggregation
│   └── store_test.go
└── ui/
    ├── model.go      # Bubbletea model and messages
    ├── update.go     # Key handling and commands
    ├── view.go       # Rendering logic
    ├── styles.go     # Lipgloss styles
    └── ui_test.go
```
