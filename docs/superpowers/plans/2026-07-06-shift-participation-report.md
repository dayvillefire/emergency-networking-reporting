# Shift Participation Report — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a standalone CLI tool (`en-shift-report`) that lets users interactively select personnel and see per-shift call participation percentages across three rolling periods.

**Architecture:** New `cmd/shift-report/` package with 6 files: main (entry + flags), tui (Bubble Tea multi-select picker), compute (shift-by-user stats), output (terminal + CSV), html (HTML report), pdf (PDF report). One addition to `internal/shared/dates.go` for `CurrentMonth()`. Reuses `shared.FetchIncidents`, `shared.FilterByDateRange`, `shared.BuildNameMap`, and existing shared types.

**Tech Stack:** Go stdlib + Bubble Tea (TUI) + gofpdf (PDF) + html/template (HTML)

---

### Task 1: Add `CurrentMonth()` helper to shared dates

**Files:**
- Modify: `internal/shared/dates.go`

- [ ] **Step 1: Add the CurrentMonth function**

Append after the `LastYear` function block (after line 36):

```go
// CurrentMonth returns a DateRange from the 1st of the current month through today.
func CurrentMonth() DateRange {
	now := time.Now()
	return DateRange{
		Start: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
		End:   now,
	}
}
```

This matches the existing pattern: `LastMonth`, `LastQuarter`, `LastYear` all return `DateRange` with `End: now`. The start is the 1st of the current month at midnight.

- [ ] **Step 2: Verify compilation**

```bash
go build ./internal/shared/
```

Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/shared/dates.go
git commit -m "feat: add CurrentMonth() date range helper"
```

---

### Task 2: Create compute.go — shift-by-user statistics

**Files:**
- Create: `cmd/shift-report/compute.go`

- [ ] **Step 1: Write the compute package file**

Create `cmd/shift-report/compute.go`:

```go
package main

import (
	"sort"

	"github.com/dayvillefire/emergency-networking-reporting/internal/shared"
)

// ShiftUserRecord holds participation stats for one user in one shift within a period.
type ShiftUserRecord struct {
	UserName     string
	Shift        string
	CallCount    int
	TotalInShift int
	Pct          float64
}

// PeriodResult holds all shift-user records for a single time period.
type PeriodResult struct {
	Label      string
	Range      shared.DateRange
	TotalCalls int
	Shifts     []string               // ordered shift names
	Records    []ShiftUserRecord      // sorted by UserName then Shift
}

// ComputeShiftParticipation calculates per-shift call percentages for each selected user
// across the given incidents. selectedUsers is a set of resolved "Last, First" names.
// nameMap maps raw personnel IDs to display names.
func ComputeShiftParticipation(
	incidents []shared.NormalizedIncident,
	selectedUsers map[string]bool,
	nameMap map[string]string,
) ([]string, []ShiftUserRecord) {
	// Collect unique shift names and count total calls per shift.
	shiftTotals := make(map[string]int)
	for _, inc := range incidents {
		shift := inc.Shift
		if shift == "" {
			shift = "(unknown)"
		}
		shiftTotals[shift]++
	}

	// Count per-user per-shift.
	// Key: userName + "||" + shift
	userShiftCounts := make(map[string]int)
	for _, inc := range incidents {
		shift := inc.Shift
		if shift == "" {
			shift = "(unknown)"
		}
		// Collect unique resolved names from both on-scene and not-on-scene.
		seen := make(map[string]bool)
		for _, raw := range inc.OnScenePersonnel {
			name := resolveName(raw, nameMap)
			if selectedUsers[name] {
				seen[name] = true
			}
		}
		for _, raw := range inc.NotOnScenePersonnel {
			name := resolveName(raw, nameMap)
			if selectedUsers[name] {
				seen[name] = true
			}
		}
		for name := range seen {
			key := name + "||" + shift
			userShiftCounts[key]++
		}
	}

	// Build ordered shift list (sorted alphabetically).
	var shifts []string
	for shift := range shiftTotals {
		shifts = append(shifts, shift)
	}
	sort.Strings(shifts)

	// Collect unique user names.
	userSet := make(map[string]bool)
	for name := range selectedUsers {
		userSet[name] = true
	}
	for key := range userShiftCounts {
		// key is "name||shift"
		idx := stringsLastIndex(key, "||")
		if idx >= 0 {
			userSet[key[:idx]] = true
		}
	}
	var users []string
	for name := range userSet {
		users = append(users, name)
	}
	sort.Strings(users)

	// Build records: one per user per shift.
	var records []ShiftUserRecord
	for _, user := range users {
		for _, shift := range shifts {
			key := user + "||" + shift
			count := userShiftCounts[key]
			total := shiftTotals[shift]
			pct := 0.0
			if total > 0 {
				pct = float64(count) / float64(total) * 100
			}
			records = append(records, ShiftUserRecord{
				UserName:     user,
				Shift:        shift,
				CallCount:    count,
				TotalInShift: total,
				Pct:          pct,
			})
		}
	}

	return shifts, records
}

// stringsLastIndex returns the last index of substr in s, or -1.
func stringsLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// resolveName looks up a raw personnel identifier in the name map.
func resolveName(raw string, nameMap map[string]string) string {
	if name, ok := nameMap[raw]; ok {
		return name
	}
	return raw
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./cmd/shift-report/
```

Expected: `compute.go` compiles. May complain about missing `main` function — that's fine, Task 7 adds it.

- [ ] **Step 3: Commit**

```bash
git add cmd/shift-report/compute.go
git commit -m "feat: add shift participation computation logic"
```

---

### Task 3: Create tui.go — multi-select user picker

**Files:**
- Create: `cmd/shift-report/tui.go`

- [ ] **Step 1: Write the TUI**

Create `cmd/shift-report/tui.go`:

```go
package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dayvillefire/emergency-networking-reporting/enapi"
	"github.com/dayvillefire/emergency-networking-reporting/internal/shared"
)

type shiftReportState int

const (
	srStateStart shiftReportState = iota
	srStateFetchingUsers
	srStateSelectingUsers
	srStateFetchingIncidents
	srStateComputing
	srStateDone
	srStateError
)

type userListMsg struct {
	users []string // "Last, First" format
	err   error
}

type incidentsFetchedMsg struct {
	incidents []shared.NormalizedIncident
	err       error
}

type resultsMsg struct {
	periods []PeriodResult
	err     error
}

type srModel struct {
	client  *enapi.Client
	state   shiftReportState
	spinner spinner.Model
	err     error

	// User selection
	allUsers      []string            // all names in "Last, First" format
	selected      map[int]bool        // index -> selected
	cursor        int
	nameMap       map[string]string

	// Results
	periods    []PeriodResult
	htmlPath   string
	pdfPath    string

	// Options
	outputHTML bool
	outputPDF  bool
	outputCSV  bool
	deptFilter string
}

var (
	srBannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(lipgloss.Color("#c41e3a")).
			Padding(1, 4).
			Align(lipgloss.Center)
	srTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#c41e3a")).MarginBottom(1)
	srSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#c41e3a")).Bold(true).PaddingLeft(2)
	srUnselected    = lipgloss.NewStyle().PaddingLeft(2)
	srHelpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888")).MarginTop(1)
	srSuccessStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ecc71")).Bold(true)
)

func newSRModel(client *enapi.Client, outputHTML, outputPDF, outputCSV bool, deptFilter string) srModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#c41e3a"))
	return srModel{
		client:     client,
		state:      srStateFetchingUsers,
		spinner:    s,
		selected:   make(map[int]bool),
		outputHTML: outputHTML,
		outputPDF:  outputPDF,
		outputCSV:  outputCSV,
		deptFilter: deptFilter,
	}
}

func (m srModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, fetchUsers(m.client))
}

func fetchUsers(client *enapi.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		users, err := client.ListUsers(ctx)
		if err != nil {
			return userListMsg{err: err}
		}
		var names []string
		nameMap := make(map[string]string)
		for _, u := range users {
			if !u.Active {
				continue
			}
			display := u.LastName + ", " + u.FirstName
			names = append(names, display)
			// Build name map inline: map both personnel ID and user ID
			nameMap[display] = display
			if u.PersonnelID != "" {
				nameMap[u.PersonnelID] = display
			}
			nameMap[fmt.Sprintf("%d", u.ID)] = display
		}
		sort.Strings(names)
		return userListMsg{users: names, err: nil}
	}
}

func fetchIncidentsCmd(client *enapi.Client, deptFilter string) tea.Cmd {
	return func() tea.Msg {
		fetchStart := shared.LastQuarter().Start
		fetchEnd := time.Now()
		incidents, err := shared.FetchIncidents(client, fetchStart, fetchEnd)
		if err != nil {
			return incidentsFetchedMsg{err: err}
		}
		if deptFilter != "" {
			var filtered []shared.NormalizedIncident
			for _, inc := range incidents {
				if strings.EqualFold(inc.Station, deptFilter) {
					filtered = append(filtered, inc)
				}
			}
			incidents = filtered
		}
		return incidentsFetchedMsg{incidents: incidents}
	}
}

func (m srModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.state == srStateFetchingUsers || m.state == srStateFetchingIncidents || m.state == srStateComputing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.state {
		case srStateStart:
			m.state = srStateFetchingUsers
			return m, m.spinner.Tick
		case srStateSelectingUsers:
			return m.handleUserKey(msg)
		case srStateDone, srStateError:
			if msg.String() == "enter" || msg.String() == "q" {
				return m, tea.Quit
			}
		}

	case userListMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = srStateError
			return m, nil
		}
		m.allUsers = msg.users
		// Default: select all
		for i := range m.allUsers {
			m.selected[i] = true
		}
		m.nameMap = make(map[string]string)
		for _, name := range m.allUsers {
			m.nameMap[name] = name
		}
		m.state = srStateSelectingUsers
		return m, nil

	case incidentsFetchedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = srStateError
			return m, nil
		}
		m.state = srStateComputing
		return m, computeResultsCmd(msg.incidents, m.selectedUsersSet(), m.nameMap)

	case resultsMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = srStateError
			return m, nil
		}
		m.periods = msg.periods
		// Generate output
		if m.outputHTML {
			path, err := saveShiftHTML(m.periods)
			if err == nil {
				m.htmlPath = path
			}
		}
		if m.outputPDF {
			path, err := saveShiftPDF(m.periods)
			if err == nil {
				m.pdfPath = path
			}
		}
		m.state = srStateDone
		return m, nil
	}
	return m, nil
}

func (m srModel) handleUserKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.allUsers) {
			m.cursor++
		}
	case " ":
		if m.cursor == 0 {
			// "Select All / Deselect All" row
			allSelected := true
			for i := range m.allUsers {
				if !m.selected[i] {
					allSelected = false
					break
				}
			}
			newVal := !allSelected
			for i := range m.allUsers {
				m.selected[i] = newVal
			}
		} else {
			idx := m.cursor - 1 // offset for "Select All" row
			m.selected[idx] = !m.selected[idx]
		}
	case "enter":
		// Check at least one selected
		any := false
		for i := range m.allUsers {
			if m.selected[i] {
				any = true
				break
			}
		}
		if !any {
			// stay, don't proceed
			return m, nil
		}
		m.state = srStateFetchingIncidents
		return m, tea.Batch(m.spinner.Tick, fetchIncidentsCmd(m.client, m.deptFilter))
	}
	return m, nil
}

func (m srModel) selectedUsersSet() map[string]bool {
	set := make(map[string]bool)
	for i, name := range m.allUsers {
		if m.selected[i] {
			set[name] = true
		}
	}
	return set
}

func computeResultsCmd(incidents []shared.NormalizedIncident, selectedUsers map[string]bool, nameMap map[string]string) tea.Cmd {
	return func() tea.Msg {
		periods := []struct {
			Label string
			Range shared.DateRange
		}{
			{"Previous Quarter", shared.LastQuarter()},
			{"Previous Month", shared.LastMonth()},
			{"Current Month", shared.CurrentMonth()},
		}

		var results []PeriodResult
		for _, p := range periods {
			subset := shared.FilterByDateRange(incidents, p.Range.Start, p.Range.End)
			shifts, records := ComputeShiftParticipation(subset, selectedUsers, nameMap)
			results = append(results, PeriodResult{
				Label:      p.Label,
				Range:      p.Range,
				TotalCalls: len(subset),
				Shifts:     shifts,
				Records:    records,
			})
		}
		return resultsMsg{periods: results}
	}
}

func (m srModel) View() string {
	switch m.state {
	case srStateStart:
		return m.viewBanner()
	case srStateFetchingUsers:
		return srCenter(fmt.Sprintf("\n\n  %s Loading users...\n\n", m.spinner.View()))
	case srStateSelectingUsers:
		return m.viewUserPicker()
	case srStateFetchingIncidents:
		return srCenter(fmt.Sprintf("\n\n  %s Fetching incidents...\n\n", m.spinner.View()))
	case srStateComputing:
		return srCenter(fmt.Sprintf("\n\n  %s Computing statistics...\n\n", m.spinner.View()))
	case srStateError:
		return srCenter(fmt.Sprintf("\n  Error: %v\n\n  Press q to quit.\n", m.err))
	case srStateDone:
		return m.viewDone()
	}
	return ""
}

func (m srModel) viewBanner() string {
	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(srBannerStyle.Render(" SHIFT PARTICIPATION REPORT "))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Render(
		lipgloss.NewStyle().Foreground(lipgloss.Color("#666")).Render("Select users to include in the report"),
	))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Render(
		srHelpStyle.Render("Press any key to begin"),
	))
	return b.String()
}

func (m srModel) viewUserPicker() string {
	var b strings.Builder
	b.WriteString(srTitleStyle.Render("Select Users"))
	b.WriteString(fmt.Sprintf("  %d selected\n\n", m.selectedCount()))
	// "Select All / Deselect All" row at position 0
	allSelected := m.selectedCount() == len(m.allUsers)
	toggleLabel := "Select All"
	if allSelected {
		toggleLabel = "Deselect All"
	}
	if m.cursor == 0 {
		b.WriteString(srSelectedStyle.Render(fmt.Sprintf("> [%s] %s", checkMark(allSelected), toggleLabel)))
	} else {
		b.WriteString(srUnselected.Render(fmt.Sprintf("  [%s] %s", checkMark(allSelected), toggleLabel)))
	}
	b.WriteString("\n")

	start := 0
	adjustedCursor := m.cursor - 1
	if adjustedCursor > 5 {
		start = adjustedCursor - 5
	}
	end := start + 12
	if end > len(m.allUsers) {
		end = len(m.allUsers)
	}
	if start > 0 {
		b.WriteString(fmt.Sprintf("  ↑ %d more...\n", start))
	}
	for i := start; i < end; i++ {
		checked := checkMark(m.selected[i])
		if i == adjustedCursor {
			b.WriteString(srSelectedStyle.Render(fmt.Sprintf("> [%s] %s", checked, m.allUsers[i])))
		} else {
			b.WriteString(srUnselected.Render(fmt.Sprintf("  [%s] %s", checked, m.allUsers[i])))
		}
		b.WriteString("\n")
	}
	if end < len(m.allUsers) {
		b.WriteString(fmt.Sprintf("  ↓ %d more...\n", len(m.allUsers)-end))
	}
	b.WriteString(srHelpStyle.Render("\n  ↑↓/jk navigate  ·  space toggle  ·  enter confirm  ·  ctrl+c quit"))
	return b.String()
}

func (m srModel) viewDone() string {
	var b strings.Builder
	b.WriteString(srTitleStyle.Render("Shift Participation Report"))
	b.WriteString("\n")
	b.WriteString(formatPeriodResults(m.periods))
	if m.htmlPath != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", srSuccessStyle.Render(fmt.Sprintf("HTML report saved: %s", m.htmlPath))))
	}
	if m.pdfPath != "" {
		b.WriteString(fmt.Sprintf("  %s\n", srSuccessStyle.Render(fmt.Sprintf("PDF report saved: %s", m.pdfPath))))
	}
	b.WriteString(srHelpStyle.Render("\n  Press q to quit"))
	return b.String()
}

func (m srModel) selectedCount() int {
	count := 0
	for i := range m.allUsers {
		if m.selected[i] {
			count++
		}
	}
	return count
}

func checkMark(v bool) string {
	if v {
		return "✓"
	}
	return " "
}

func srCenter(s string) string { return "\n" + s }
```

This matches the existing `cmd/report/tui.go` pattern: same Bubble Tea state machine, same spinner/lipgloss styles (colors and layouts match), same `ctrl+c` quit, same key handling.

- [ ] **Step 2: Verify compilation**

```bash
go build ./cmd/shift-report/
```

Expected: `tui.go` compiles. May still complain about missing `main`.

- [ ] **Step 3: Commit**

```bash
git add cmd/shift-report/tui.go
git commit -m "feat: add multi-select user picker TUI for shift report"
```

---

### Task 4: Create output.go — terminal table + CSV

**Files:**
- Create: `cmd/shift-report/output.go`

- [ ] **Step 1: Write output.go**

Create `cmd/shift-report/output.go`:

```go
package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

// formatPeriodResults returns the terminal text table for all periods.
func formatPeriodResults(periods []PeriodResult) string {
	var b strings.Builder
	for _, pr := range periods {
		b.WriteString(formatOnePeriod(pr))
		b.WriteString("\n")
	}
	return b.String()
}

// formatOnePeriod renders a single period block as a text table.
func formatOnePeriod(pr PeriodResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s (%s – %s) — %d total calls\n",
		pr.Label,
		pr.Range.Start.Format("Jan 2, 2006"),
		pr.Range.End.Format("Jan 2, 2006"),
		pr.TotalCalls,
	))
	if pr.TotalCalls == 0 || len(pr.Records) == 0 {
		b.WriteString("  (no calls in period)\n")
		return b.String()
	}

	w := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)

	// Header: user name column + one column per shift
	fmt.Fprint(w, "\t")
	for _, shift := range pr.Shifts {
		fmt.Fprintf(w, "%s\t", shift)
	}
	fmt.Fprintln(w)

	// Pivot records back to user-per-row.
	userRows := pivotToUserRows(pr.Shifts, pr.Records)
	for _, row := range userRows {
		fmt.Fprintf(w, "%s\t", row.UserName)
		for _, cell := range row.Cells {
			if cell.CallCount == 0 {
				fmt.Fprint(w, "—\t")
			} else {
				fmt.Fprintf(w, "%.1f%% (%d)\t", cell.Pct, cell.CallCount)
			}
		}
		fmt.Fprintln(w)
	}
	w.Flush()
	return b.String()
}

type userRow struct {
	UserName string
	Cells    []ShiftUserRecord // one per shift, in shifts order
}

func pivotToUserRows(shifts []string, records []ShiftUserRecord) []userRow {
	userMap := make(map[string][]ShiftUserRecord)
	userOrder := make([]string, 0)
	for _, rec := range records {
		if _, ok := userMap[rec.UserName]; !ok {
			userOrder = append(userOrder, rec.UserName)
		}
		userMap[rec.UserName] = append(userMap[rec.UserName], rec)
	}
	var rows []userRow
	for _, name := range userOrder {
		cells := make([]ShiftUserRecord, len(shifts))
		for i, shift := range shifts {
			cells[i] = ShiftUserRecord{UserName: name, Shift: shift}
			for _, rec := range userMap[name] {
				if rec.Shift == shift {
					cells[i] = rec
					break
				}
			}
		}
		rows = append(rows, userRow{UserName: name, Cells: cells})
	}
	return rows
}

// writeShiftCSV writes CSV output for all periods.
func writeShiftCSV(periods []PeriodResult) {
	w := csv.NewWriter(os.Stdout)
	// Determine all shift names across periods for consistent columns.
	shiftSet := make(map[string]bool)
	for _, pr := range periods {
		for _, shift := range pr.Shifts {
			shiftSet[shift] = true
		}
	}
	var allShifts []string
	for shift := range shiftSet {
		allShifts = append(allShifts, shift)
	}
	// Sort alphabetically for stable output.
	sortStrings(allShifts)

	// Header
	header := []string{"period", "period_start", "period_end", "total_calls", "user"}
	for _, shift := range allShifts {
		header = append(header, snakeCase(shift)+"_pct", snakeCase(shift)+"_calls")
	}
	w.Write(header)

	for _, pr := range periods {
		userRows := pivotToUserRows(pr.Shifts, pr.Records)
		for _, row := range userRows {
			line := []string{
				pr.Label,
				pr.Range.Start.Format("2006-01-02"),
				pr.Range.End.Format("2006-01-02"),
				fmt.Sprintf("%d", pr.TotalCalls),
				row.UserName,
			}
			for _, shift := range allShifts {
				found := false
				for _, cell := range row.Cells {
					if cell.Shift == shift {
						line = append(line, fmt.Sprintf("%.1f", cell.Pct), fmt.Sprintf("%d", cell.CallCount))
						found = true
						break
					}
				}
				if !found {
					line = append(line, "0.0", "0")
				}
			}
			w.Write(line)
		}
	}
	w.Flush()
}

func snakeCase(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", "_"))
}

func sortStrings(s []string) {
	// Simple insertion sort — no dependency needed.
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./cmd/shift-report/
```

Expected: `output.go` compiles.

- [ ] **Step 3: Commit**

```bash
git add cmd/shift-report/output.go
git commit -m "feat: add terminal table and CSV output for shift report"
```

---

### Task 5: Create html.go — HTML report generation

**Files:**
- Create: `cmd/shift-report/html.go`

- [ ] **Step 1: Write html.go**

Create `cmd/shift-report/html.go`:

```go
package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"
)

const shiftHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Shift Participation Report — {{.GeneratedAt.Format "January 2, 2006"}}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a2e; background: #f8f9fa; padding: 2rem; }
  header { text-align: center; margin-bottom: 2rem; padding-bottom: 1rem; border-bottom: 3px solid #c41e3a; }
  header h1 { font-size: 1.8rem; color: #c41e3a; }
  header p { color: #666; margin-top: 0.25rem; }
  section { background: #fff; border-radius: 8px; padding: 1.5rem; margin-bottom: 1.5rem; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
  section h2 { font-size: 1.25rem; color: #c41e3a; margin-bottom: 1rem; border-bottom: 2px solid #eee; padding-bottom: 0.5rem; }
  section p.summary { color: #666; margin-bottom: 0.75rem; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 0.5rem 0.75rem; border-bottom: 1px solid #eee; }
  th { font-weight: 600; color: #555; font-size: 0.85rem; text-transform: uppercase; }
  td { font-size: 0.95rem; }
  td.empty { color: #ccc; }
  footer { text-align: center; color: #999; font-size: 0.85rem; margin-top: 2rem; }
  @media print { body { padding: 0; } section { box-shadow: none; border: 1px solid #ddd; } }
</style>
</head>
<body>
<header>
  <h1>Shift Participation Report</h1>
  <p>Generated {{.GeneratedAt.Format "January 2, 2006"}}</p>
</header>

{{range .Periods}}
<section>
  <h2>{{.Label}} &mdash; {{.Range.Start.Format "Jan 2, 2006"}} to {{.Range.End.Format "Jan 2, 2006"}}</h2>
  <p class="summary">{{.TotalCalls}} total calls</p>
  {{if .TotalCalls}}
  <table>
    <tr>
      <th>User</th>
      {{range .Shifts}}
      <th>{{.}}</th>
      {{end}}
    </tr>
    {{range .UserRows}}
    <tr>
      <td>{{.UserName}}</td>
      {{range .Cells}}
      <td>{{if .CallCount}}{{printf "%.1f" .Pct}}% ({{.CallCount}}){{else}}<span class="empty">&mdash;</span>{{end}}</td>
      {{end}}
    </tr>
    {{end}}
  </table>
  {{end}}
</section>
{{end}}

<footer>
  Generated by Emergency Networking Shift Participation Report
</footer>
</body>
</html>`

// userRowTemplate holds a user and their cells for one period section.
type userRowTemplate struct {
	UserName string
	Cells    []ShiftUserRecord
}

// periodTemplate holds data for one period section of the HTML.
type periodTemplate struct {
	Label      string
	Range      struct{ Start, End time.Time }
	TotalCalls int
	Shifts     []string
	UserRows   []userRowTemplate
}

// htmlPageData is the top-level data for the HTML template.
type htmlPageData struct {
	GeneratedAt time.Time
	Periods     []periodTemplate
}

func saveShiftHTML(periods []PeriodResult) (string, error) {
	var pt []periodTemplate
	for _, pr := range periods {
		rows := pivotToUserRows(pr.Shifts, pr.Records)
		var urt []userRowTemplate
		for _, r := range rows {
			urt = append(urt, userRowTemplate{UserName: r.UserName, Cells: r.Cells})
		}
		pt = append(pt, periodTemplate{
			Label:      pr.Label,
			Range:      struct{ Start, End time.Time }{pr.Range.Start, pr.Range.End},
			TotalCalls: pr.TotalCalls,
			Shifts:     pr.Shifts,
			UserRows:   urt,
		})
	}
	data := htmlPageData{
		GeneratedAt: time.Now(),
		Periods:     pt,
	}

	tmpl, err := template.New("shift-report").Funcs(template.FuncMap{
		"displayName": func(name string) string {
			return strings.ReplaceAll(name, "_", " ")
		},
	}).Parse(shiftHTMLTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	filename := fmt.Sprintf("shift_participation_%s.html", time.Now().Format("2006-01-02_150405"))
	if err := os.WriteFile(filename, buf.Bytes(), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return filename, nil
}
```

This follows the existing `cmd/report/html.go` patterns: embedded template constant, same CSS, same `funcMap` for `displayName`, same section/card styling, same sanitize/save approach.

- [ ] **Step 2: Verify compilation**

```bash
go build ./cmd/shift-report/
```

Expected: `html.go` compiles.

- [ ] **Step 3: Commit**

```bash
git add cmd/shift-report/html.go
git commit -m "feat: add HTML output for shift participation report"
```

---

### Task 6: Create pdf.go — PDF report generation

**Files:**
- Create: `cmd/shift-report/pdf.go`

- [ ] **Step 1: Write pdf.go**

Create `cmd/shift-report/pdf.go`:

```go
package main

import (
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
)

func saveShiftPDF(periods []PeriodResult) (string, error) {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)

	pdf.AddPage()
	pageW, _ := pdf.GetPageSize()

	// Red header bar — matches existing report style
	pdf.SetFillColor(196, 30, 58)
	pdf.Rect(0, 0, pageW, 18, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetY(4)
	pdf.CellFormat(0, 9, "Shift Participation Report", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 8)
	pdf.CellFormat(0, 5, fmt.Sprintf("Generated %s", time.Now().Format("January 2, 2006")), "", 1, "C", false, 0, "")

	pdf.SetTextColor(26, 26, 46)
	pdf.SetY(24)

	for _, pr := range periods {
		// Section header
		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetTextColor(196, 30, 58)
		pdf.CellFormat(0, 8, fmt.Sprintf("%s (%s – %s)", pr.Label,
			pr.Range.Start.Format("Jan 2, 2006"),
			pr.Range.End.Format("Jan 2, 2006")), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(102, 102, 102)
		pdf.CellFormat(0, 6, fmt.Sprintf("%d total calls", pr.TotalCalls), "", 1, "L", false, 0, "")
		pdf.Ln(3)

		if pr.TotalCalls == 0 || len(pr.Records) == 0 {
			pdf.SetFont("Helvetica", "I", 10)
			pdf.SetTextColor(136, 136, 136)
			pdf.CellFormat(0, 6, "(no calls in period)", "", 1, "L", false, 0, "")
			pdf.Ln(4)
			continue
		}

		// Build table
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetTextColor(26, 26, 46)

		// Calculate column widths dynamically
		numCols := 1 + len(pr.Shifts)
		userColW := 50.0
		shiftColW := (pageW - 30 - userColW) / float64(len(pr.Shifts))
		if shiftColW > 40 {
			shiftColW = 40
		}

		// Header row
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(userColW, 7, "User", "1", 0, "L", true, 0, "")
		for _, shift := range pr.Shifts {
			pdf.CellFormat(shiftColW, 7, safeText(shift), "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)

		// Data rows
		pdf.SetFont("Helvetica", "", 8)
		userRows := pivotToUserRows(pr.Shifts, pr.Records)
		for _, row := range userRows {
			// Check for page break
			if pdf.GetY() > 250 {
				pdf.AddPage()
				// Repeat header on new page
				pdf.SetFillColor(240, 240, 240)
				pdf.SetFont("Helvetica", "B", 8)
				pdf.CellFormat(userColW, 7, "User", "1", 0, "L", true, 0, "")
				for _, shift := range pr.Shifts {
					pdf.CellFormat(shiftColW, 7, safeText(shift), "1", 0, "C", true, 0, "")
				}
				pdf.Ln(-1)
				pdf.SetFont("Helvetica", "", 8)
			}
			pdf.CellFormat(userColW, 6, safeText(row.UserName), "0", 0, "L", false, 0, "")
			for _, cell := range row.Cells {
				text := "—"
				if cell.CallCount > 0 {
					text = fmt.Sprintf("%.1f%% (%d)", cell.Pct, cell.CallCount)
				}
				pdf.CellFormat(shiftColW, 6, text, "0", 0, "C", false, 0, "")
			}
			pdf.Ln(-1)
			// Draw subtle row separator
			pdf.SetDrawColor(230, 230, 230)
			y := pdf.GetY()
			pdf.Line(15, y, pageW-15, y)
			pdf.SetY(y)
		}
		pdf.Ln(4)
	}

	// Footer
	pdf.SetTextColor(153, 153, 153)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.SetY(-20)
	pdf.CellFormat(0, 10, "Generated by Emergency Networking Shift Participation Report", "", 0, "C", false, 0, "")

	filename := fmt.Sprintf("shift_participation_%s.pdf", time.Now().Format("2006-01-02_150405"))
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return "", fmt.Errorf("write pdf: %w", err)
	}
	return filename, nil
}

// safeText replaces characters that gofpdf can't render.
func safeText(s string) string {
	// gofpdf uses a limited encoding; replace problematic chars.
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] < 128 {
			out = append(out, s[i])
		} else {
			out = append(out, '?')
		}
	}
	return string(out)
}
```

This follows the existing `cmd/report/pdf.go` patterns: same gofpdf usage, same red header bar styling, same auto-page-break logic, same shift-based table rendering.

- [ ] **Step 2: Verify compilation**

```bash
go build ./cmd/shift-report/
```

Expected: `pdf.go` compiles.

- [ ] **Step 3: Commit**

```bash
git add cmd/shift-report/pdf.go
git commit -m "feat: add PDF output for shift participation report"
```

---

### Task 7: Create main.go — entry point, flags, orchestration

**Files:**
- Create: `cmd/shift-report/main.go`

- [ ] **Step 1: Write main.go**

Create `cmd/shift-report/main.go`:

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dayvillefire/emergency-networking-reporting/enapi"
	"github.com/dayvillefire/emergency-networking-reporting/internal/shared"
)

func main() {
	csvOutput := flag.Bool("csv", false, "Output CSV instead of terminal table")
	htmlOutput := flag.Bool("html", false, "Save HTML report file")
	pdfOutput := flag.Bool("pdf", false, "Save PDF report file")
	deptFilter := flag.String("department", "", "Filter to specific department (station name)")
	flag.Parse()

	token := shared.ReadToken()
	if token == "" {
		fmt.Fprintln(os.Stderr, "API_TOKEN not found in .env or environment")
		os.Exit(1)
	}

	client := enapi.NewClient(
		enapi.WithToken(token),
		enapi.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
	)

	// In CSV-only mode, skip the TUI — fetch all users, include all by default,
	// fetch incidents, compute, and output CSV.
	if *csvOutput && !*htmlOutput && !*pdfOutput {
		runCSVMode(client, *deptFilter)
		return
	}

	// Launch TUI for interactive user selection.
	m := newSRModel(client, *htmlOutput, *pdfOutput, *csvOutput, *deptFilter)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runCSVMode(client *enapi.Client, deptFilter string) {
	// Fetch all users.
	users, err := client.ListUsers(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching users: %v\n", err)
		os.Exit(1)
	}

	// Build name map and default-select all active users.
	nameMap := make(map[string]string)
	selectedUsers := make(map[string]bool)
	for _, u := range users {
		if !u.Active {
			continue
		}
		display := u.LastName + ", " + u.FirstName
		nameMap[display] = display
		if u.PersonnelID != "" {
			nameMap[u.PersonnelID] = display
		}
		nameMap[fmt.Sprintf("%d", u.ID)] = display
		selectedUsers[display] = true
	}

	// Fetch incidents.
	fetchStart := shared.LastQuarter().Start
	fetchEnd := time.Now()
	incidents, err := shared.FetchIncidents(client, fetchStart, fetchEnd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching incidents: %v\n", err)
		os.Exit(1)
	}
	if deptFilter != "" {
		var filtered []shared.NormalizedIncident
		for _, inc := range incidents {
			if inc.Station == deptFilter {
				filtered = append(filtered, inc)
			}
		}
		incidents = filtered
	}

	// Compute three periods.
	periods := []struct {
		Label string
		Range shared.DateRange
	}{
		{"Previous Quarter", shared.LastQuarter()},
		{"Previous Month", shared.LastMonth()},
		{"Current Month", shared.CurrentMonth()},
	}

	var results []PeriodResult
	for _, p := range periods {
		subset := shared.FilterByDateRange(incidents, p.Range.Start, p.Range.End)
		shifts, records := ComputeShiftParticipation(subset, selectedUsers, nameMap)
		results = append(results, PeriodResult{
			Label:      p.Label,
			Range:      p.Range,
			TotalCalls: len(subset),
			Shifts:     shifts,
			Records:    records,
		})
	}

	writeShiftCSV(results)
}
```

- [ ] **Step 2: Verify full build**

```bash
go build -o en-shift-report ./cmd/shift-report/
```

Expected: binary `en-shift-report` is produced with no errors.

- [ ] **Step 3: Commit**

```bash
git add cmd/shift-report/main.go
git commit -m "feat: add shift report entry point with --csv headless mode"
```

---

### Task 8: Update Makefile

**Files:**
- Modify: `Makefile`

- [ ] **Step 1: Add build-shift-report target**

In `Makefile`, line 2 currently reads:
```makefile
RESPONDER_BINARY := en-responder-stats
```

Add after it:
```makefile
SHIFT_BINARY := en-shift-report
```

After the `build-responder` block (current line 14-15):
```makefile
build-responder:
	go build -o $(RESPONDER_BINARY) $(RESPONDER_PKG)
```

Add:
```makefile
SHIFT_PKG := ./cmd/shift-report

build-shift-report:
	go build -o $(SHIFT_BINARY) $(SHIFT_PKG)
```

And update `all` and `clean` to include the new binary:
- Change `all: build build-responder windows` → `all: build build-responder build-shift-report windows`
- Change `clean: rm -f ...` line to include `$(SHIFT_BINARY)`

Final Makefile:

```makefile
BINARY := en-report
RESPONDER_BINARY := en-responder-stats
SHIFT_BINARY := en-shift-report
WINDOWS_BINARY := $(BINARY).exe
PKG := ./cmd/report
RESPONDER_PKG := ./cmd/responder-stats
SHIFT_PKG := ./cmd/shift-report

.PHONY: all build build-responder build-shift-report windows clean

all: build build-responder build-shift-report windows

build:
	go build -o $(BINARY) $(PKG)

build-responder:
	go build -o $(RESPONDER_BINARY) $(RESPONDER_PKG)

build-shift-report:
	go build -o $(SHIFT_BINARY) $(SHIFT_PKG)

windows:
	GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_BINARY) $(PKG)

clean:
	rm -f $(BINARY) $(RESPONDER_BINARY) $(SHIFT_BINARY) $(WINDOWS_BINARY)
```

- [ ] **Step 2: Verify build via Make**

```bash
make build-shift-report
```

Expected: binary `en-shift-report` is produced with no errors.

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "build: add build-shift-report target to Makefile"
```

---

## Verification

After all tasks are complete, run these checks:

1. **Compilation:**
   ```bash
   go build ./...
   ```
   Expected: all packages compile.

2. **Build via Make:**
   ```bash
   make build-shift-report
   ./en-shift-report --help
   ```
   Expected: binary runs, shows flag usage.

3. **Full build:**
   ```bash
   make
   ```
   Expected: all binaries produced including `en-shift-report`.

4. **Interactive flow:** Run `./en-shift-report`. Verify: loading spinner → user picker appears with real names → space toggles selection → enter confirms → fetch spinner → results display.

5. **CSV headless mode:** Run `./en-shift-report --csv`. Verify: CSV prints to stdout with correct columns, no TUI launches.

6. **HTML output:** Run `./en-shift-report --html`. Select users, confirm. Verify: HTML file created, opens in browser with styled tables.

7. **PDF output:** Run `./en-shift-report --pdf`. Select users, confirm. Verify: PDF file created with proper formatting.

8. **Department filter:** Run `./en-shift-report --department "Station Name"`. Verify: incident set is filtered.

9. **Edge case — no users selected:** Press enter with all users deselected. Verify: stays on picker, doesn't crash.

10. **Edge case — no API token:** Unset `API_TOKEN`, run tool. Verify: exits 1 with clear message.
