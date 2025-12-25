# hstat Auto-Layout System Specification

## Overview

Redesign hstat's TUI to automatically fill the terminal and adapt to window resizing. The layout should maximize use of available space, respond gracefully to tmux pane resizing, and provide a consistent bordered visual style.

## Layout Structure

### Sections (Top to Bottom)

1. **Header/Stats** - Title bar with timing stats
2. **Status Codes** - Grouped by category (1xx, 2xx, 3xx, 4xx, 5xx)
3. **Data Sections** - Hosts, IPs, and Paths (always visible)

All sections are bordered with sharp corners (`┌─────┐` style).

### Header/Stats Section

```
┌─ hstat | 2m30s | 1.2k reqs | 45.2/s ─────────────────────────────────┐
│ Response: avg 45 | p50 32 | p95 120 | p99 250 | max 312              │
│ Connect:  avg 12 | max 89                                            │
└──────────────────────────────────────────────────────────────────────┘
```

Contents:
- Title with elapsed time, request count, current rate
- Error rate indicators (4xx:X.X% 5xx:X.X%) with trend arrows
- Stream status warnings (if applicable)
- Response time stats: avg, p50, p95, p99, max
- Connect time stats: avg, max

### Status Codes Section

Displayed in horizontal columns grouped by category:

```
┌─ Status Codes ───────────────────────────────────────────────────────┐
│   1xx (-)      2xx (94.9%)   3xx (1.9%)   4xx (2.4%)    5xx (0.7%)   │
│                200: 850 (90.3%)  301: 18 (1.9%)  404: 19 (2.0%)  500: 5 (0.5%)  │
│                201: 42 (4.5%)                    422: 2 (0.2%)   502: 2 (0.2%)  │
│                                                  400: 2 (0.2%)                  │
└──────────────────────────────────────────────────────────────────────┘
```

- Header row shows category with percentage: `2xx (94.9%)`
- Individual codes below with: `code: count (%)`
- Colors preserved: 2xx green, 4xx yellow, 5xx red
- No count in header row (percentage only)

**Responsive behavior:**
- Wide: All 5 columns side by side
- Medium: Wrap to 2-3 rows (e.g., 1xx/2xx/3xx on row 1, 4xx/5xx on row 2)
- Narrow: Stack vertically
- Very tight space: Collapse to grouped totals only (no individual codes)

### Data Sections (Hosts, IPs, Paths)

All three sections are always visible with identical column structure:

```
┌─ Hosts (42) ─────────────────────────────────────────────────────────┐
│  Host                          Count      %    4xx    5xx            │
│  api.example.com                1,234   45.2%   0.5    0.1           │
│  www.example.com                  892   32.7%   2.1    0.3           │
│  admin.example.com                234    8.6%     -      -           │
│  (other)                          367   13.5%                        │
└──────────────────────────────────────────────────────────────────────┘
```

- Columns: Label, Count, Percentage, 4xx rate, 5xx rate
- Show as many rows as fit in available space
- Always show "(other)" row when there are more items than displayed
- Error rate columns show `-` when rate is 0

**Paths section note:** Paths tend to be longer strings. In multi-column layouts, paths get the widest column or full width.

### Responsive Data Section Layout

Layout adapts based on terminal width:

**Wide terminal (3 columns):**
```
┌─ Hosts ─────────────┐ ┌─ IPs ──────────────┐ ┌─ Paths ─────────────────────┐
│                     │ │                    │ │                             │
│                     │ │                    │ │                             │
└─────────────────────┘ └────────────────────┘ └─────────────────────────────┘
```

**Medium terminal (2 columns + 1 below):**
```
┌─ Hosts ─────────────────────┐ ┌─ IPs ──────────────────────┐
│                             │ │                            │
└─────────────────────────────┘ └────────────────────────────┘
┌─ Paths ────────────────────────────────────────────────────┐
│                                                            │
└────────────────────────────────────────────────────────────┘
```

**Narrow terminal (stacked):**
```
┌─ Hosts ────────────────────────────────────────────────────┐
│                                                            │
└────────────────────────────────────────────────────────────┘
┌─ IPs ──────────────────────────────────────────────────────┐
│                                                            │
└────────────────────────────────────────────────────────────┘
┌─ Paths ────────────────────────────────────────────────────┐
│                                                            │
└────────────────────────────────────────────────────────────┘
```

**Breakpoints:** Calculate dynamically based on minimum useful column widths:
- Minimum host column: ~30 chars
- Minimum IP column: ~25 chars
- Minimum path column: ~40 chars
- Borders/padding: ~6 chars per box

### Space Allocation

When space is limited:
- Prioritize the active section (the one with cursor focus)
- If only room for minimal items, show more rows in the focused section
- Equal split between hosts/IPs/paths when no space pressure

## Navigation & Interaction

### Navigable Sections

Only Hosts and IPs are navigable (Tab cycles between them). Paths is display-only.

### Active Section Indicator

- Active section has **magenta/purple border**
- Inactive sections have default border color

### Selection Styling

- Selected row: **bold only** (no underline)
- Cursor indicator: `>` prefix on selected row

### Keyboard Shortcuts

No changes to existing shortcuts:
- `Tab` / `Shift+Tab` / `l` / `h`: Navigate between Hosts and IPs
- `j` / `k` / arrows: Move cursor within section
- `g` / `G`: Jump to top/bottom
- `Enter`: Filter by selected host/IP
- `Esc`: Clear filter (or quit if no filter)
- `w`: Whois lookup (IP selected)
- `i`: ipinfo.io lookup (IP selected)
- `?`: Show help
- `q` / `Ctrl+C`: Quit

### Footer

No persistent footer. Help available via `?` key. This reclaims a row for data.

## Filtering Behavior

When a host or IP is selected and Enter is pressed:

1. **Filter indicator** appears in the filtered section's title:
   ```
   ┌─ Hosts (42) [host=api.example.com] ───────────────────────────────┐
   ```

2. **Source section is dimmed** (the section you filtered by)

3. **Other sections filter accordingly:**
   - Status codes: Show only codes for filtered host/IP
   - IPs (when filtering by host): Show only IPs that accessed that host
   - Hosts (when filtering by IP): Show only hosts accessed by that IP
   - Paths: Show only paths for the filtered host/IP

4. **Esc clears the filter**

When unfiltered, Paths shows all paths across all hosts/IPs.

## Modals

### Styling

- Sharp-corner borders matching main layout
- Try **without background dimming** (needs testing - may revert to dimming)
- Centered in terminal

### Types

1. **Whois modal** - `w` key when IP selected
2. **ipinfo.io modal** - `i` key when IP selected
3. **Help modal** - `?` key (converted from full-screen to modal)

## Resize Behavior

- **Instant recalculation** on every resize event
- **Cursor clamping**: If cursor is on row 15 and resize leaves only 10 rows, cursor moves to row 10

## Terminal Size

- **Minimum: 60x20** (increased from 40x15)
- Shows "Terminal too small" message below minimum

## Flag Changes

- **Remove `-n` flag entirely**
- Number of visible items is always auto-calculated from available space

## Implementation Notes

### Layout Calculation Order

1. Get terminal dimensions
2. Reserve space for header/stats section (fixed height)
3. Calculate status codes section height (depends on which codes are present and width)
4. Remaining height goes to data sections (hosts/IPs/paths)
5. Determine column layout based on width breakpoints
6. Divide remaining height among visible data sections (equal split, or prioritize active section if tight)
7. Calculate number of items that fit in each section

### Items to Test During Implementation

1. Modal appearance without background dimming - does it have enough contrast?
2. Status code column wrapping at various widths
3. Very long hostnames/paths truncation
4. Rapid resize behavior (tmux pane dragging)
5. Behavior at exactly 60x20 minimum size

### Files to Modify

- `ui/view.go` - Main rendering logic, layout calculations
- `ui/model.go` - Remove topN field, add layout state
- `ui/styles.go` - Add border styles, active section style (magenta)
- `ui/update.go` - Minor updates for layout state
- `main.go` - Remove `-n` flag
- `store/store.go` - May need to support dynamic topN in queries
