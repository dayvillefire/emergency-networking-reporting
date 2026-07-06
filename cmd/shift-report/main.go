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
