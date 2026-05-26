package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dayvillefire/emergency-networking-reporting/enapi"
	"github.com/dayvillefire/emergency-networking-reporting/internal/shared"
)

type reportState int

const (
	stateStart reportState = iota
	stateFetchingStations
	stateSelectingDept
	stateSelectingYear
	stateSelectingScope
	stateSelectingOptions
	stateFetchingIncidents
	stateProcessing
	stateDone
	stateError
)

type stationListMsg struct {
	stations []string
	err      error
}

type fetchProgressMsg struct {
	nerisPages   int
	nerisCount   int
	nerisElapsed time.Duration
	incPages     int
	incCount     int
	incElapsed   time.Duration
	done         bool
	incidents    []shared.NormalizedIncident
	err          error
}

type model struct {
	client *enapi.Client
	now    time.Time

	state   reportState
	spinner spinner.Model
	err     error

	stations     []string
	cursor       int
	selectedDept string

	availableYears []int
	selectedYear   int
	yearCursor     int
	scopeCursor    int
	scopes         []string
	selectedScope         string
	showPersonnelDetails  bool
	showCharts            bool
	optionsCursor         int
	periodStart           time.Time
	periodEnd             time.Time

	nerisPages   int
	nerisCount   int
	nerisElapsed time.Duration
	incPages     int
	incCount     int
	incElapsed   time.Duration
	progressCh   chan fetchProgressMsg
	nameMap      map[string]string

	filteredIncidents []shared.NormalizedIncident

	stats       *ReportStats
	periodLabel string
	htmlPath    string
	pdfPath     string
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

func newModel(client *enapi.Client, now time.Time, nameMap map[string]string) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#c41e3a"))
	years := make([]int, 0)
	for y := now.Year(); y >= now.Year()-5; y-- {
		years = append(years, y)
	}
	return model{
		client:         client,
		now:            now,
		state:          stateStart,
		spinner:        s,
		selectedYear:          now.Year(),
		availableYears:        years,
		nameMap:               nameMap,
		showPersonnelDetails:  true,
		showCharts:            true,
		optionsCursor:         0,
	}
}

func (m model) Init() tea.Cmd { return fetchStations(m.client) }

func fetchStations(client *enapi.Client) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		stationSet := make(map[string]bool)
		resp, err := client.ListNerisIncidents(ctx, enapi.VWithPerPage(100), enapi.VWithPage(1))
		if err == nil {
			for _, inc := range resp.Data {
				if st := string(inc.IncidentStation); st != "" {
					stationSet[st] = true
				}
			}
		}
		resp2, err2 := client.ListIncidents(ctx, enapi.VWithPerPage(100), enapi.VWithPage(1))
		if err2 == nil {
			for _, inc := range resp2.Data {
				if inc.Station != "" {
					stationSet[inc.Station] = true
				}
			}
		}
		if len(stationSet) == 0 && err != nil && err2 != nil {
			return stationListMsg{err: err}
		}
		var stations []string
		for st := range stationSet {
			stations = append(stations, st)
		}
		sort.Strings(stations)
		stations = append([]string{"All Departments"}, stations...)
		return stationListMsg{stations: stations}
	}
}

func fetchAllIncidents(client *enapi.Client, start, end time.Time) (chan fetchProgressMsg, tea.Cmd) {
	ch := make(chan fetchProgressMsg, 256)
	cmd := func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
		defer cancel()
		var mu sync.Mutex
		var nerisIncidents, incIncidents []shared.NormalizedIncident
		var nerisErr, incErr error
		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			startT := time.Now()
			for page := 1; ; page++ {
				resp, err := client.ListNerisIncidents(ctx, enapi.VWithPerPage(100), enapi.VWithPage(page))
				if err != nil {
					nerisErr = err
					return
				}
				for _, inc := range resp.Data {
					t := inc.IncidentPsapTime.Time
					if !t.IsZero() && (t.Equal(start) || t.After(start)) && t.Before(end.Add(24*time.Hour)) {
						mu.Lock()
						nerisIncidents = append(nerisIncidents, shared.NormNerisIncident(inc))
						mu.Unlock()
					}
				}
				select {
				case ch <- fetchProgressMsg{nerisPages: page, nerisCount: len(nerisIncidents), nerisElapsed: time.Since(startT)}:
				default:
				}
				if len(resp.Data) < 100 || page*100 >= resp.Total {
					break
				}
			}
		}()

		go func() {
			defer wg.Done()
			startT := time.Now()
			zeroStreak := 0
			for page := 1; ; page++ {
				resp, err := client.ListIncidents(ctx, enapi.VWithPerPage(100), enapi.VWithPage(page))
				if err != nil {
					incErr = err
					return
				}
				var count int
				for _, inc := range resp.Data {
					if inc.IncidentType == "" {
						continue
					}
					t := inc.Psap.Time
					if !t.IsZero() && (t.Equal(start) || t.After(start)) && t.Before(end.Add(24*time.Hour)) {
						mu.Lock()
						incIncidents = append(incIncidents, shared.NormIncident(inc))
						count++
						mu.Unlock()
					}
				}
				select {
				case ch <- fetchProgressMsg{incPages: page, incCount: len(incIncidents), incElapsed: time.Since(startT)}:
				default:
				}
				if len(resp.Data) < 100 || page*100 >= resp.Total {
					break
				}
				if count == 0 {
					zeroStreak++
					if zeroStreak >= 5 {
						break
					}
				} else {
					zeroStreak = 0
				}
			}
		}()

		wg.Wait()
		close(ch)
		if nerisErr != nil {
			return fetchProgressMsg{done: true, err: nerisErr}
		}
		if incErr != nil {
			return fetchProgressMsg{done: true, err: incErr}
		}
		return fetchProgressMsg{
			done:       true,
			incidents:  append(nerisIncidents, incIncidents...),
			nerisCount: len(nerisIncidents),
			incCount:   len(incIncidents),
		}
	}
	return ch, cmd
}

type progressTickMsg struct{}

func listenProgress(ch chan fetchProgressMsg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return progressTickMsg{}
		}
		return msg
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
		case stateSelectingOptions:
			return m.handleOptionsKey(msg)
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

	case progressTickMsg:
		return m, nil

	case fetchProgressMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = stateError
			return m, nil
		}
		if msg.done {
			m.filteredIncidents = msg.incidents
			m.state = stateProcessing
			return m, processStats(m)
		}
		m.nerisPages = msg.nerisPages
		m.nerisCount = msg.nerisCount
		m.nerisElapsed = msg.nerisElapsed
		m.incPages = msg.incPages
		m.incCount = msg.incCount
		m.incElapsed = msg.incElapsed
		return m, listenProgress(m.progressCh)

	case statsDoneMsg:
		m.stats = msg.stats
		m.periodLabel = msg.periodLabel
		m.htmlPath = msg.htmlPath
		m.pdfPath = msg.pdfPath
		m.state = stateDone
		return m, nil

	case statsErrMsg:
		m.err = msg.err
		m.state = stateError
		return m, nil
	}
	return m, nil
}

func (m model) handleDeptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 { m.cursor-- }
	case "down", "j":
		if m.cursor < len(m.stations)-1 { m.cursor++ }
	case "enter":
		m.selectedDept = m.stations[m.cursor]
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
		m.scopes = []string{"Full Year", "Fiscal Year", "Q1", "Q2", "Q3", "Q4"}
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
		m.state = stateSelectingOptions
		var fetchCmd tea.Cmd
		m.progressCh, fetchCmd = fetchAllIncidents(m.client, m.periodStart, m.periodEnd)
		return m, tea.Batch(m.spinner.Tick, fetchCmd, listenProgress(m.progressCh))
	}
	return m, nil
}

func (m model) handleOptionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.optionsCursor > 0 {
			m.optionsCursor--
		}
	case "down", "j":
		if m.optionsCursor < 1 {
			m.optionsCursor++
		}
	case " ":
		switch m.optionsCursor {
		case 0:
			m.showPersonnelDetails = !m.showPersonnelDetails
		case 1:
			m.showCharts = !m.showCharts
		}
	case "enter":
		m.state = stateFetchingIncidents
		var fetchCmd tea.Cmd
		m.progressCh, fetchCmd = fetchAllIncidents(m.client, m.periodStart, m.periodEnd)
		return m, tea.Batch(m.spinner.Tick, fetchCmd, listenProgress(m.progressCh))
	}
	return m, nil
}

func computePeriodRange(year int, scope string) (time.Time, time.Time) {
	switch scope {
	case "Full Year":
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
	case "Fiscal Year":
		return time.Date(year-1, 5, 1, 0, 0, 0, 0, time.UTC),
			time.Date(year, 4, 30, 23, 59, 59, 0, time.UTC)
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
	return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(year, 12, 31, 23, 59, 59, 0, time.UTC)
}

type statsDoneMsg struct {
	stats                *ReportStats
	periodLabel          string
	htmlPath             string
	pdfPath              string
	showPersonnelDetails bool
}
type statsErrMsg struct{ err error }

func processStats(m model) tea.Cmd {
	return func() tea.Msg {
		deptName := m.selectedDept
		if deptName == "All Departments" {
			counts := make(map[string]int)
			for _, inc := range m.filteredIncidents {
				if inc.Station != "" { counts[inc.Station]++ }
			}
			maxCount := 0
			majority := "Fire Department"
			for st, c := range counts {
				if c > maxCount { maxCount = c; majority = st }
			}
			deptName = majority
		}
		periodLabel := periodString(m.periodStart, m.periodEnd)
		showCharts := m.showCharts
		stats := ComputeStats(m.filteredIncidents, deptName, m.periodStart, m.periodEnd, m.nameMap)
		htmlPath, err := SaveHTML(stats, periodLabel, showCharts, m.showPersonnelDetails)
		if err != nil {
			return statsErrMsg{err}
		}
		var pdfPath string
		// PDF is always generated; personnel details toggle is respected in the PDF too
		pdfPath, _ = SavePDF(stats, periodLabel, showCharts, m.showPersonnelDetails)
		return statsDoneMsg{
			stats: stats, periodLabel: periodLabel,
			htmlPath: htmlPath, pdfPath: pdfPath,
			showPersonnelDetails: m.showPersonnelDetails,
		}
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
	case stateSelectingOptions:
		return m.viewOptions()
	case stateFetchingIncidents:
		return m.viewFetchProgress()
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

func (m model) viewFetchProgress() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("Fetching incidents: %s", periodString(m.periodStart, m.periodEnd))))
	b.WriteString("\n\n")
	if m.nerisPages > 0 {
		b.WriteString(fmt.Sprintf("  %s NERIS:  page %d  |  %d matched  |  %s elapsed  |  batch: 100\n",
			m.spinner.View(), m.nerisPages, m.nerisCount, m.nerisElapsed.Round(time.Second)))
	}
	if m.incPages > 0 {
		b.WriteString(fmt.Sprintf("  %s NFIRS:  page %d  |  %d matched  |  %s elapsed  |  batch: 100\n",
			m.spinner.View(), m.incPages, m.incCount, m.incElapsed.Round(time.Second)))
	}
	if m.nerisPages == 0 && m.incPages == 0 {
		b.WriteString(fmt.Sprintf("  %s Starting...\n", m.spinner.View()))
	}
	b.WriteString(helpStyle.Render("\n  Fetching data from both NERIS and NFIRS sources..."))
	return b.String()
}

func (m model) viewOptions() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Report Options"))
	b.WriteString("\n\n")

	type optDef struct {
		label string
		value bool
	}
	opts := []optDef{
		{"Show personnel details (member lists)", m.showPersonnelDetails},
		{"Show charts/graphs (monthly trends)", m.showCharts},
	}

	for i, opt := range opts {
		check := "[ ]"
		if opt.value {
			check = "[x]"
		}
		line := fmt.Sprintf("%s %s", check, opt.label)
		if i == m.optionsCursor {
			b.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", line)))
		} else {
			b.WriteString(unselected.Render(fmt.Sprintf("  %s", line)))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  ↑↓ navigate  ·  space toggle  ·  enter continue  ·  ctrl+c quit"))
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
	b.WriteString(fmt.Sprintf("  Personnel — %d total, High: %d, Active: %d, Good Standing: %d\n",
		m.stats.PersonnelTotalResponding, m.stats.PersonnelHighCount, m.stats.PersonnelActiveCount, m.stats.PersonnelGoodStandingCount))
	b.WriteString(fmt.Sprintf("  Special Units — Drone: %d  Rehab: %d\n",
		m.stats.DroneTeamCount, m.stats.RehabTeamCount))
	if m.pdfPath != "" {
		b.WriteString(fmt.Sprintf("\n  %s\n", successStyle.Render(fmt.Sprintf("PDF report saved:  %s", m.pdfPath))))
	}
	b.WriteString(fmt.Sprintf("  %s\n\n", successStyle.Render(fmt.Sprintf("HTML report saved: %s", m.htmlPath))))
	b.WriteString(helpStyle.Render("  Press q to quit"))
	return b.String()
}

func center(s string) string { return "\n" + s }

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
	if start.Month() == 5 && start.Day() == 1 && end.Month() == 4 && end.Day() == 30 {
		return fmt.Sprintf("FY_%d", end.Year())
	}
	if start.Month() == 1 && start.Day() == 1 && end.Month() == 12 {
		return fmt.Sprintf("%d", start.Year())
	}
	return fmt.Sprintf("%s_%d", start.Month().String(), start.Year())
}
