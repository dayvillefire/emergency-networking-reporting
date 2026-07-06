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
	allUsers      []string         // all names in "Last, First" format
	selected      map[int]bool     // index -> selected
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
