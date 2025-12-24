# hstat TODO

## Completed

### Bugs Fixed

- [x] **Terminal corruption on exit** - Fixed by adding `stty sane` cleanup on exit and proper signal handling.
- [x] **Modal position shifting** - Fixed modal to use fixed terminal dimensions instead of background content length.

### Features Implemented

- [x] **Whois lookup modal** - Press `w` on a selected IP to see whois data in a modal
  - Shells out to `whois` command
  - Filters out comment lines for cleaner output
  - Display in overlay modal, dismiss with Esc/Enter/q

- [x] **ipinfo API lookup** - Press `i` on a selected IP to query ipinfo.io API
  - Queries ipinfo.io API (free, no key for basic: org, country, city, location)
  - Faster and more structured than whois
  - Display in overlay modal, dismiss with Esc/Enter/q

- [x] **Test suite** - Comprehensive tests for all packages
  - Parser tests: log line parsing, edge cases
  - Store tests: aggregation, filtering, percentiles, pruning
  - UI tests: key handling, modal rendering, helper functions

## Future Ideas

- [ ] Request method breakdown (GET/POST/etc.)
- [ ] Path/endpoint stats (top N paths)
- [ ] Export current stats to JSON
- [ ] Configurable colors/theme
- [ ] Dyno distribution view
- [ ] Scrollable modal content for long whois output
- [ ] Keyboard shortcut to copy IP to clipboard
