package shared

import (
	"strconv"
	"strings"

	"github.com/dayvillefire/emergency-networking-reporting/enapi"
)

// NormNerisIncident converts a NERIS incident to the normalized type.
func NormNerisIncident(inc enapi.NerisIncident) NormalizedIncident {
	n := NormalizedIncident{
		PSAPTime:           inc.IncidentPsapTime.Time,
		DispatchTime:       inc.IncidentDispatchTime.Time,
		EnrouteTime:        inc.IncidentEnrouteTime.Time,
		ArrivalTime:        inc.IncidentArrivalTime.Time,
		ClearTime:          inc.IncidentClearTime.Time,
		Shift:              string(inc.IncidentShift),
		Station:            string(inc.IncidentStation),
		District:           string(inc.IncidentDistrict),
		DispatchedAs:       string(inc.IncidentDispatchedAs),
		PrimaryIncidentType: string(inc.PrimaryIncidentType),
		IncidentTypes:      inc.IncidentType,
	}

	mg := strings.ToLower(string(inc.MutualAidGivenOrReceived))
	md := strings.ToLower(string(inc.MutualAidDirection))
	n.MutualAidGiven = mg == "yes" && md == "given"
	n.MutualAidReceived = mg == "yes" && md == "received"

	for _, u := range inc.Units {
		n.UnitNames = append(n.UnitNames, string(u.UnitName))
		p, err := strconv.Atoi(string(u.UnitNumberOfPersonnel))
		if err == nil && p > 0 {
			n.UnitPersonnel += p
		}
	}
	if n.UnitPersonnel == 0 {
		n.UnitPersonnel = len(inc.Personnel)
	}
	n.PersonnelCount = len(inc.Personnel)

	n.UnitPersonnel += len(inc.IncidentAdditionalResponders)
	n.PersonnelCount += len(inc.IncidentAdditionalResponders)
	for _, r := range inc.IncidentAdditionalResponders {
		if r != "" {
			n.NotOnScenePersonnel = append(n.NotOnScenePersonnel, r)
		}
	}
	for _, ma := range inc.MutualAid {
		if maCount, err := strconv.Atoi(string(ma.MutualAidNumberOfPersonnel)); err == nil && maCount > 0 {
			n.UnitPersonnel += maCount
			n.PersonnelCount += maCount
		}
	}

	for _, p := range inc.Personnel {
		name := string(p.PersonnelName)
		if name == "" {
			continue
		}
		if string(p.PersonnelUnit) != "" {
			n.OnScenePersonnel = append(n.OnScenePersonnel, name)
		} else {
			n.NotOnScenePersonnel = append(n.NotOnScenePersonnel, name)
		}
	}
	return n
}

// NormIncident converts a legacy NFIRS incident to the normalized type.
func NormIncident(inc enapi.Incident) NormalizedIncident {
	n := NormalizedIncident{
		PSAPTime:           inc.Psap.Time,
		EnrouteTime:        inc.Enroute.Time,
		ArrivalTime:        inc.Arrival.Time,
		ClearTime:          inc.LastUnitCleared.Time,
		Shift:              inc.Shift,
		Station:            inc.Station,
		District:           inc.District,
		DispatchedAs:       inc.DispatchedAs,
		PrimaryIncidentType: inc.IncidentType,
		IncidentTypes:      []string{inc.IncidentType},
	}

	for _, u := range inc.Units {
		if !u.Dispatch.Time.IsZero() {
			n.DispatchTime = u.Dispatch.Time
			break
		}
	}

	ma := strings.ToLower(inc.MutualAid)
	n.MutualAidGiven = strings.Contains(ma, "given") && !strings.Contains(ma, "received")
	n.MutualAidReceived = strings.Contains(ma, "received")

	for _, u := range inc.Units {
		n.UnitNames = append(n.UnitNames, u.Apparatus)
		n.PersonnelCount += len(u.Personnel)
		for _, p := range u.Personnel {
			n.UnitPersonnel++
			if p.CrewMember != "" {
				n.OnScenePersonnel = append(n.OnScenePersonnel, p.CrewMember)
			}
		}
	}

	n.UnitPersonnel += len(inc.AdditionalResponders)
	n.PersonnelCount += len(inc.AdditionalResponders)
	for _, r := range inc.AdditionalResponders {
		if r != "" {
			n.NotOnScenePersonnel = append(n.NotOnScenePersonnel, r)
		}
	}
	if maCount, err := strconv.Atoi(inc.MutualAidPersonnelCount); err == nil && maCount > 0 {
		n.UnitPersonnel += maCount
		n.PersonnelCount += maCount
	}

	return n
}

// NormNerisSlice normalizes a slice of NERIS incidents.
func NormNerisSlice(incidents []enapi.NerisIncident) []NormalizedIncident {
	out := make([]NormalizedIncident, len(incidents))
	for i, inc := range incidents {
		out[i] = NormNerisIncident(inc)
	}
	return out
}

// NormIncidentSlice normalizes a slice of legacy incidents.
func NormIncidentSlice(incidents []enapi.Incident) []NormalizedIncident {
	out := make([]NormalizedIncident, len(incidents))
	for i, inc := range incidents {
		out[i] = NormIncident(inc)
	}
	return out
}
