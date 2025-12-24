package parser

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Entry represents a parsed Heroku router log line
type Entry struct {
	Timestamp time.Time
	Status    int
	Service   int // ms
	Connect   int // ms
	Host      string
	Path      string
	IP        string // first from fwd chain
}

var (
	statusRe  = regexp.MustCompile(`status=(\d+)`)
	serviceRe = regexp.MustCompile(`service=(\d+)ms`)
	connectRe = regexp.MustCompile(`connect=(\d+)ms`)
	hostRe    = regexp.MustCompile(`host=([^\s]+)`)
	pathRe    = regexp.MustCompile(`path="([^"]*)"`)
	fwdRe     = regexp.MustCompile(`fwd="([^"]*)"`)     // quoted, possibly empty
	fwdAltRe  = regexp.MustCompile(`fwd=([0-9][^\s]*)`) // unquoted IP
)

// Parse parses a Heroku router log line into an Entry.
// Returns nil if the line is not a valid router log.
func Parse(line string) *Entry {
	// Must be a router log line (contains "heroku[router]")
	if !strings.Contains(line, "heroku[router]") {
		return nil
	}

	// Must have status
	statusMatch := statusRe.FindStringSubmatch(line)
	if statusMatch == nil {
		return nil
	}

	status, _ := strconv.Atoi(statusMatch[1])

	entry := &Entry{
		Timestamp: time.Now(),
		Status:    status,
	}

	if m := serviceRe.FindStringSubmatch(line); m != nil {
		entry.Service, _ = strconv.Atoi(m[1])
	}

	if m := connectRe.FindStringSubmatch(line); m != nil {
		entry.Connect, _ = strconv.Atoi(m[1])
	}

	if m := hostRe.FindStringSubmatch(line); m != nil {
		entry.Host = m[1]
	}

	if m := pathRe.FindStringSubmatch(line); m != nil {
		path := m[1]
		// Strip query string
		if idx := strings.Index(path, "?"); idx != -1 {
			path = path[:idx]
		}
		entry.Path = path
	}

	if m := fwdRe.FindStringSubmatch(line); m != nil && m[1] != "" {
		// Take first IP from chain (e.g., "1.2.3.4, 5.6.7.8" -> "1.2.3.4")
		entry.IP = strings.Split(m[1], ",")[0]
		entry.IP = strings.TrimSpace(entry.IP)
	} else if m := fwdAltRe.FindStringSubmatch(line); m != nil {
		// Try unquoted format
		entry.IP = strings.Split(m[1], ",")[0]
		entry.IP = strings.TrimSpace(entry.IP)
	}

	return entry
}
