package main

import (
	"sort"

	"github.com/dayvillefire/emergency-networking-reporting/internal/shared"
)

// ShiftUserRecord holds participation stats for one user in one shift within a period.
type ShiftUserRecord struct {
	UserName        string
	Shift           string
	CallCount       int     // combined total (union of on-scene + not-on-scene)
	OnSceneCount    int     // apparatus calls (from OnScenePersonnel)
	NotOnSceneCount int     // non-apparatus calls (from NotOnScenePersonnel)
	OnScenePct      float64 // OnSceneCount / TotalInShift * 100
	NotOnScenePct   float64 // NotOnSceneCount / TotalInShift * 100
	TotalInShift    int
	Pct             float64 // combined percentage
}

// PeriodResult holds all shift-user records for a single time period.
type PeriodResult struct {
	Label      string
	Range      shared.DateRange
	TotalCalls int
	Shifts     []string          // ordered shift names
	Records    []ShiftUserRecord // sorted by UserName then Shift
}

// ComputeShiftParticipation calculates per-shift call percentages for each selected user
// across the given incidents. selectedUsers is a set of resolved "Last, First" names.
// nameMap maps raw personnel IDs to display names.
func ComputeShiftParticipation(
	incidents []shared.NormalizedIncident,
	selectedUsers map[string]bool,
	nameMap map[string]string,
) ([]string, []ShiftUserRecord) {
	// Collect unique shift names and count total calls per shift.
	shiftTotals := make(map[string]int)
	for _, inc := range incidents {
		shift := inc.Shift
		if shift == "" {
			shift = "(unknown)"
		}
		shiftTotals[shift]++
	}

	// Count per-user per-shift, split by on-scene (apparatus) vs not-on-scene.
	// Key: userName + "||" + shift
	userShiftOnScene := make(map[string]int)
	userShiftNotOnScene := make(map[string]int)
	userShiftCombined := make(map[string]int)
	for _, inc := range incidents {
		shift := inc.Shift
		if shift == "" {
			shift = "(unknown)"
		}
		// Collect unique names per list (dedup within each list).
		onScene := make(map[string]bool)
		notOnScene := make(map[string]bool)
		for _, raw := range inc.OnScenePersonnel {
			name := resolveName(raw, nameMap)
			if selectedUsers[name] {
				onScene[name] = true
			}
		}
		for _, raw := range inc.NotOnScenePersonnel {
			name := resolveName(raw, nameMap)
			if selectedUsers[name] {
				notOnScene[name] = true
			}
		}
		for name := range onScene {
			userShiftOnScene[name+"||"+shift]++
		}
		for name := range notOnScene {
			userShiftNotOnScene[name+"||"+shift]++
		}
		// Combined = union (dedup across both lists).
		for name := range onScene {
			userShiftCombined[name+"||"+shift]++
		}
		for name := range notOnScene {
			if !onScene[name] {
				userShiftCombined[name+"||"+shift]++
			}
		}
	}

	// Build ordered shift list (sorted alphabetically).
	var shifts []string
	for shift := range shiftTotals {
		shifts = append(shifts, shift)
	}
	sort.Strings(shifts)

	// Collect unique user names.
	userSet := make(map[string]bool)
	for name := range selectedUsers {
		userSet[name] = true
	}
	for key := range userShiftCombined {
		// key is "name||shift"
		idx := stringsLastIndex(key, "||")
		if idx >= 0 {
			userSet[key[:idx]] = true
		}
	}
	var users []string
	for name := range userSet {
		users = append(users, name)
	}
	sort.Strings(users)

	// Build records: one per user per shift.
	var records []ShiftUserRecord
	for _, user := range users {
		for _, shift := range shifts {
			key := user + "||" + shift
			count := userShiftCombined[key]
			osCount := userShiftOnScene[key]
			nsCount := userShiftNotOnScene[key]
			total := shiftTotals[shift]
			pct := 0.0
			osPct := 0.0
			nsPct := 0.0
			if total > 0 {
				pct = float64(count) / float64(total) * 100
				osPct = float64(osCount) / float64(total) * 100
				nsPct = float64(nsCount) / float64(total) * 100
			}
			records = append(records, ShiftUserRecord{
				UserName:        user,
				Shift:           shift,
				CallCount:       count,
				OnSceneCount:    osCount,
				NotOnSceneCount: nsCount,
				OnScenePct:      osPct,
				NotOnScenePct:   nsPct,
				TotalInShift:    total,
				Pct:             pct,
			})
		}
	}

	return shifts, records
}

// stringsLastIndex returns the last index of substr in s, or -1.
func stringsLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// resolveName looks up a raw personnel identifier in the name map.
func resolveName(raw string, nameMap map[string]string) string {
	if name, ok := nameMap[raw]; ok {
		return name
	}
	return raw
}
