package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dayvillefire/emergency-networking-reporting/internal/shared"
)

func TestLoadConfigValid(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(p, []byte("teams:\n  - name: Drone Team\n    patterns: [UAV163]\n  - name: Rehab Team\n    patterns: [S263, REHAB163]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(p)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(cfg.Teams) != 2 || cfg.Teams[1].Patterns[1] != "REHAB163" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	cases := map[string]string{
		"missing file": "",
		"no teams":     "teams: []\n",
		"empty name":   "teams:\n  - name: \"\"\n    patterns: [X]\n",
		"no patterns":  "teams:\n  - name: A\n    patterns: []\n",
	}
	for name, body := range cases {
		dir := t.TempDir()
		p := filepath.Join(dir, "c.yaml")
		if body != "" {
			if err := os.WriteFile(p, []byte(body), 0644); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := LoadConfig(p); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestTeamMatches(t *testing.T) {
	team := Team{Name: "Rehab", Patterns: []string{"S263", "REHAB163"}}
	for _, u := range []string{"s263", "REHAB163", "unit-S263-b", "S263"} {
		if !teamMatches(u, team) {
			t.Errorf("teamMatches(%q) = false, want true", u)
		}
	}
	if teamMatches("ENGINE1", team) {
		t.Error("teamMatches(ENGINE1) = true, want false")
	}
}

func TestComputeSpecialUnits(t *testing.T) {
	teams := []Team{
		{Name: "Drone", Patterns: []string{"UAV163"}},
		{Name: "Rehab", Patterns: []string{"S263", "REHAB163"}},
	}
	incidents := []shared.NormalizedIncident{
		{UnitNames: []string{"UAV163"}},
		{UnitNames: []string{"REHAB163"}},
		{UnitNames: []string{"S263"}},
		{UnitNames: []string{"ENGINE1"}},
	}
	s := &ReportStats{TotalCalls: len(incidents)}
	computeSpecialUnits(s, incidents, teams)
	if len(s.Teams) != 2 {
		t.Fatalf("got %d teams, want 2", len(s.Teams))
	}
	if s.Teams[0].Count != 1 || s.Teams[1].Count != 2 {
		t.Fatalf("counts = %d/%d, want 1/2", s.Teams[0].Count, s.Teams[1].Count)
	}
	if s.Teams[1].Pct != 50.0 {
		t.Fatalf("rehab pct = %v, want 50.0", s.Teams[1].Pct)
	}
}
