package main

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>{{displayName .DeptName}} Report — {{displayName .PeriodLabel}}</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; color: #1a1a2e; background: #f8f9fa; padding: 2rem; }
  header { text-align: center; margin-bottom: 2rem; padding-bottom: 1rem; border-bottom: 3px solid #c41e3a; }
  header h1 { font-size: 1.8rem; color: #c41e3a; }
  header p { color: #666; margin-top: 0.25rem; }
  .summary { display: grid; grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 1rem; margin-bottom: 2rem; }
  .card { background: #fff; border-radius: 8px; padding: 1.25rem; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
  .card h3 { font-size: 1rem; color: #666; margin-bottom: 0.5rem; text-transform: uppercase; letter-spacing: 0.5px; }
  .card .value { font-size: 2rem; font-weight: 700; color: #c41e3a; }
  .card .detail { font-size: 0.85rem; color: #888; }
  section { background: #fff; border-radius: 8px; padding: 1.5rem; margin-bottom: 1.5rem; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
  section h2 { font-size: 1.25rem; color: #c41e3a; margin-bottom: 1rem; border-bottom: 2px solid #eee; padding-bottom: 0.5rem; }
  table { width: 100%; border-collapse: collapse; }
  th, td { text-align: left; padding: 0.5rem 0.75rem; border-bottom: 1px solid #eee; }
  th { font-weight: 600; color: #555; font-size: 0.85rem; text-transform: uppercase; }
  td { font-size: 0.95rem; }
  .bar { display: inline-block; height: 8px; border-radius: 4px; background: #c41e3a; vertical-align: middle; margin-right: 0.5rem; }
  .bar-bg { display: inline-block; width: 100px; height: 8px; border-radius: 4px; background: #eee; vertical-align: middle; margin-right: 0.5rem; }
  footer { text-align: center; color: #999; font-size: 0.85rem; margin-top: 2rem; }
  @media print { body { padding: 0; } .card, section { box-shadow: none; border: 1px solid #ddd; } }
</style>
</head>
<body>
<header>
  <h1>{{displayName .DeptName}}</h1>
  <p>{{displayName .PeriodLabel}} &middot; Generated {{.GeneratedAt.Format "January 2, 2006"}}</p>
</header>

<div class="summary">
  <div class="card">
    <h3>Total Calls</h3>
    <div class="value">{{.TotalCalls}}</div>
  </div>
  <div class="card">
    <h3>EMS Calls</h3>
    <div class="value">{{.EMSCount}}</div>
    <div class="detail">{{printf "%.1f" .EMSPct}}%</div>
  </div>
  <div class="card">
    <h3>Fire Calls</h3>
    <div class="value">{{.FireCount}}</div>
    <div class="detail">{{printf "%.1f" .FirePct}}%</div>
  </div>
  <div class="card">
    <h3>Avg Personnel</h3>
    <div class="value">{{printf "%.1f" .AvgPersonnel}}</div>
    <div class="detail">per call</div>
  </div>
</div>

<section>
  <h2>Response Times</h2>
  <table>
    <tr><th>Metric</th><th>Average (trimmed)</th></tr>
    <tr><td>Dispatch to En Route</td><td>{{formatDuration .DispatchToEnRouteAvg}}</td></tr>
    <tr><td>Dispatch to Arrival</td><td>{{formatDuration .DispatchToArrivalAvg}}</td></tr>
    <tr><td>Time on Scene</td><td>{{formatDuration .TimeOnSceneAvg}}</td></tr>
  </table>
</section>

<section>
  <h2>Concurrent Calls</h2>
  <p>{{.ConcurrentCount}} of {{.TotalCalls}} calls overlapped ({{printf "%.1f" .ConcurrentPct}}%)</p>
</section>

<section>
  <h2>Mutual Aid</h2>
  <table>
    <tr><th>Direction</th><th>Count</th><th>%</th></tr>
    <tr><td>Given</td><td>{{.MutualAidGiven}}</td><td>{{printf "%.1f" .MutualAidGivenPct}}%</td></tr>
    <tr><td>Received</td><td>{{.MutualAidReceived}}</td><td>{{printf "%.1f" .MutualAidReceivedPct}}%</td></tr>
  </table>

  <h3 style="margin-top: 1.25rem;">Given &mdash; by District</h3>
  <table>
    <tr><th>District</th><th>Count</th></tr>
    {{range $district, $count := .MutualAidGivenDistricts}}
    <tr><td>{{$district}}</td><td>{{$count}}</td></tr>
    {{end}}
  </table>
</section>

<section>
  <h2>EMS Mutual Aid by District</h2>
  <p>Total EMS Mutual Aid Calls: {{.EMSMutualAidTotal}}</p>
  <table>
    <tr><th>District</th><th>Count</th></tr>
    {{range $district, $count := .EMSMutualAidDistricts}}
    <tr><td>{{$district}}</td><td>{{$count}}</td></tr>
    {{end}}
  </table>
</section>

<section>
  <h2>Calls by Shift</h2>
  <table>
    <tr><th>Shift</th><th>Count</th><th>%</th></tr>
    {{range $shift, $count := .ShiftCounts}}
    <tr><td>{{$shift}}</td><td>{{$count}}</td><td>{{printf "%.1f" (percentage $count $.TotalCalls)}}%</td></tr>
    {{end}}
  </table>
</section>

<section>
  <h2>Special Incident Types</h2>
  <table>
    <tr><th>Type</th><th>Count</th><th>%</th></tr>
    <tr><td>Hazardous Materials</td><td>{{.HazmatCount}}</td><td>{{printf "%.1f" .HazmatPct}}%</td></tr>
    <tr><td>Motor Vehicle Accidents</td><td>{{.MVACount}}</td><td>{{printf "%.1f" .MVAPct}}%</td></tr>
    <tr><td>Carbon Monoxide Emergencies</td><td>{{.COCount}}</td><td>{{printf "%.1f" .COPct}}%</td></tr>
  </table>
</section>

<section>
  <h2>Special Unit Responses</h2>
  <table>
    <tr><th>Unit</th><th>Count</th><th>%</th></tr>
    <tr><td>Drone Team (UAV163)</td><td>{{.DroneTeamCount}}</td><td>{{printf "%.1f" .DroneTeamPct}}%</td></tr>
    <tr><td>Rehab Team (S263)</td><td>{{.RehabTeamCount}}</td><td>{{printf "%.1f" .RehabTeamPct}}%</td></tr>
  </table>
</section>

<footer>
  Generated by Emergency Networking Reporting Tool
</footer>
</body>
</html>`

var funcMap = template.FuncMap{
	"formatDuration": func(seconds float64) string {
		if seconds <= 0 {
			return "N/A"
		}
		d := time.Duration(seconds) * time.Second
		if d < time.Minute {
			return fmt.Sprintf("%ds", int(d.Seconds()))
		}
		if d < time.Hour {
			m := int(d.Minutes())
			s := int(d.Seconds()) % 60
			return fmt.Sprintf("%dm %ds", m, s)
		}
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", h, m)
	},
	"percentage": func(count, total int) float64 {
		if total == 0 {
			return 0
		}
		return float64(count) / float64(total) * 100
	},
	"displayName": func(name string) string {
		return strings.ReplaceAll(name, "_", " ")
	},
}

// GenerateHTML renders the report as HTML and returns it as a string.
func GenerateHTML(s *ReportStats, periodLabel string) (string, error) {
	data := struct {
		*ReportStats
		PeriodLabel string
	}{s, periodLabel}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

// SaveHTML writes the HTML report to a file named <dept>_<period>.html.
func SaveHTML(s *ReportStats, periodLabel string) (string, error) {
	html, err := GenerateHTML(s, periodLabel)
	if err != nil {
		return "", err
	}
	filename := sanitizeFilename(s.DeptName) + "_" + sanitizeFilename(periodLabel) + ".html"
	if err := os.WriteFile(filename, []byte(html), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return filename, nil
}

func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		" ", "_", "/", "_", "\\", "_", ":", "_",
		"*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_",
	)
	return replacer.Replace(name)
}
