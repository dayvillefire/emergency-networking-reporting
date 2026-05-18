package main

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dayvillefire/emergency-networking-reporting/enapi"
)

// ReportStats holds all computed statistics for a reporting period.
type ReportStats struct {
	PeriodStart  time.Time
	PeriodEnd    time.Time
	DeptName     string
	GeneratedAt  time.Time
	TotalCalls   int

	// Section 1: EMS vs Fire
	EMSCount    int
	FireCount   int
	EMSPct      float64
	FirePct     float64

	// Section 2: Average personnel
	AvgPersonnel float64

	// Section 3: Response times (trimmed, in seconds)
	DispatchToEnRouteAvg float64
	DispatchToArrivalAvg float64
	TimeOnSceneAvg       float64

	// Section 4: Concurrent calls
	ConcurrentCount int
	ConcurrentPct   float64

	// Section 5: Mutual aid
	MutualAidGiven             int
	MutualAidGivenPct          float64
	MutualAidReceived          int
	MutualAidReceivedPct       float64
	MutualAidGivenDistricts    map[string]int

	// Section 6: EMS mutual aid by district
	EMSMutualAidTotal    int
	EMSMutualAidDistricts map[string]int

	// Section 7: Calls per shift
	ShiftCounts map[string]int

	// Section 8: Special incident types
	HazmatCount int
	HazmatPct   float64
	MVACount    int
	MVAPct      float64
	COCount     int
	COPct       float64

	// Section 9: Special unit responses
	DroneTeamCount int
	DroneTeamPct   float64
	RehabTeamCount int
	RehabTeamPct   float64

	// Monthly breakdowns for charting (quarter/year reports)
	Monthly []MonthlyStats
}

// MonthlyStats holds per-month breakdowns for charting.
type MonthlyStats struct {
	Month              string
	EMS                int
	Fire               int
	AvgPersonnel       float64
	DispatchToEnRoute  float64
	DispatchToArrival  float64
	TimeOnScene        float64
	Concurrent         int
	MutualAidGiven     int
	MutualAidReceived  int
	Hazmat             int
	MVA                int
	CO                 int
	DroneTeam          int
	RehabTeam          int
}

// ComputeStats runs all report calculations on the filtered incidents.
func ComputeStats(incidents []enapi.NerisIncident, deptName string, start, end time.Time) *ReportStats {
	s := &ReportStats{
		PeriodStart:             start,
		PeriodEnd:               end,
		DeptName:                deptName,
		GeneratedAt:             time.Now(),
		TotalCalls:              len(incidents),
		ShiftCounts:             make(map[string]int),
		EMSMutualAidDistricts:   make(map[string]int),
		MutualAidGivenDistricts:    make(map[string]int),
	}
	if len(incidents) == 0 {
		return s
	}

	computeEMSvsFire(s, incidents)
	computeAvgPersonnel(s, incidents)
	computeResponseTimes(s, incidents)
	computeConcurrent(s, incidents)
	computeMutualAid(s, incidents)
	computeEMSMutualAidByDistrict(s, incidents)
	computeByShift(s, incidents)
	computeSpecialTypes(s, incidents)
	computeSpecialUnits(s, incidents)
	computeMonthly(s, incidents)

	return s
}

func computeMonthly(s *ReportStats, incidents []enapi.NerisIncident) {
	months := monthsInRange(s.PeriodStart, s.PeriodEnd)
	if len(months) <= 1 {
		return
	}
	for _, m := range months {
		ms := MonthlyStats{Month: m.Format("January")}
		var bucket []enapi.NerisIncident
		mEnd := time.Date(m.Year(), m.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)
		for _, inc := range incidents {
			t := inc.IncidentPsapTime.Time
			if !t.IsZero() && (t.Equal(m) || t.After(m)) && t.Before(mEnd.Add(24*time.Hour)) {
				bucket = append(bucket, inc)
			}
		}
		if len(bucket) == 0 {
			s.Monthly = append(s.Monthly, ms)
			continue
		}
		// EMS vs Fire
		for _, inc := range bucket {
			if isEMS(inc) {
				ms.EMS++
			} else {
				ms.Fire++
			}
		}
		// Personnel
		var tp, cwp int
		for _, inc := range bucket {
			cp := 0
			for _, u := range inc.Units {
				n, _ := strconv.Atoi(string(u.UnitNumberOfPersonnel))
				if n > 0 { cp += n }
			}
			if cp == 0 { cp = len(inc.Personnel) }
			if cp > 0 { tp += cp; cwp++ }
		}
		if cwp > 0 { ms.AvgPersonnel = float64(tp) / float64(cwp) }
		// Response times
		ms.DispatchToEnRoute, ms.DispatchToArrival, ms.TimeOnScene = monthlyResponseTimes(bucket)
		// Concurrent
		ms.Concurrent = countConcurrent(bucket)
		// Mutual aid
		for _, inc := range bucket {
			if strings.ToLower(string(inc.MutualAidGivenOrReceived)) == "yes" {
				switch strings.ToLower(string(inc.MutualAidDirection)) {
				case "given": ms.MutualAidGiven++
				case "received": ms.MutualAidReceived++
				}
			}
		}
		// Special types
		for _, inc := range bucket {
			pt := strings.ToLower(string(inc.PrimaryIncidentType))
			disp := strings.ToLower(string(inc.IncidentDispatchedAs))
			if strings.Contains(pt, "fuel") || strings.Contains(pt, "hazmat") || strings.Contains(pt, "hazardous") || strings.Contains(disp, "fuel") || strings.Contains(disp, "hazmat") { ms.Hazmat++ }
			if strings.HasPrefix(disp, "[mva]") || strings.Contains(pt, "motor vehicle") || strings.Contains(pt, "vehicle fire") { ms.MVA++ }
			if strings.Contains(pt, "carbon monoxide") || strings.Contains(disp, "carbon monoxide") { ms.CO++ }
		}
		// Special units
		for _, inc := range bucket {
			for _, u := range inc.Units {
				if strings.Contains(strings.ToUpper(string(u.UnitName)), "UAV163") { ms.DroneTeam++; break }
			}
			for _, u := range inc.Units {
				if strings.Contains(strings.ToUpper(string(u.UnitName)), "S263") { ms.RehabTeam++; break }
			}
		}
		s.Monthly = append(s.Monthly, ms)
	}
}

func monthsInRange(start, end time.Time) []time.Time {
	var months []time.Time
	m := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	for !m.After(end) {
		months = append(months, m)
		m = m.AddDate(0, 1, 0)
	}
	return months
}

func monthlyResponseTimes(incidents []enapi.NerisIncident) (enRoute, arrival, scene float64) {
	var dToE, dToA, onS []float64
	zero := time.Time{}
	for _, inc := range incidents {
		if inc.IncidentDispatchTime.Time != zero && inc.IncidentEnrouteTime.Time != zero {
			if d := inc.IncidentEnrouteTime.Time.Sub(inc.IncidentDispatchTime.Time).Seconds(); d > 0 { dToE = append(dToE, d) }
		}
		if inc.IncidentDispatchTime.Time != zero && inc.IncidentArrivalTime.Time != zero {
			if d := inc.IncidentArrivalTime.Time.Sub(inc.IncidentDispatchTime.Time).Seconds(); d > 0 { dToA = append(dToA, d) }
		}
		if inc.IncidentArrivalTime.Time != zero && inc.IncidentClearTime.Time != zero {
			if d := inc.IncidentClearTime.Time.Sub(inc.IncidentArrivalTime.Time).Seconds(); d > 0 { onS = append(onS, d) }
		}
	}
	return trimmedMean(dToE), trimmedMean(dToA), trimmedMean(onS)
}

func countConcurrent(incidents []enapi.NerisIncident) int {
	zero := time.Time{}
	type interval struct{ start, end time.Time }
	intervals := make([]interval, 0, len(incidents))
	for _, inc := range incidents {
		if inc.IncidentPsapTime.Time == zero || inc.IncidentClearTime.Time == zero { continue }
		intervals = append(intervals, interval{inc.IncidentPsapTime.Time, inc.IncidentClearTime.Time})
	}
	sort.Slice(intervals, func(i, j int) bool { return intervals[i].start.Before(intervals[j].start) })
	concurrent := make(map[int]bool)
	for i := range intervals {
		for j := i + 1; j < len(intervals); j++ {
			if intervals[j].start.Before(intervals[i].end) || intervals[j].start.Equal(intervals[i].end) {
				concurrent[i] = true; concurrent[j] = true
			} else { break }
		}
	}
	return len(concurrent)
}

func isEMS(inc enapi.NerisIncident) bool {
	// Check dispatched-as code for medical call types
	disp := string(inc.IncidentDispatchedAs)
	dispLower := strings.ToLower(disp)
	if strings.HasPrefix(disp, "[M]") || strings.Contains(dispLower, "medical") {
		return true
	}
	// Public lift assists count as EMS
	if strings.Contains(dispLower, "lift assist") || strings.Contains(dispLower, "public assist") {
		return true
	}
	// Check primary incident type for medical keywords
	pt := strings.ToLower(string(inc.PrimaryIncidentType))
	medicalTypes := []string{
		"breathing", "cardiac", "chest pain", "fall", "sick",
		"overdose", "unconscious", "pregnancy", "back pain",
		"altered mental", "psychological", "seizure", "stroke",
		"hemorrhage", "allergic", "diabetic",
	}
	for _, kw := range medicalTypes {
		if strings.Contains(pt, kw) {
			return true
		}
	}
	return false
}

func computeEMSvsFire(s *ReportStats, incidents []enapi.NerisIncident) {
	for _, inc := range incidents {
		if isEMS(inc) {
			s.EMSCount++
		} else {
			s.FireCount++
		}
	}
	if s.TotalCalls > 0 {
		s.EMSPct = float64(s.EMSCount) / float64(s.TotalCalls) * 100
		s.FirePct = float64(s.FireCount) / float64(s.TotalCalls) * 100
	}
}

func computeAvgPersonnel(s *ReportStats, incidents []enapi.NerisIncident) {
	var totalPersonnel int
	var callsWithPersonnel int
	for _, inc := range incidents {
		callPersonnel := 0
		// Count from unit personnel numbers
		for _, u := range inc.Units {
			n, err := strconv.Atoi(string(u.UnitNumberOfPersonnel))
			if err == nil && n > 0 {
				callPersonnel += n
			}
		}
		// Fall back to incident-level personnel array
		if callPersonnel == 0 {
			callPersonnel = len(inc.Personnel)
		}
		if callPersonnel > 0 {
			totalPersonnel += callPersonnel
			callsWithPersonnel++
		}
	}
	if callsWithPersonnel > 0 {
		s.AvgPersonnel = float64(totalPersonnel) / float64(callsWithPersonnel)
	}
}

func computeResponseTimes(s *ReportStats, incidents []enapi.NerisIncident) {
	var dispatchToEnRoute, dispatchToArrival, timeOnScene []float64

	for _, inc := range incidents {
		dispatchT := inc.IncidentDispatchTime.Time
		enrouteT := inc.IncidentEnrouteTime.Time
		arrivalT := inc.IncidentArrivalTime.Time
		clearT := inc.IncidentClearTime.Time

		zero := time.Time{}

		if dispatchT != zero && enrouteT != zero {
			d := enrouteT.Sub(dispatchT).Seconds()
			if d > 0 {
				dispatchToEnRoute = append(dispatchToEnRoute, d)
			}
		}
		if dispatchT != zero && arrivalT != zero {
			d := arrivalT.Sub(dispatchT).Seconds()
			if d > 0 {
				dispatchToArrival = append(dispatchToArrival, d)
			}
		}
		if arrivalT != zero && clearT != zero {
			d := clearT.Sub(arrivalT).Seconds()
			if d > 0 {
				timeOnScene = append(timeOnScene, d)
			}
		}
	}

	s.DispatchToEnRouteAvg = trimmedMean(dispatchToEnRoute)
	s.DispatchToArrivalAvg = trimmedMean(dispatchToArrival)
	s.TimeOnSceneAvg = trimmedMean(timeOnScene)
}

// trimmedMean sorts durations, drops top/bottom 10%, and returns the mean.
func trimmedMean(durations []float64) float64 {
	if len(durations) == 0 {
		return 0
	}
	sort.Float64s(durations)
	drop := len(durations) / 10
	if drop == 0 {
		drop = 0 // keep as-is if too few
	}
	// If fewer than 5 items, drop nothing (can't trim meaningfully)
	if len(durations) < 5 {
		var sum float64
		for _, d := range durations {
			sum += d
		}
		return sum / float64(len(durations))
	}
	trimmed := durations[drop : len(durations)-drop]
	if len(trimmed) == 0 {
		return 0
	}
	var sum float64
	for _, d := range trimmed {
		sum += d
	}
	return sum / float64(len(trimmed))
}

func computeConcurrent(s *ReportStats, incidents []enapi.NerisIncident) {
	zero := time.Time{}
	type interval struct{ start, end time.Time }
	intervals := make([]interval, 0, len(incidents))
	for _, inc := range incidents {
		psap := inc.IncidentPsapTime.Time
		clearT := inc.IncidentClearTime.Time
		if psap == zero || clearT == zero {
			continue
		}
		intervals = append(intervals, interval{psap, clearT})
	}

	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].start.Before(intervals[j].start)
	})

	concurrent := make(map[int]bool)
	for i := 0; i < len(intervals); i++ {
		for j := i + 1; j < len(intervals); j++ {
			// If j starts before i ends, they overlap
			if intervals[j].start.Before(intervals[i].end) || intervals[j].start.Equal(intervals[i].end) {
				concurrent[i] = true
				concurrent[j] = true
			} else {
				break // intervals are sorted by start, no more overlaps for i
			}
		}
	}
	s.ConcurrentCount = len(concurrent)
	if s.TotalCalls > 0 {
		s.ConcurrentPct = float64(s.ConcurrentCount) / float64(s.TotalCalls) * 100
	}
}

func computeMutualAid(s *ReportStats, incidents []enapi.NerisIncident) {
	for _, inc := range incidents {
		if strings.ToLower(string(inc.MutualAidGivenOrReceived)) == "yes" {
			district := string(inc.IncidentDistrict)
			if district == "" {
				district = "(none)"
			}
			dir := strings.ToLower(string(inc.MutualAidDirection))
			switch dir {
			case "given":
				s.MutualAidGiven++
				s.MutualAidGivenDistricts[district]++
			case "received":
				s.MutualAidReceived++
			}
		}
	}
	if s.TotalCalls > 0 {
		s.MutualAidGivenPct = float64(s.MutualAidGiven) / float64(s.TotalCalls) * 100
		s.MutualAidReceivedPct = float64(s.MutualAidReceived) / float64(s.TotalCalls) * 100
	}
}

func computeEMSMutualAidByDistrict(s *ReportStats, incidents []enapi.NerisIncident) {
	for _, inc := range incidents {
		if !isEMS(inc) {
			continue
		}
		if strings.ToLower(string(inc.MutualAidGivenOrReceived)) != "yes" {
			continue
		}
		if strings.ToLower(string(inc.MutualAidDirection)) != "given" {
			continue
		}
		district := string(inc.IncidentDistrict)
		if district == "" {
			district = "(none)"
		}
		s.EMSMutualAidDistricts[district]++
		s.EMSMutualAidTotal++
	}
}

func computeByShift(s *ReportStats, incidents []enapi.NerisIncident) {
	for _, inc := range incidents {
		shift := string(inc.IncidentShift)
		if shift == "" {
			shift = "(unknown)"
		}
		s.ShiftCounts[shift]++
	}
}

func computeSpecialTypes(s *ReportStats, incidents []enapi.NerisIncident) {
	for _, inc := range incidents {
		pt := strings.ToLower(string(inc.PrimaryIncidentType))
		disp := strings.ToLower(string(inc.IncidentDispatchedAs))
		allTypes := strings.ToLower(strings.Join(inc.IncidentType, " "))

		// Hazardous materials: fuel spills, hazmat, chemical
		if strings.Contains(pt, "fuel") || strings.Contains(pt, "hazmat") ||
			strings.Contains(pt, "hazardous") || strings.Contains(pt, "chemical") ||
			strings.Contains(disp, "fuel") || strings.Contains(disp, "hazmat") ||
			strings.Contains(allTypes, "hazmat") || strings.Contains(allTypes, "hazardous") {
			s.HazmatCount++
		}
		// Motor vehicle accidents
		if strings.HasPrefix(disp, "[mva]") || strings.Contains(pt, "motor vehicle") ||
			strings.Contains(pt, "vehicle fire") || strings.Contains(allTypes, "mva") {
			s.MVACount++
		}
		// Carbon monoxide: CO alarm, carbon monoxide
		if strings.Contains(pt, "carbon monoxide") || strings.Contains(disp, "carbon monoxide") ||
			strings.Contains(pt, "co alarm") || strings.Contains(disp, "co alarm") ||
			strings.Contains(allTypes, "carbon_monoxide") || strings.Contains(allTypes, "co_") {
			s.COCount++
		}
	}
	if s.TotalCalls > 0 {
		s.HazmatPct = float64(s.HazmatCount) / float64(s.TotalCalls) * 100
		s.MVAPct = float64(s.MVACount) / float64(s.TotalCalls) * 100
		s.COPct = float64(s.COCount) / float64(s.TotalCalls) * 100
	}
}

func computeSpecialUnits(s *ReportStats, incidents []enapi.NerisIncident) {
	for _, inc := range incidents {
		for _, u := range inc.Units {
			unitName := strings.ToUpper(string(u.UnitName))
			if strings.Contains(unitName, "UAV163") {
				s.DroneTeamCount++
				break // count each incident once
			}
		}
		for _, u := range inc.Units {
			unitName := strings.ToUpper(string(u.UnitName))
			if strings.Contains(unitName, "S263") {
				s.RehabTeamCount++
				break
			}
		}
	}
	if s.TotalCalls > 0 {
		s.DroneTeamPct = float64(s.DroneTeamCount) / float64(s.TotalCalls) * 100
		s.RehabTeamPct = float64(s.RehabTeamCount) / float64(s.TotalCalls) * 100
	}
}
