package shared

import "sort"

// PersonnelResponseSummary holds the three tiers of responder participation.
type PersonnelResponseSummary struct {
	High         []PersonnelRecord // >=30%
	Active       []PersonnelRecord // >=20% (excluding High)
	GoodStanding []PersonnelRecord // >=10% (excluding Active and High)
}

// ComputePersonnelResponse aggregates call counts per person across incidents,
// computes percentages against total calls, sorts by call count descending,
// and tiers into High (>=30%), Active (>=20%), GoodStanding (>=10%).
func ComputePersonnelResponse(incidents []NormalizedIncident, nameMap map[string]string) PersonnelResponseSummary {
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

	total := len(incidents)
	var list []PersonnelRecord
	for name, r := range counts {
		r.OnScene = onScene[name]
		r.NotOnScene = notOnScene[name]
		if total > 0 {
			r.Pct = float64(r.TotalCalls) / float64(total) * 100
		}
		list = append(list, *r)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].TotalCalls > list[j].TotalCalls })

	var s PersonnelResponseSummary
	for _, r := range list {
		if r.Pct >= 30 {
			s.High = append(s.High, r)
		} else if r.Pct >= 20 {
			s.Active = append(s.Active, r)
		} else if r.Pct >= 10 {
			s.GoodStanding = append(s.GoodStanding, r)
		}
	}
	return s
}

func resolveName(raw string, nameMap map[string]string) string {
	if name, ok := nameMap[raw]; ok {
		return name
	}
	return raw
}
