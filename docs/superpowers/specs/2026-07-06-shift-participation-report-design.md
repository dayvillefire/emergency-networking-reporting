# Shift Participation Report — Design Spec

**Date:** 2026-07-06
**Status:** Approved

## Purpose

A standalone CLI tool that lets department leadership select a group of users and see, per shift, what percentage of calls each user responded to — across three rolling time windows: previous quarter, previous month, and current month.

## Motivation

The existing `en-responder-stats` tool shows personnel participation tiers (>=30%, >=20%, >=10%) aggregated across all shifts combined. Leadership wants shift-aware participation data: who's showing up for which shifts, with percentages relative to the shift's total call volume.

## Architecture

New standalone tool: `cmd/shift-report/`

```
cmd/shift-report/
  main.go        Entry point, flags, orchestration
  tui.go         Multi-select user picker (Bubble Tea)
  compute.go     Shift-by-user statistics computation
  output.go      Terminal text table + CSV output
  html.go        HTML report generation
  pdf.go         PDF report generation
```

### Additions to existing shared code

| File | Change |
|------|--------|
| `internal/shared/dates.go` | Add `CurrentMonth()` helper returning `DateRange` from 1st of current month through today |

### What this tool reuses

| Component | Source |
|-----------|--------|
| Token reading | `shared.ReadToken()` |
| API client | `enapi.NewClient()` |
| Incident fetching (NERIS + NFIRS) | `shared.FetchIncidents()` |
| Date filtering | `shared.FilterByDateRange()` |
| Period helpers | `shared.LastMonth()`, `shared.LastQuarter()`, new `shared.CurrentMonth()` |
| Name resolution | `shared.BuildNameMap()` |
| User listing | `enapi.ListUsers()` |
| Normalized incident data | `shared.NormalizedIncident` (Shift, OnScenePersonnel, NotOnScenePersonnel) |

### Why a standalone tool

The existing `en-report` TUI already has 7 states and produces a general-purpose department report. Adding user selection and a shift-participation section would bloat it. A focused tool with its own interactive user picker is simpler to use and maintain.

## Data Flow

```
1. Read token → create enapi.Client
2. Fetch all users via ListUsers → interactive multi-select picker
3. Fetch all incidents (NERIS + NFIRS) across widest range
   (previous quarter start through today)
4. For each of 3 periods (previous quarter, previous month, current month):
   a. FilterByDateRange(incidents, period.Start, period.End)
   b. Collect all unique shift values present in the subset
   c. For each shift:
      - Count total calls with that shift value
      - For each selected user, count how many of those calls
        they appeared on (OnScenePersonnel + NotOnScenePersonnel)
      - Compute: (user calls / shift total calls) × 100
5. Output based on flags: terminal table (default), CSV, HTML, PDF
```

### Key data structure

```go
type ShiftUserRecord struct {
    UserName     string
    Shift        string
    CallCount    int
    TotalInShift int
    Pct          float64
}
```

This differs from `shared.PersonnelRecord` by being shift-scoped rather than tier-scoped.

## User Interface

### Interactive Flow

```
stateStart → stateFetchingUsers → stateSelectingUsers
→ stateFetchingIncidents → stateComputing → stateDone
```

**User selection screen:** Multi-select list of all users (Last, First format). Space toggles selection with a ✓ marker. Up/down/j/k to navigate. "Select All" / "Deselect All" at the top. Footer shows selection count.

### Flags

| Flag | Behavior |
|------|----------|
| (none) | Terminal table output |
| `--csv` | CSV to stdout |
| `--html` | Save HTML file |
| `--pdf` | Save PDF file |
| `--department` | Filter incidents to a specific station (skip department picker) |

`--html` + `--pdf` together generates both. Without either flag, terminal output only.

## Output Formats

### Terminal Table

Three blocks, one per period. Dynamic columns based on shifts present:

```
Previous Quarter (Apr 7 – Jul 6, 2026) — 312 total calls
                Day Shift     Duty Night    Night Shift
Smith, John     52.3% (27)    18.4% (9)     —
Doe, Jane       38.5% (20)    42.9% (21)    12.5% (3)
```

Empty cells shown as `—`. Users sorted alphabetically. Percentages show one decimal place.

### CSV

One row per user per period. Period/range context columns + dynamic shift columns:

```
period, period_start, period_end, total_calls, user,
  day_shift_pct, day_shift_calls, duty_night_pct, duty_night_calls, ...
```

### HTML

Styled table per period. Reuses CSS patterns from `cmd/report/html.go`. No Chart.js charts — shift-by-user data is tabular, not visual.

### PDF

Single PDF with a section per period. Reuses table-drawing patterns from `cmd/report/pdf.go` (section headers, addTableRow, auto page-break via ensureSpace). No charts for PDF either.

## Error Handling

- Missing API token: print to stderr, exit 1 (matches existing tools)
- API fetch failures: print error, exit 1
- Zero users selected: print "No users selected" to stderr, exit 1
- Period with zero calls: show "(no calls in period)" instead of an empty table
- Shift value is empty string: display as "(unknown)" (matches existing computeByShift)

## Build

Add `build-shift-report` target to Makefile alongside existing `build-responder`:

```makefile
build-shift-report:
    go build -o en-shift-report ./cmd/shift-report
```

## Verification

1. `go build ./cmd/shift-report` compiles without errors
2. `./en-shift-report` launches interactive user picker with real users from API
3. Select 2-3 users, confirm → spinner → terminal table with 3 period blocks
4. `./en-shift-report --csv` produces valid CSV with correct column structure
5. `./en-shift-report --html` produces a valid HTML file that renders correctly in a browser
6. `./en-shift-report --pdf` produces a valid PDF with all 3 periods
7. `./en-shift-report --department "Station X"` filters to that station's calls
8. Edge case: zero users selected exits cleanly with message
