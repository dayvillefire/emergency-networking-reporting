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
)

type reportState int

const (
	stateStart reportState = iota
	stateFetchingStations
	stateSelectingDept
	stateSelectingYear
	stateSelectingScope
	stateFetchingIncidents
	stateProcessing
	stateDone
	stateError
)

type stationListMsg struct {
	stations []string
	err      error
}

type fetchedPageMsg struct {
	incidents []enapi.NerisIncident
	page      int
	done      bool
	err       error
}

type model struct {
	client *enapi.Client
	now    time.Time

	state   reportState
	spinner spinner.Model
	err     error

	// Station data (lightweight fetch)
	stations       []string
	cursor         int
	selectedDept   string

	// Period selection
	availableYears []int
	selectedYear   int
	yearCursor     int
	scopeCursor    int
	scopes         []string
	selectedScope  string
	periodStart    time.Time
	periodEnd      time.Time

	// Full incident fetch
	allIncidents     []enapi.NerisIncident
	filteredIncidents []enapi.NerisIncident
	currentPage      int

	// Results
	stats       *ReportStats
	periodLabel string
	htmlPath    string
}

var (
	bannerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#c41e3a")).
		Padding(1, 4).
		Align(lipgloss.Center)

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#c41e3a")).MarginBottom(1)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#c41e3a")).Bold(true).PaddingLeft(2)
	unselected    = lipgloss.NewStyle().PaddingLeft(2)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888")).MarginTop(1)
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#2ecc71")).Bold(true)
)

var months = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

func newModel(client *enapi.Client, now time.Time) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#c41e3a"))

	// Available years: current year and previous 5 years
	years := make([]int, 0)
	for y := now.Year(); y >= now.Year()-5; y-- {
		years = append(years, y)
	}

	return model{
		client:        client,
		now:           now,
		state:         stateStart,
		spinner:       s,
		selectedYear:  now.Year(),
		availableYears: years,
	}
}

// init creates the model and runs the station fetch immediately.
func (m model) Init() tea.Cmd {
	return fetchStations(m.client)
}

func fetchStations(client *enapi.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Lightweight fetch: one page to get station names
		resp, err := client.ListNerisIncidents(ctx, enapi.VWithPerPage(100), enapi.VWithPage(1))
		if err != nil {
			return stationListMsg{err: err}
		}

		stationSet := make(map[string]bool)
		for _, inc := range resp.Data {
			st := string(inc.IncidentStation)
			if st != "" {
				stationSet[st] = true
			}
		}
		var stations []string
		for st := range stationSet {
			stations = append(stations, st)
		}
		sort.Strings(stations)
		// "All Departments" is always first
		stations = append([]string{"All Departments"}, stations...)
		return stationListMsg{stations: stations}
	}
}

func fetchIncidentPage(client *enapi.Client, start, end time.Time, page int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		resp, err := client.ListNerisIncidents(ctx, enapi.VWithPerPage(100), enapi.VWithPage(page))
		if err != nil {
			return fetchedPageMsg{err: err}
		}
		// Filter by period
		var filtered []enapi.NerisIncident
		for _, inc := range resp.Data {
			t := inc.IncidentPsapTime.Time
			if !t.IsZero() && (t.Equal(start) || t.After(start)) && t.Before(end.Add(24*time.Hour)) {
				filtered = append(filtered, inc)
			}
		}
		done := len(resp.Data) < 100 || page*100 >= resp.Total
		return fetchedPageMsg{incidents: filtered, page: page, done: done}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if m.state == stateFetchingStations || m.state == stateFetchingIncidents || m.state == stateProcessing {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.state == stateStart {
			// Any key dismisses the banner
			m.state = stateFetchingStations
			return m, m.spinner.Tick
		}
		switch m.state {
		case stateSelectingDept:
			return m.handleDeptKey(msg)
		case stateSelectingYear:
			return m.handleYearKey(msg)
		case stateSelectingScope:
			return m.handleScopeKey(msg)
		case stateDone, stateError:
			if msg.String() == "enter" || msg.String() == "q" {
				return m, tea.Quit
			}
		}

	case stationListMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
			return m, nil
		}
		m.stations = msg.stations
		m.state = stateSelectingDept
		return m, nil

	case fetchedPageMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
			return m, nil
		}
		m.allIncidents = append(m.allIncidents, msg.incidents...)
		if !msg.done {
			return m, fetchIncidentPage(m.client, m.periodStart, m.periodEnd, msg.page+1)
		}
		// Done fetching — filter and proceed
		m.filteredIncidents = m.allIncidents
		m.state = stateProcessing
		return m, processStats(m)

	case statsDoneMsg:
		m.stats = msg.stats
		m.periodLabel = msg.periodLabel
		m.htmlPath = msg.htmlPath
		m.state = stateDone
		return m, nil

	case statsErrMsg:
		m.err = msg.err
		m.state = stateError
		return m, nil
	}
	return m, nil
}

// ---- Key Handlers ----

func (m model) handleDeptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 { m.cursor-- }
	case "down", "j":
		if m.cursor < len(m.stations)-1 { m.cursor++ }
	case "enter":
		m.selectedDept = m.stations[m.cursor]
		if m.selectedDept == "All Departments" {
			// Will be resolved later from incident data
		}
		m.state = stateSelectingYear
	}
	return m, nil
}

func (m model) handleYearKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.yearCursor > 0 { m.yearCursor-- }
	case "down", "j":
		if m.yearCursor < len(m.availableYears)-1 { m.yearCursor++ }
	case "enter":
		m.selectedYear = m.availableYears[m.yearCursor]
		m.scopes = []string{"Full Year", "Q1", "Q2", "Q3", "Q4"}
		m.scopes = append(m.scopes, months...)
		m.scopeCursor = 0
		m.state = stateSelectingScope
	}
	return m, nil
}

func (m model) handleScopeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.scopeCursor > 0 { m.scopeCursor-- }
	case "down", "j":
		if m.scopeCursor < len(m.scopes)-1 { m.scopeCursor++ }
	case "enter":
		m.selectedScope = m.scopes[m.scopeCursor]
		m.periodStart, m.periodEnd = computePeriodRange(m.selectedYear, m.selectedScope)
		m.state = stateFetchingIncidents
		m.allIncidents = nil
		cmds := []tea.Cmd{m.spinner.Tick, fetchIncidentPage(m.client, m.periodStart, m.periodEnd, 1)}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func computePeriodRange(year int, scope string) (time.Time, time.Time) {
	switch scope {
	case "Full Year":
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
	case "Q1":
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, 3, 31, 23, 59, 59, 0, time.UTC)
	case "Q2":
		return time.Date(year, 4, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, 6, 30, 23, 59, 59, 0, time.UTC)
	case "Q3":
		return time.Date(year, 7, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, 9, 30, 23, 59, 59, 0, time.UTC)
	case "Q4":
		return time.Date(year, 10, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
	default:
		for i, name := range months {
			if scope == name {
				mo := time.Month(i + 1)
				return time.Date(year, mo, 1, 0, 0, 0, 0, time.UTC),
					time.Date(year, mo+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)
			}
		}
	}
	// Fallback: full year
	return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
}

// ---- Stats processing ----

type statsDoneMsg struct {
	stats       *ReportStats
	periodLabel string
	htmlPath    string
}
type statsErrMsg struct{ err error }

func processStats(m model) tea.Cmd {
	return func() tea.Msg {
		deptName := m.selectedDept
		if deptName == "All Departments" {
			counts := make(map[string]int)
			for _, inc := range m.filteredIncidents {
				st := string(inc.IncidentStation)
				if st != "" { counts[st]++ }
			}
			maxCount := 0
			majority := "Fire Department"
			for st, c := range counts {
				if c > maxCount { maxCount = c; majority = st }
			}
			deptName = majority
		}
		periodLabel := periodString(m.periodStart, m.periodEnd)
		stats := ComputeStats(m.filteredIncidents, deptName, m.periodStart, m.periodEnd)
		htmlPath, err := SaveHTML(stats, periodLabel)
		if err != nil {
			return statsErrMsg{err}
		}
		return statsDoneMsg{stats: stats, periodLabel: periodLabel, htmlPath: htmlPath}
	}
}

// ---- Views ----

func (m model) View() string {
	switch m.state {
	case stateStart:
		return m.viewBanner()

	case stateFetchingStations:
		return center(fmt.Sprintf("\n\n  %s Loading stations...\n\n", m.spinner.View()))

	case stateSelectingDept:
		return m.viewPicker("Select Department", m.stations, m.cursor)

	case stateSelectingYear:
		yearStrs := make([]string, len(m.availableYears))
		for i, y := range m.availableYears {
			yearStrs[i] = fmt.Sprintf("%d", y)
		}
		return m.viewPicker(fmt.Sprintf("Department: %s — Select Year", m.selectedDept), yearStrs, m.yearCursor)

	case stateSelectingScope:
		label := fmt.Sprintf("Department: %s — Year: %d — Select Scope", m.selectedDept, m.selectedYear)
		return m.viewPicker(label, m.scopes, m.scopeCursor)

	case stateFetchingIncidents:
		return center(fmt.Sprintf("\n\n  %s Fetching incidents for %s...\n  %d loaded so far\n\n",
			m.spinner.View(), periodString(m.periodStart, m.periodEnd), len(m.allIncidents)))

	case stateProcessing:
		return center(fmt.Sprintf("\n\n  %s Computing statistics...\n\n", m.spinner.View()))

	case stateError:
		return center(fmt.Sprintf("\n  Error: %v\n\n  Press q to quit.\n", m.err))

	case stateDone:
		return m.viewDone()
	}
	return ""
}

func (m model) viewBanner() string {
	var b strings.Builder
	b.WriteString("\n\n")
	b.WriteString(bannerStyle.Render(" EMERGENCY NETWORKING "))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Render(
		lipgloss.NewStyle().Foreground(lipgloss.Color("#666")).Render("Department Reporting Tool"),
	))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Align(lipgloss.Center).Render(
		helpStyle.Render("Press any key to begin"),
	))
	return b.String()
}

func (m model) viewPicker(header string, items []string, cursor int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n")

	start := 0
	if cursor > 5 { start = cursor - 5 }
	end := start + 12
	if end > len(items) { end = len(items) }

	if start > 0 {
		b.WriteString(fmt.Sprintf("  ↑ %d more...\n", start))
	}
	for i := start; i < end; i++ {
		if i == cursor {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", items[i])))
		} else {
			b.WriteString(unselected.Render(fmt.Sprintf("  %s", items[i])))
		}
		b.WriteString("\n")
	}
	if end < len(items) {
		b.WriteString(fmt.Sprintf("  ↓ %d more...\n", len(items)-end))
	}
	b.WriteString(helpStyle.Render("\n  ↑↓/jk navigate  ·  enter select  ·  ctrl+c quit"))
	return b.String()
}

func (m model) viewDone() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Report: %s", m.stats.DeptName)))
	b.WriteString(fmt.Sprintf("  Period: %s  ·  Total Calls: %d\n\n", m.periodLabel, m.stats.TotalCalls))

	b.WriteString(fmt.Sprintf("  EMS:  %d (%.1f%%)", m.stats.EMSCount, m.stats.EMSPct))
	b.WriteString(fmt.Sprintf("    Fire: %d (%.1f%%)\n", m.stats.FireCount, m.stats.FirePct))
	b.WriteString(fmt.Sprintf("  Avg Personnel: %.1f    Concurrent: %d (%.1f%%)\n",
		m.stats.AvgPersonnel, m.stats.ConcurrentCount, m.stats.ConcurrentPct))
	b.WriteString(fmt.Sprintf("  Mutual Aid — Given: %d (%.1f%%)  Received: %d (%.1f%%)\n",
		m.stats.MutualAidGiven, m.stats.MutualAidGivenPct, m.stats.MutualAidReceived, m.stats.MutualAidReceivedPct))
	b.WriteString(fmt.Sprintf("  Special — Hazmat: %d  MVA: %d  CO: %d\n",
		m.stats.HazmatCount, m.stats.MVACount, m.stats.COCount))
	b.WriteString(fmt.Sprintf("  Special Units — Drone: %d  Rehab: %d\n",
		m.stats.DroneTeamCount, m.stats.RehabTeamCount))

	b.WriteString(fmt.Sprintf("\n  %s\n\n", successStyle.Render(fmt.Sprintf("HTML report saved: %s", m.htmlPath))))
	b.WriteString(helpStyle.Render("  Press q to quit"))
	return b.String()
}

func center(s string) string {
	return "\n" + s
}

func periodString(start, end time.Time) string {
	month := start.Month()
	if month >= 1 && month <= 3 && start.Day() == 1 && end.Month() == 3 {
		return fmt.Sprintf("Q1_%d", start.Year())
	}
	if month >= 4 && month <= 6 && start.Day() == 1 && end.Month() == 6 {
		return fmt.Sprintf("Q2_%d", start.Year())
	}
	if month >= 7 && month <= 9 && start.Day() == 1 && end.Month() == 9 {
		return fmt.Sprintf("Q3_%d", start.Year())
	}
	if month >= 10 && month <= 12 && start.Day() == 1 && end.Month() == 12 {
		return fmt.Sprintf("Q4_%d", start.Year())
	}
	if start.Month() == 1 && start.Day() == 1 && end.Month() == 12 {
		return fmt.Sprintf("%d", start.Year())
	}
	return fmt.Sprintf("%s_%d", start.Month().String(), start.Year())
}
