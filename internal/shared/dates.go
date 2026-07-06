package shared

import "time"

// DateRange represents an inclusive date range for incident filtering.
type DateRange struct {
	Start time.Time
	End   time.Time
}

// LastMonth returns a DateRange covering the rolling 30-day period ending now.
func LastMonth() DateRange {
	now := time.Now()
	return DateRange{
		Start: now.AddDate(0, -1, 0),
		End:   now,
	}
}

// LastQuarter returns a DateRange covering the rolling 90-day period ending now.
func LastQuarter() DateRange {
	now := time.Now()
	return DateRange{
		Start: now.AddDate(0, -3, 0),
		End:   now,
	}
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
