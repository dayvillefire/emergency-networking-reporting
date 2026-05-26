package shared

import "time"

// NormalizedIncident is a unified incident type that both NerisIncident
// and legacy Incident (NFIRS-format) can be converted into.
type NormalizedIncident struct {
	PSAPTime            time.Time
	DispatchTime        time.Time
	EnrouteTime         time.Time
	ArrivalTime         time.Time
	ClearTime           time.Time
	Shift               string
	Station             string
	District            string
	DispatchedAs        string
	PrimaryIncidentType string
	IncidentTypes       []string
	MutualAidGiven      bool
	MutualAidReceived   bool
	UnitPersonnel       int
	UnitNames           []string
	PersonnelCount      int
	OnScenePersonnel    []string
	NotOnScenePersonnel []string
}

// PersonnelRecord holds participation stats for one person.
type PersonnelRecord struct {
	Name       string
	TotalCalls int
	Pct        float64
	OnScene    int
	NotOnScene int
}
