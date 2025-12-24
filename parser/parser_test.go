package parser

import (
	"testing"
)

func TestParse_ValidRouterLog(t *testing.T) {
	line := `2024-01-15T10:30:00.000000+00:00 heroku[router]: at=info method=GET path="/api/users" host=example.com request_id=abc123 fwd="1.2.3.4" dyno=web.1 connect=1ms service=25ms status=200 bytes=1234 protocol=https`

	entry := Parse(line)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	if entry.Status != 200 {
		t.Errorf("expected status 200, got %d", entry.Status)
	}
	if entry.Service != 25 {
		t.Errorf("expected service 25ms, got %d", entry.Service)
	}
	if entry.Connect != 1 {
		t.Errorf("expected connect 1ms, got %d", entry.Connect)
	}
	if entry.Host != "example.com" {
		t.Errorf("expected host example.com, got %s", entry.Host)
	}
	if entry.IP != "1.2.3.4" {
		t.Errorf("expected IP 1.2.3.4, got %s", entry.IP)
	}
}

func TestParse_MultipleIPsInFwd(t *testing.T) {
	line := `2024-01-15T10:30:00.000000+00:00 heroku[router]: at=info method=GET path="/" host=example.com fwd="1.2.3.4, 5.6.7.8" status=200 service=10ms connect=1ms`

	entry := Parse(line)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	// Should take first IP from chain
	if entry.IP != "1.2.3.4" {
		t.Errorf("expected IP 1.2.3.4, got %s", entry.IP)
	}
}

func TestParse_UnquotedFwd(t *testing.T) {
	line := `2024-01-15T10:30:00.000000+00:00 heroku[router]: at=info method=GET path="/" host=example.com fwd=1.2.3.4 status=200 service=10ms connect=1ms`

	entry := Parse(line)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	if entry.IP != "1.2.3.4" {
		t.Errorf("expected IP 1.2.3.4, got %s", entry.IP)
	}
}

func TestParse_EmptyFwd(t *testing.T) {
	line := `2024-01-15T10:30:00.000000+00:00 heroku[router]: at=info method=GET path="/" host=example.com fwd="" status=200 service=10ms connect=1ms`

	entry := Parse(line)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	if entry.IP != "" {
		t.Errorf("expected empty IP, got %s", entry.IP)
	}
}

func TestParse_NonRouterLog(t *testing.T) {
	lines := []string{
		`2024-01-15T10:30:00.000000+00:00 app[web.1]: Starting process`,
		`2024-01-15T10:30:00.000000+00:00 heroku[web.1]: State changed`,
		`some random text`,
		``,
	}

	for _, line := range lines {
		entry := Parse(line)
		if entry != nil {
			t.Errorf("expected nil for non-router log %q, got entry", line)
		}
	}
}

func TestParse_NoStatus(t *testing.T) {
	line := `2024-01-15T10:30:00.000000+00:00 heroku[router]: at=info method=GET path="/" host=example.com`

	entry := Parse(line)
	if entry != nil {
		t.Error("expected nil for log without status")
	}
}

func TestParse_VariousStatusCodes(t *testing.T) {
	tests := []struct {
		status int
		line   string
	}{
		{200, `heroku[router]: status=200 service=10ms`},
		{301, `heroku[router]: status=301 service=10ms`},
		{404, `heroku[router]: status=404 service=10ms`},
		{500, `heroku[router]: status=500 service=10ms`},
		{503, `heroku[router]: status=503 service=10ms`},
	}

	for _, tc := range tests {
		entry := Parse(tc.line)
		if entry == nil {
			t.Errorf("expected entry for status %d", tc.status)
			continue
		}
		if entry.Status != tc.status {
			t.Errorf("expected status %d, got %d", tc.status, entry.Status)
		}
	}
}

func TestParse_LargeServiceTime(t *testing.T) {
	line := `heroku[router]: status=200 service=30000ms connect=5000ms host=slow.example.com`

	entry := Parse(line)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	if entry.Service != 30000 {
		t.Errorf("expected service 30000ms, got %d", entry.Service)
	}
	if entry.Connect != 5000 {
		t.Errorf("expected connect 5000ms, got %d", entry.Connect)
	}
}

func TestParse_MissingOptionalFields(t *testing.T) {
	// Only status is required
	line := `heroku[router]: status=200`

	entry := Parse(line)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	if entry.Status != 200 {
		t.Errorf("expected status 200, got %d", entry.Status)
	}
	if entry.Service != 0 {
		t.Errorf("expected service 0, got %d", entry.Service)
	}
	if entry.Connect != 0 {
		t.Errorf("expected connect 0, got %d", entry.Connect)
	}
	if entry.Host != "" {
		t.Errorf("expected empty host, got %s", entry.Host)
	}
	if entry.IP != "" {
		t.Errorf("expected empty IP, got %s", entry.IP)
	}
}

func TestParse_Path(t *testing.T) {
	line := `2024-01-15T10:30:00.000000+00:00 heroku[router]: at=info method=GET path="/api/users" host=example.com fwd="1.2.3.4" status=200 service=25ms`

	entry := Parse(line)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	if entry.Path != "/api/users" {
		t.Errorf("expected path /api/users, got %s", entry.Path)
	}
}

func TestParse_PathWithQueryString(t *testing.T) {
	line := `heroku[router]: path="/api/users?page=1&limit=10" host=example.com status=200 service=25ms`

	entry := Parse(line)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	// Should extract path without query string
	if entry.Path != "/api/users" {
		t.Errorf("expected path /api/users (without query), got %s", entry.Path)
	}
}

func TestParse_PathRoot(t *testing.T) {
	line := `heroku[router]: path="/" host=example.com status=200 service=25ms`

	entry := Parse(line)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	if entry.Path != "/" {
		t.Errorf("expected path /, got %s", entry.Path)
	}
}

func TestParse_PathMissing(t *testing.T) {
	line := `heroku[router]: host=example.com status=200 service=25ms`

	entry := Parse(line)
	if entry == nil {
		t.Fatal("expected entry, got nil")
	}

	if entry.Path != "" {
		t.Errorf("expected empty path, got %s", entry.Path)
	}
}
