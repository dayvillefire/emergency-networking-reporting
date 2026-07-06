package shared

import "time"

// DateRange represents an inclusive date range for incident filtering.
type DateRange struct {
	Start time.Time
	End   time.Time
}

// LastMonth returns a DateRange covering the previous calendar month.
func LastMonth() DateRange {
	now := time.Now()
	// First day of previous month (Go normalizes month 0 → December of previous year).
	start := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, now.Location())
	// Last day of previous month: day 0 of current month = last day of prior month.
	end := time.Date(now.Year(), now.Month(), 0, 23, 59, 59, 999999999, now.Location())
	return DateRange{Start: start, End: end}
}

// LastQuarter returns a DateRange covering the previous calendar quarter.
func LastQuarter() DateRange {
	now := time.Now()
	// First month of the current quarter: ((month-1)/3)*3 + 1.
	// e.g. July (7) → ((6)/3)*3+1 = 7, March (3) → ((2)/3)*3+1 = 1.
	currentQStartMonth := ((now.Month()-1)/3)*3 + 1
	// Previous quarter starts 3 months earlier.
	prevQStartMonth := currentQStartMonth - 3
	year := now.Year()
	if prevQStartMonth <= 0 {
		prevQStartMonth += 12
		year--
	}
	start := time.Date(year, time.Month(prevQStartMonth), 1, 0, 0, 0, 0, now.Location())
	// Last day of previous quarter = day before current quarter starts.
	end := time.Date(now.Year(), time.Month(currentQStartMonth), 0, 23, 59, 59, 999999999, now.Location())
	return DateRange{Start: start, End: end}
}

// LastYear returns a DateRange covering the rolling 365-day period ending now.
func LastYear() DateRange {
	now := time.Now()
	return DateRange{
		Start: now.AddDate(-1, 0, 0),
		End:   now,
	}
}

// CurrentMonth returns a DateRange from the 1st of the current month through today.
func CurrentMonth() DateRange {
	now := time.Now()
	return DateRange{
		Start: time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
		End:   now,
	}
}

// FilterByDateRange returns incidents whose PSAPTime falls within [start, end].
func FilterByDateRange(incidents []NormalizedIncident, start, end time.Time) []NormalizedIncident {
	var out []NormalizedIncident
	for _, inc := range incidents {
		t := inc.PSAPTime
		if !t.IsZero() && (t.Equal(start) || t.After(start)) && t.Before(end.Add(24*time.Hour)) {
			out = append(out, inc)
		}
	}
	return out
}
