package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/dayvillefire/emergency-networking-reporting/enapi"
	"github.com/dayvillefire/emergency-networking-reporting/internal/shared"
)

type periodResult struct {
	Label      string
	Range      shared.DateRange
	TotalCalls int
	Summary    shared.PersonnelResponseSummary
}

var (
	csvOutput  = flag.Bool("csv", false, "Output CSV instead of text table")
	deptFilter = flag.String("department", "", "Filter to specific department (station name)")
	startDate  = flag.String("start", "", "Start date (YYYY-MM-DD)")
	endDate    = flag.String("end", "", "End date (YYYY-MM-DD)")
)

func main() {
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

	nameMap := shared.BuildNameMap(client)

	// Determine fetch range and periods from CLI flags.
	var (
		fetchStart, fetchEnd time.Time
		periods              []struct {
			Label string
			Range shared.DateRange
		}
	)

	switch {
	case *startDate != "" && *endDate != "":
		s, err := time.Parse("2006-01-02", *startDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid -start date %q: %v\n", *startDate, err)
			os.Exit(1)
		}
		e, err := time.Parse("2006-01-02", *endDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid -end date %q: %v\n", *endDate, err)
			os.Exit(1)
		}
		if e.Before(s) {
			fmt.Fprintf(os.Stderr, "-end date %s must not be before -start date %s\n", *endDate, *startDate)
			os.Exit(1)
		}
		fetchStart, fetchEnd = s, e
		periods = []struct {
			Label string
			Range shared.DateRange
		}{
			{"Custom Range", shared.DateRange{Start: s, End: e}},
		}

	case *startDate != "":
		s, err := time.Parse("2006-01-02", *startDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid -start date %q: %v\n", *startDate, err)
			os.Exit(1)
		}
		now := time.Now()
		fetchStart, fetchEnd = s, now
		periods = []struct {
			Label string
			Range shared.DateRange
		}{
			{"Custom Range", shared.DateRange{Start: s, End: now}},
		}

	case *endDate != "":
		e, err := time.Parse("2006-01-02", *endDate)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid -end date %q: %v\n", *endDate, err)
			os.Exit(1)
		}
		s := e.AddDate(-1, 0, 0)
		fetchStart, fetchEnd = s, e
		periods = []struct {
			Label string
			Range shared.DateRange
		}{
			{"Custom Range", shared.DateRange{Start: s, End: e}},
		}

	default:
		fetchStart = shared.LastYear().Start
		fetchEnd = shared.LastYear().End
		periods = []struct {
			Label string
			Range shared.DateRange
		}{
			{"Last Month", shared.LastMonth()},
			{"Last Quarter", shared.LastQuarter()},
			{"Last Year", shared.LastYear()},
		}
	}

	allIncidents, err := shared.FetchIncidents(client, fetchStart, fetchEnd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching incidents: %v\n", err)
		os.Exit(1)
	}

	// Apply department filter if specified.
	if *deptFilter != "" {
		var filtered []shared.NormalizedIncident
		for _, inc := range allIncidents {
			if strings.EqualFold(inc.Station, *deptFilter) {
				filtered = append(filtered, inc)
			}
		}
		allIncidents = filtered
	}

	var results []periodResult
	for _, p := range periods {
		subset := shared.FilterByDateRange(allIncidents, p.Range.Start, p.Range.End)
		summary := shared.ComputePersonnelResponse(subset, nameMap)
		results = append(results, periodResult{
			Label:      p.Label,
			Range:      p.Range,
			TotalCalls: len(subset),
			Summary:    summary,
		})
	}

	if *csvOutput {
		writeCSV(results)
	} else {
		writeTextTable(results)
	}
}

func writeTextTable(results []periodResult) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, r := range results {
		fmt.Fprintf(w, "%s  (%s – %s)\tTotal Calls: %d\n",
			r.Label,
			r.Range.Start.Format("2006-01-02"),
			r.Range.End.Format("2006-01-02"),
			r.TotalCalls,
		)
		if r.TotalCalls == 0 || (len(r.Summary.High) == 0 && len(r.Summary.Active) == 0 && len(r.Summary.GoodStanding) == 0) {
			fmt.Fprintf(w, "  (no responding members)\n\n")
			continue
		}
		fmt.Fprintf(w, "  Tier\tName\tCalls\tPct\n")
		fmt.Fprintf(w, "  ───\t────\t─────\t───\n")

		writeTier(w, r.Summary.High, ">=30%", "High")
		writeTier(w, r.Summary.Active, ">=20%", "Active")
		writeTier(w, r.Summary.GoodStanding, ">=10%", "Good Standing")
		fmt.Fprintln(w)
	}
	w.Flush()
}

func writeTier(w *tabwriter.Writer, records []shared.PersonnelRecord, threshold, label string) {
	if len(records) == 0 {
		return
	}
	for i, r := range records {
		tierLabel := threshold
		tierName := label
		if i > 0 {
			tierLabel = ""
			tierName = ""
		}
		fmt.Fprintf(w, "  %s\t%s\t%s\t%d\t%.1f%%\n",
			tierLabel, tierName, r.Name, r.TotalCalls, r.Pct)
	}
}

func writeCSV(results []periodResult) {
	w := csv.NewWriter(os.Stdout)
	w.Write([]string{"period", "period_start", "period_end", "total_calls", "tier", "tier_label", "name", "calls", "pct"})

	for _, r := range results {
		writeCSVTier(w, r, r.Summary.High, 30, "High")
		writeCSVTier(w, r, r.Summary.Active, 20, "Active")
		writeCSVTier(w, r, r.Summary.GoodStanding, 10, "Good Standing")
	}
	w.Flush()
}

func writeCSVTier(w *csv.Writer, pr periodResult, records []shared.PersonnelRecord, tier int, label string) {
	for _, rec := range records {
		w.Write([]string{
			pr.Label,
			pr.Range.Start.Format("2006-01-02"),
			pr.Range.End.Format("2006-01-02"),
			fmt.Sprintf("%d", pr.TotalCalls),
			fmt.Sprintf("%d", tier),
			label,
			rec.Name,
			fmt.Sprintf("%d", rec.TotalCalls),
			fmt.Sprintf("%.1f", rec.Pct),
		})
	}
}
