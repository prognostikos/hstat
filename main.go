package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/betternow/hstat/parser"
	"github.com/betternow/hstat/store"
	"github.com/betternow/hstat/ui"
	tea "github.com/charmbracelet/bubbletea"
)

const version = "0.1.0"

func main() {
	// Parse flags
	showVersion := flag.Bool("version", false, "Show version and exit")
	showVersionShort := flag.Bool("v", false, "Show version and exit")
	windowStr := flag.String("window", "5m", "Percentile calculation window (e.g., 5m, 10m, 1h, or 'all')")
	windowShort := flag.String("w", "", "Shorthand for -window")
	topN := flag.Int("top", 15, "Number of hosts/IPs to show")
	topNShort := flag.Int("n", 0, "Shorthand for -top")
	refreshStr := flag.String("refresh", "1s", "Screen refresh interval")
	refreshShort := flag.String("r", "", "Shorthand for -refresh")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "hstat v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "Usage: heroku logs --tail -a myapp | hstat [options]\n\n")
		fmt.Fprintf(os.Stderr, "Real-time Heroku router log monitor with interactive filtering.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Handle version flag
	if *showVersion || *showVersionShort {
		fmt.Printf("hstat v%s\n", version)
		os.Exit(0)
	}

	// Handle shorthand flags
	if *windowShort != "" {
		windowStr = windowShort
	}
	if *topNShort != 0 {
		topN = topNShort
	}
	if *refreshShort != "" {
		refreshStr = refreshShort
	}

	// Parse window duration
	var window time.Duration
	if *windowStr != "all" {
		var err error
		window, err = time.ParseDuration(*windowStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid window duration: %s\n", *windowStr)
			os.Exit(1)
		}
	}

	// Parse refresh duration
	refresh, err := time.ParseDuration(*refreshStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid refresh duration: %s\n", *refreshStr)
		os.Exit(1)
	}

	// Check if stdin is a terminal (we need piped input)
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		fmt.Fprintln(os.Stderr, "Error: hstat requires log input via stdin")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage: heroku logs --tail -a myapp | hstat")
		fmt.Fprintln(os.Stderr, "   or: hstat < router.log")
		os.Exit(1)
	}

	// Create store and model
	s := store.New(window)
	m := ui.NewModel(s, *topN, refresh)

	// Open TTY for keyboard input (since stdin is the log pipe)
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening /dev/tty: %v\n", err)
		os.Exit(1)
	}
	defer tty.Close()

	// Create program with explicit TTY input
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithInput(tty))

	// Handle signals for clean exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		p.Quit()
	}()

	// Start stdin reader in goroutine
	go readStdin(p, s)

	// Run program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func readStdin(p *tea.Program, s *store.Store) {
	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		line := scanner.Text()
		entry := parser.Parse(line)
		if entry != nil {
			p.Send(ui.EntryMsg{Entry: entry})
		}
	}

	// Signal that stream has ended
	p.Send(ui.StreamEndedMsg{})
}
