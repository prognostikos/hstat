package ui

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case EntryMsg:
		m.store.Add(msg.Entry)
		m.lastEntryTime = time.Now()
		return m, nil

	case TickMsg:
		m.refreshData()
		return m, tickCmd(m.refreshRate)

	case StreamEndedMsg:
		m.streamEnded = true
		m.refreshData()
		return m, nil

	case WhoisResultMsg:
		m.modal.Loading = false
		if msg.Err != nil {
			m.modal.Content = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			m.modal.Content = msg.Content
		}
		return m, nil

	case IpinfoResultMsg:
		m.modal.Loading = false
		if msg.Err != nil {
			m.modal.Content = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			m.modal.Content = msg.Content
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Modal dismissal
	if m.modal.Visible {
		if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
			m.modal.Visible = false
			m.modal.Content = ""
			return m, nil
		}
		// Ignore other keys when modal is visible
		return m, nil
	}

	// Help toggle
	if msg.String() == "?" {
		m.showHelp = !m.showHelp
		return m, nil
	}

	// If help is showing, any key dismisses it
	if m.showHelp {
		m.showHelp = false
		return m, nil
	}

	switch msg.String() {
	// Quit
	case "q", "ctrl+c":
		return m, tea.Quit

	// Clear filter or quit
	case "esc":
		if m.filter.Host != "" || m.filter.IP != "" {
			m.filter = Filter{}
			m.refreshData()
			return m, nil
		}
		return m, tea.Quit

	// Whois lookup
	case "w":
		if m.section == SectionIPs && m.ipCursor < len(m.topIPs) {
			ip := m.topIPs[m.ipCursor].Label
			if ip != "" && ip != "(unknown)" {
				m.modal.Visible = true
				m.modal.Title = fmt.Sprintf("whois %s", ip)
				m.modal.Loading = true
				m.modal.Content = "Loading..."
				return m, runWhois(ip)
			}
		}
		return m, nil

	// IP info lookup (via ipinfo.io API)
	case "i":
		if m.section == SectionIPs && m.ipCursor < len(m.topIPs) {
			ip := m.topIPs[m.ipCursor].Label
			if ip != "" && ip != "(unknown)" {
				m.modal.Visible = true
				m.modal.Title = fmt.Sprintf("ipinfo %s", ip)
				m.modal.Loading = true
				m.modal.Content = "Loading..."
				return m, runIpinfo(ip)
			}
		}
		return m, nil

	// Section navigation
	case "tab", "l":
		m.section = (m.section + 1) % 2
		return m, nil

	case "shift+tab", "h":
		if m.section == 0 {
			m.section = 1
		} else {
			m.section--
		}
		return m, nil

	// Cursor movement
	case "j", "down":
		m.moveCursor(1)
		return m, nil

	case "k", "up":
		m.moveCursor(-1)
		return m, nil

	case "g":
		m.moveCursorTo(0)
		return m, nil

	case "G":
		m.moveCursorToEnd()
		return m, nil

	// Filter
	case "enter":
		m.applyFilter()
		return m, nil
	}

	return m, nil
}

func (m *Model) moveCursor(delta int) {
	switch m.section {
	case SectionHosts:
		m.hostCursor += delta
		if m.hostCursor < 0 {
			m.hostCursor = 0
		}
		if m.hostCursor >= len(m.topHosts) {
			m.hostCursor = max(0, len(m.topHosts)-1)
		}
	case SectionIPs:
		m.ipCursor += delta
		if m.ipCursor < 0 {
			m.ipCursor = 0
		}
		if m.ipCursor >= len(m.topIPs) {
			m.ipCursor = max(0, len(m.topIPs)-1)
		}
	}
}

func (m *Model) moveCursorTo(pos int) {
	switch m.section {
	case SectionHosts:
		m.hostCursor = pos
	case SectionIPs:
		m.ipCursor = pos
	}
}

func (m *Model) moveCursorToEnd() {
	switch m.section {
	case SectionHosts:
		m.hostCursor = max(0, len(m.topHosts)-1)
	case SectionIPs:
		m.ipCursor = max(0, len(m.topIPs)-1)
	}
}

func (m *Model) applyFilter() {
	switch m.section {
	case SectionHosts:
		if m.hostCursor < len(m.topHosts) {
			m.filter = Filter{Host: m.topHosts[m.hostCursor].Label}
			m.refreshData()
		}
	case SectionIPs:
		if m.ipCursor < len(m.topIPs) {
			m.filter = Filter{IP: m.topIPs[m.ipCursor].Label}
			m.refreshData()
		}
	}
}

// runWhois executes whois command and returns result
func runWhois(ip string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("whois", ip)
		output, err := cmd.Output()
		if err != nil {
			return WhoisResultMsg{IP: ip, Err: err}
		}
		// Trim and limit output
		content := strings.TrimSpace(string(output))
		lines := strings.Split(content, "\n")
		// Filter out comment lines and empty lines for cleaner output
		var filtered []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "%") {
				filtered = append(filtered, line)
			}
		}
		return WhoisResultMsg{IP: ip, Content: strings.Join(filtered, "\n")}
	}
}

// IpinfoResponse represents the ipinfo.io API response
type IpinfoResponse struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Loc      string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
}

// runIpinfo queries ipinfo.io API and returns result
func runIpinfo(ip string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(fmt.Sprintf("https://ipinfo.io/%s/json", ip))
		if err != nil {
			return IpinfoResultMsg{IP: ip, Err: err}
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return IpinfoResultMsg{IP: ip, Err: err}
		}

		var info IpinfoResponse
		if err := json.Unmarshal(body, &info); err != nil {
			return IpinfoResultMsg{IP: ip, Err: err}
		}

		// Format the response nicely
		var b strings.Builder
		b.WriteString(fmt.Sprintf("IP:       %s\n", info.IP))
		if info.Hostname != "" {
			b.WriteString(fmt.Sprintf("Hostname: %s\n", info.Hostname))
		}
		if info.Org != "" {
			b.WriteString(fmt.Sprintf("Org:      %s\n", info.Org))
		}
		if info.City != "" || info.Region != "" || info.Country != "" {
			location := strings.Join(nonEmpty(info.City, info.Region, info.Country), ", ")
			b.WriteString(fmt.Sprintf("Location: %s\n", location))
		}
		if info.Loc != "" {
			b.WriteString(fmt.Sprintf("Coords:   %s\n", info.Loc))
		}
		if info.Timezone != "" {
			b.WriteString(fmt.Sprintf("Timezone: %s\n", info.Timezone))
		}
		if info.Postal != "" {
			b.WriteString(fmt.Sprintf("Postal:   %s\n", info.Postal))
		}

		return IpinfoResultMsg{IP: ip, Content: strings.TrimSpace(b.String())}
	}
}

// nonEmpty filters out empty strings
func nonEmpty(strs ...string) []string {
	var result []string
	for _, s := range strs {
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}
