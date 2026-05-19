package main

import (
	"sort"
	"strings"
	"time"
)

// PersonnelRecord holds participation stats for one person.
type PersonnelRecord struct {
	Name        string
	TotalCalls  int
	Pct         float64
	OnScene     int
	NotOnScene  int
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

	PersonnelHigh             []PersonnelRecord
	PersonnelHighCount        int
	PersonnelActive           []PersonnelRecord
	PersonnelActiveCount      int
	PersonnelGoodStanding     []PersonnelRecord
	PersonnelGoodStandingCount int
	PersonnelTotalResponding  int

	// Monthly breakdowns for charting (quarter/year reports)
	Monthly []MonthlyStats
}

// ComputeStats runs all report calculations on the filtered incidents.
func ComputeStats(incidents []NormalizedIncident, deptName string, start, end time.Time, nameMap map[string]string) *ReportStats {
	s := &ReportStats{
		PeriodStart:             start,
		PeriodEnd:               end,
		DeptName:                deptName,
		GeneratedAt:             time.Now(),
		TotalCalls:              len(incidents),
		ShiftCounts:             make(map[string]int),
		EMSMutualAidDistricts:   make(map[string]int),
		MutualAidGivenDistricts: make(map[string]int),
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
	computePersonnelResponse(s, incidents, nameMap)
	computeMonthly(s, incidents)

	return s
}

func isEMS(inc NormalizedIncident) bool {
	disp := inc.DispatchedAs
	dispLower := strings.ToLower(disp)
	if strings.HasPrefix(disp, "[M]") || strings.Contains(dispLower, "medical") {
		return true
	}
	if strings.Contains(dispLower, "lift assist") || strings.Contains(dispLower, "public assist") {
		return true
	}

	// NERIS text-based classification
	pt := strings.ToLower(inc.PrimaryIncidentType)
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

	// NFIRS numeric codes: 300-series are EMS/rescue calls (exclude MVA codes)
	code := strings.TrimSpace(inc.PrimaryIncidentType)
	if len(code) >= 3 {
		prefix := code[:3]
		switch prefix {
		case "321", "311", "554":
			return true // EMS call, medical assist, lift assist
		case "322", "323", "324":
			return false // MVA — not counted as medical
		}
		// 381.x is rescue — count as Fire
		if strings.HasPrefix(code, "381") {
			return false
		}
	}

	return false
}

func computeEMSvsFire(s *ReportStats, incidents []NormalizedIncident) {
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

func computeAvgPersonnel(s *ReportStats, incidents []NormalizedIncident) {
	var totalPersonnel int
	var callsWithPersonnel int
	for _, inc := range incidents {
		p := inc.UnitPersonnel
		if p == 0 {
			p = inc.PersonnelCount
		}
		if p > 0 {
			totalPersonnel += p
			callsWithPersonnel++
		}
	}
	if callsWithPersonnel > 0 {
		s.AvgPersonnel = float64(totalPersonnel) / float64(callsWithPersonnel)
	}
}

func computeResponseTimes(s *ReportStats, incidents []NormalizedIncident) {
	var dispatchToEnRoute, dispatchToArrival, timeOnScene []float64
	zero := time.Time{}

	for _, inc := range incidents {
		if inc.DispatchTime != zero && inc.EnrouteTime != zero {
			if d := inc.EnrouteTime.Sub(inc.DispatchTime).Seconds(); d > 0 {
				dispatchToEnRoute = append(dispatchToEnRoute, d)
			}
		}
		if inc.DispatchTime != zero && inc.ArrivalTime != zero {
			if d := inc.ArrivalTime.Sub(inc.DispatchTime).Seconds(); d > 0 {
				dispatchToArrival = append(dispatchToArrival, d)
			}
		}
		if inc.ArrivalTime != zero && inc.ClearTime != zero {
			if d := inc.ClearTime.Sub(inc.ArrivalTime).Seconds(); d > 0 {
				timeOnScene = append(timeOnScene, d)
			}
		}
	}

	s.DispatchToEnRouteAvg = trimmedMean(dispatchToEnRoute)
	s.DispatchToArrivalAvg = trimmedMean(dispatchToArrival)
	s.TimeOnSceneAvg = trimmedMean(timeOnScene)
}

func trimmedMean(durations []float64) float64 {
	if len(durations) == 0 {
		return 0
	}
	sort.Float64s(durations)
	if len(durations) < 5 {
		var sum float64
		for _, d := range durations {
			sum += d
		}
		return sum / float64(len(durations))
	}
	drop := len(durations) / 10
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

func computeConcurrent(s *ReportStats, incidents []NormalizedIncident) {
	s.ConcurrentCount = countConcurrent(incidents)
	if s.TotalCalls > 0 {
		s.ConcurrentPct = float64(s.ConcurrentCount) / float64(s.TotalCalls) * 100
	}
}

func computeMutualAid(s *ReportStats, incidents []NormalizedIncident) {
	for _, inc := range incidents {
		district := inc.District
		if district == "" {
			district = "(none)"
		}
		if inc.MutualAidGiven {
			s.MutualAidGiven++
			s.MutualAidGivenDistricts[district]++
		}
		if inc.MutualAidReceived {
			s.MutualAidReceived++
		}
	}
	if s.TotalCalls > 0 {
		s.MutualAidGivenPct = float64(s.MutualAidGiven) / float64(s.TotalCalls) * 100
		s.MutualAidReceivedPct = float64(s.MutualAidReceived) / float64(s.TotalCalls) * 100
	}
}

func computeEMSMutualAidByDistrict(s *ReportStats, incidents []NormalizedIncident) {
	for _, inc := range incidents {
		if !isEMS(inc) {
			continue
		}
		if !inc.MutualAidGiven {
			continue
		}
		district := inc.District
		if district == "" {
			district = "(none)"
		}
		s.EMSMutualAidDistricts[district]++
		s.EMSMutualAidTotal++
	}
}

func computeByShift(s *ReportStats, incidents []NormalizedIncident) {
	for _, inc := range incidents {
		shift := inc.Shift
		if shift == "" {
			shift = "(unknown)"
		}
		s.ShiftCounts[shift]++
	}
}

func isHazmat(inc NormalizedIncident) bool {
	pt := strings.ToLower(inc.PrimaryIncidentType)
	disp := strings.ToLower(inc.DispatchedAs)
	allTypes := strings.ToLower(strings.Join(inc.IncidentTypes, " "))
	if strings.Contains(pt, "fuel") || strings.Contains(pt, "hazmat") ||
		strings.Contains(pt, "hazardous") || strings.Contains(pt, "chemical") ||
		strings.Contains(disp, "fuel") || strings.Contains(disp, "hazmat") ||
		strings.Contains(allTypes, "hazmat") || strings.Contains(allTypes, "hazardous") {
		return true
	}
	if len(inc.PrimaryIncidentType) == 3 && inc.PrimaryIncidentType[0] == '4' {
		return true
	}
	return false
}

func isMVA(inc NormalizedIncident) bool {
	pt := strings.ToLower(inc.PrimaryIncidentType)
	disp := strings.ToLower(inc.DispatchedAs)
	allTypes := strings.ToLower(strings.Join(inc.IncidentTypes, " "))
	if strings.HasPrefix(disp, "[mva]") || strings.Contains(pt, "motor vehicle") ||
		strings.Contains(pt, "vehicle fire") || strings.Contains(allTypes, "mva") {
		return true
	}
	switch inc.PrimaryIncidentType {
	case "322", "323", "324":
		return true
	}
	return false
}

func isCO(inc NormalizedIncident) bool {
	pt := strings.ToLower(inc.PrimaryIncidentType)
	disp := strings.ToLower(inc.DispatchedAs)
	allTypes := strings.ToLower(strings.Join(inc.IncidentTypes, " "))
	if strings.Contains(pt, "carbon monoxide") || strings.Contains(disp, "carbon monoxide") ||
		strings.Contains(pt, "co alarm") || strings.Contains(disp, "co alarm") ||
		strings.Contains(allTypes, "carbon_monoxide") || strings.Contains(allTypes, "co_") {
		return true
	}
	switch inc.PrimaryIncidentType {
	case "424", "425", "736", "746":
		return true
	}
	return false
}


func computeSpecialTypes(s *ReportStats, incidents []NormalizedIncident) {
	for _, inc := range incidents {
		if isHazmat(inc) { s.HazmatCount++ }
		if isMVA(inc) { s.MVACount++ }
		if isCO(inc) { s.COCount++ }
	}
	if s.TotalCalls > 0 {
		s.HazmatPct = float64(s.HazmatCount) / float64(s.TotalCalls) * 100
		s.MVAPct = float64(s.MVACount) / float64(s.TotalCalls) * 100
		s.COPct = float64(s.COCount) / float64(s.TotalCalls) * 100
	}
}

func computeSpecialUnits(s *ReportStats, incidents []NormalizedIncident) {
	for _, inc := range incidents {
		for _, name := range inc.UnitNames {
			if strings.Contains(strings.ToUpper(name), "UAV163") {
				s.DroneTeamCount++
				break
			}
		}
		for _, name := range inc.UnitNames {
			if strings.Contains(strings.ToUpper(name), "S263") {
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

// ---- Monthly ----

func resolveName(raw string, nameMap map[string]string) string {
	if name, ok := nameMap[raw]; ok {
		return name
	}
	return raw
}

func computePersonnelResponse(s *ReportStats, incidents []NormalizedIncident, nameMap map[string]string) {
	counts := make(map[string]*PersonnelRecord)
	onScene := make(map[string]int)
	notOnScene := make(map[string]int)

	for _, inc := range incidents {
		seen := make(map[string]bool)
		for _, raw := range inc.OnScenePersonnel {
			name := resolveName(raw, nameMap)
			seen[name] = true
			onScene[name]++
		}
		for _, raw := range inc.NotOnScenePersonnel {
			name := resolveName(raw, nameMap)
			seen[name] = true
			notOnScene[name]++
		}
		for name := range seen {
			if counts[name] == nil {
				counts[name] = &PersonnelRecord{Name: name}
			}
			counts[name].TotalCalls++
		}
	}

	var list []PersonnelRecord
	for name, r := range counts {
		r.OnScene = onScene[name]
		r.NotOnScene = notOnScene[name]
		if s.TotalCalls > 0 {
			r.Pct = float64(r.TotalCalls) / float64(s.TotalCalls) * 100
		}
		list = append(list, *r)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].TotalCalls > list[j].TotalCalls })

	for _, r := range list {
		s.PersonnelTotalResponding++
		if r.Pct >= 30 {
			s.PersonnelHigh = append(s.PersonnelHigh, r)
		} else if r.Pct >= 20 {
			s.PersonnelActive = append(s.PersonnelActive, r)
		} else if r.Pct >= 10 {
			s.PersonnelGoodStanding = append(s.PersonnelGoodStanding, r)
		}
	}
	s.PersonnelHighCount = len(s.PersonnelHigh)
	s.PersonnelActiveCount = len(s.PersonnelHigh) + len(s.PersonnelActive)
	s.PersonnelGoodStandingCount = len(s.PersonnelHigh) + len(s.PersonnelActive) + len(s.PersonnelGoodStanding)
}


func computeMonthly(s *ReportStats, incidents []NormalizedIncident) {
	months := monthsInRange(s.PeriodStart, s.PeriodEnd)
	if len(months) <= 1 {
		return
	}
	for _, m := range months {
		ms := MonthlyStats{Month: m.Format("January")}
		var bucket []NormalizedIncident
		mEnd := time.Date(m.Year(), m.Month()+1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Second)
		for _, inc := range incidents {
			if !inc.PSAPTime.IsZero() && (inc.PSAPTime.Equal(m) || inc.PSAPTime.After(m)) && inc.PSAPTime.Before(mEnd.Add(24*time.Hour)) {
				bucket = append(bucket, inc)
			}
		}
		if len(bucket) == 0 {
			s.Monthly = append(s.Monthly, ms)
			continue
		}
		for _, inc := range bucket {
			if isEMS(inc) {
				ms.EMS++
			} else {
				ms.Fire++
			}
		}
		var tp, cwp int
		for _, inc := range bucket {
			p := inc.UnitPersonnel
			if p == 0 { p = inc.PersonnelCount }
			if p > 0 { tp += p; cwp++ }
		}
		if cwp > 0 { ms.AvgPersonnel = float64(tp) / float64(cwp) }
		ms.DispatchToEnRoute, ms.DispatchToArrival, ms.TimeOnScene = monthlyResponseTimes(bucket)
		ms.Concurrent = countConcurrent(bucket)
		for _, inc := range bucket {
			if inc.MutualAidGiven { ms.MutualAidGiven++ }
			if inc.MutualAidReceived { ms.MutualAidReceived++ }
		}
		for _, inc := range bucket {
			if isHazmat(inc) { ms.Hazmat++ }
			if isMVA(inc) { ms.MVA++ }
			if isCO(inc) { ms.CO++ }
		}
		for _, inc := range bucket {
			for _, name := range inc.UnitNames {
				if strings.Contains(strings.ToUpper(name), "UAV163") { ms.DroneTeam++; break }
			}
			for _, name := range inc.UnitNames {
				if strings.Contains(strings.ToUpper(name), "S263") { ms.RehabTeam++; break }
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

func monthlyResponseTimes(incidents []NormalizedIncident) (enRoute, arrival, scene float64) {
	var dToE, dToA, onS []float64
	zero := time.Time{}
	for _, inc := range incidents {
		if inc.DispatchTime != zero && inc.EnrouteTime != zero {
			if d := inc.EnrouteTime.Sub(inc.DispatchTime).Seconds(); d > 0 { dToE = append(dToE, d) }
		}
		if inc.DispatchTime != zero && inc.ArrivalTime != zero {
			if d := inc.ArrivalTime.Sub(inc.DispatchTime).Seconds(); d > 0 { dToA = append(dToA, d) }
		}
		if inc.ArrivalTime != zero && inc.ClearTime != zero {
			if d := inc.ClearTime.Sub(inc.ArrivalTime).Seconds(); d > 0 { onS = append(onS, d) }
		}
	}
	return trimmedMean(dToE), trimmedMean(dToA), trimmedMean(onS)
}

func countConcurrent(incidents []NormalizedIncident) int {
	zero := time.Time{}
	type interval struct{ start, end time.Time }
	intervals := make([]interval, 0, len(incidents))
	for _, inc := range incidents {
		if inc.PSAPTime == zero || inc.ClearTime == zero { continue }
		intervals = append(intervals, interval{inc.PSAPTime, inc.ClearTime})
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
