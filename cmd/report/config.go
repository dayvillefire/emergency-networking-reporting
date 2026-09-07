package main

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Team describes a special unit/team tracked in the report.
type Team struct {
	Name     string   `yaml:"name"`
	Patterns []string `yaml:"patterns"`
}

// Config is the report configuration loaded from YAML.
type Config struct {
	Teams []Team `yaml:"teams"`
}

const defaultConfigPath = "report-config.yaml"

// teamColorPaletteHex provides display colors for team charts, cycled modulo len.
var teamColorPaletteHex = []string{
	"#06b6d4", "#a855f7", "#22c55e", "#f59e0b",
	"#ef4444", "#6366f1", "#ec4899", "#14b8a6",
}

// teamColorPaletteRGB mirrors teamColorPaletteHex for the PDF renderer.
var teamColorPaletteRGB = [][3]int{
	{6, 182, 212}, {168, 85, 247}, {34, 197, 94}, {245, 158, 11},
	{239, 68, 68}, {99, 102, 241}, {236, 72, 153}, {20, 184, 166},
}

// resolveConfigPath honors, in order: -config flag, REPORT_CONFIG env, then default.
func resolveConfigPath(flagPath string) string {
	if flagPath != "" {
		return flagPath
	}
	if p := os.Getenv("REPORT_CONFIG"); p != "" {
		return p
	}
	return defaultConfigPath
}

// LoadConfig reads and validates the report configuration at path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if len(cfg.Teams) == 0 {
		return nil, fmt.Errorf("config %s: no teams defined", path)
	}
	for i, t := range cfg.Teams {
		if strings.TrimSpace(t.Name) == "" {
			return nil, fmt.Errorf("config %s: teams[%d] missing name", path, i)
		}
		if len(t.Patterns) == 0 {
			return nil, fmt.Errorf("config %s: team %q has no patterns", path, t.Name)
		}
		for j, p := range t.Patterns {
			if strings.TrimSpace(p) == "" {
				return nil, fmt.Errorf("config %s: team %q patterns[%d] is empty", path, t.Name, j)
			}
		}
	}
	return &cfg, nil
}

// teamMatches reports whether unitName matches any of the team's patterns.
// Matching is case-insensitive substring, matching the legacy behavior.
func teamMatches(unitName string, team Team) bool {
	u := strings.ToUpper(unitName)
	for _, p := range team.Patterns {
		if strings.Contains(u, strings.ToUpper(p)) {
			return true
		}
	}
	return false
}

// teamColors returns n colors from the palette, cycling if n exceeds the palette.
func teamColors(n int) []string {
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = teamColorPaletteHex[i%len(teamColorPaletteHex)]
	}
	return out
}
