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

	// Two header rows: shift name (spanning O/S+N/S), then O/S / N/S labels.
	fmt.Fprint(w, "\t")
	for _, shift := range pr.Shifts {
		fmt.Fprintf(w, "%s\t\t", shift)
	}
	fmt.Fprintln(w)
	fmt.Fprint(w, "\t")
	for range pr.Shifts {
		fmt.Fprint(w, "O/S\tN/S\t")
	}
	fmt.Fprintln(w)

	// Pivot records back to user-per-row.
	userRows := pivotToUserRows(pr.Shifts, pr.Records)
	for _, row := range userRows {
		fmt.Fprintf(w, "%s\t", row.UserName)
		for _, cell := range row.Cells {
			if cell.OnSceneCount == 0 {
				fmt.Fprint(w, "—\t")
			} else {
				fmt.Fprintf(w, "%.1f%% (%d)\t", cell.OnScenePct, cell.OnSceneCount)
			}
			if cell.NotOnSceneCount == 0 {
				fmt.Fprint(w, "—\t")
			} else {
				fmt.Fprintf(w, "%.1f%% (%d)\t", cell.NotOnScenePct, cell.NotOnSceneCount)
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
		header = append(header,
			snakeCase(shift)+"_os_pct", snakeCase(shift)+"_os_calls",
			snakeCase(shift)+"_ns_pct", snakeCase(shift)+"_ns_calls",
		)
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
						line = append(line,
							fmt.Sprintf("%.1f", cell.OnScenePct), fmt.Sprintf("%d", cell.OnSceneCount),
							fmt.Sprintf("%.1f", cell.NotOnScenePct), fmt.Sprintf("%d", cell.NotOnSceneCount),
						)
						found = true
						break
					}
				}
				if !found {
					line = append(line, "0.0", "0", "0.0", "0")
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
