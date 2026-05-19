package main

import (
	"bytes"
	"encoding/json"
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
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
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
  .charts { display: grid; grid-template-columns: repeat(auto-fit, minmax(480px, 1fr)); gap: 1.5rem; margin-bottom: 1.5rem; }
  .chart-box { background: #fff; border-radius: 8px; padding: 1.5rem; box-shadow: 0 1px 3px rgba(0,0,0,0.08); }
  .chart-box h3 { font-size: 1rem; color: #c41e3a; margin-bottom: 0.75rem; }
  .chart-box canvas { max-height: 280px; }
  footer { text-align: center; color: #999; font-size: 0.85rem; margin-top: 2rem; }
  @media print { body { padding: 0; } .card, section, .chart-box { box-shadow: none; border: 1px solid #ddd; } }
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

{{if .ShowCharts}}
<section>
  <h2>Monthly Trends</h2>
  <div class="charts">
    <div class="chart-box"><h3>Call Volume</h3><canvas id="chart-calls"></canvas></div>
    <div class="chart-box"><h3>Average Personnel</h3><canvas id="chart-personnel"></canvas></div>
    <div class="chart-box"><h3>Response Times (seconds)</h3><canvas id="chart-times"></canvas></div>
    <div class="chart-box"><h3>Concurrent Calls</h3><canvas id="chart-concurrent"></canvas></div>
    <div class="chart-box"><h3>Mutual Aid</h3><canvas id="chart-mutual-aid"></canvas></div>
    <div class="chart-box"><h3>Special Incident Types</h3><canvas id="chart-special-types"></canvas></div>
    <div class="chart-box"><h3>Special Unit Responses</h3><canvas id="chart-special-units"></canvas></div>
  </div>
</section>

<script>
const monthly = {{.ChartJSON}};
const months = monthly.map(m => m.Month);
const colors = { ems: '#2563eb', fire: '#c41e3a', dr: '#22c55e', da: '#f59e0b', os: '#8b5cf6', given: '#22c55e', received: '#f59e0b', hazmat: '#ef4444', mva: '#f97316', co: '#6366f1', drone: '#06b6d4', rehab: '#a855f7' };

function mkLine(id, label, data, color) {
  new Chart(document.getElementById(id), {
    type: 'line', data: { labels: months, datasets: [{ label, data, borderColor: color, backgroundColor: color+'20', fill: true, tension: 0.3 }] },
    options: { responsive: true, plugins: { legend: { display: false } }, scales: { y: { beginAtZero: true } } }
  });
}

function mkBar(id, datasets) {
  new Chart(document.getElementById(id), {
    type: 'bar', data: { labels: months, datasets }, options: { responsive: true, scales: { x: { stacked: true }, y: { stacked: true, beginAtZero: true } } }
  });
}

mkBar('chart-calls', [
  { label: 'EMS', data: monthly.map(m => m.EMS), backgroundColor: colors.ems },
  { label: 'Fire', data: monthly.map(m => m.Fire), backgroundColor: colors.fire }
]);
mkLine('chart-personnel', 'Avg Personnel', monthly.map(m => m.AvgPersonnel), '#2563eb');
new Chart(document.getElementById('chart-times'), {
  type: 'line',
  data: { labels: months, datasets: [
    { label: 'Disp→EnRoute', data: monthly.map(m => m.DispatchToEnRoute), borderColor: colors.dr, backgroundColor: colors.dr+'20', fill: true, tension: 0.3 },
    { label: 'Disp→Arrival', data: monthly.map(m => m.DispatchToArrival), borderColor: colors.da, backgroundColor: colors.da+'20', fill: true, tension: 0.3 },
    { label: 'On Scene',     data: monthly.map(m => m.TimeOnScene),     borderColor: colors.os, backgroundColor: colors.os+'20', fill: true, tension: 0.3 }
  ]},
  options: { responsive: true, plugins: { legend: { position: 'bottom' } }, scales: { y: { beginAtZero: true } } }
});
mkLine('chart-concurrent', 'Concurrent Calls', monthly.map(m => m.Concurrent), '#ef4444');
mkBar('chart-mutual-aid', [
  { label: 'Given', data: monthly.map(m => m.MutualAidGiven), backgroundColor: colors.given },
  { label: 'Received', data: monthly.map(m => m.MutualAidReceived), backgroundColor: colors.received }
]);
mkBar('chart-special-types', [
  { label: 'Hazmat', data: monthly.map(m => m.Hazmat), backgroundColor: colors.hazmat },
  { label: 'MVA', data: monthly.map(m => m.MVA), backgroundColor: colors.mva },
  { label: 'CO', data: monthly.map(m => m.CO), backgroundColor: colors.co }
]);
mkBar('chart-special-units', [
  { label: 'Drone (UAV163)', data: monthly.map(m => m.DroneTeam), backgroundColor: colors.drone },
  { label: 'Rehab (S263)', data: monthly.map(m => m.RehabTeam), backgroundColor: colors.rehab }
]);
</script>
{{end}}

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

<section>
  <h2>Personnel Response</h2>
  <p>{{.PersonnelTotalResponding}} total responding members</p>

  <h3 style="margin-top: 1rem;">High Responders (&ge;30%) — {{.PersonnelHighCount}} members</h3>
  {{if .ShowPersonnelDetails}}
  <table>
    <tr><th>Name</th><th>Total Calls</th><th>%</th><th>On Scene</th><th>Not On Scene</th></tr>
    {{range .PersonnelHigh}}
    <tr><td>{{.Name}}</td><td>{{.TotalCalls}}</td><td>{{printf "%.1f" .Pct}}%</td><td>{{.OnScene}}</td><td>{{.NotOnScene}}</td></tr>
    {{end}}
  </table>
  {{end}}

  <h3 style="margin-top: 1rem;">Active Members (&ge;20%) — {{.PersonnelActiveCount}} cumulative members</h3>
  {{if .ShowPersonnelDetails}}
  <table>
    <tr><th>Name</th><th>Total Calls</th><th>%</th><th>On Scene</th><th>Not On Scene</th></tr>
    {{range .PersonnelActive}}
    <tr><td>{{.Name}}</td><td>{{.TotalCalls}}</td><td>{{printf "%.1f" .Pct}}%</td><td>{{.OnScene}}</td><td>{{.NotOnScene}}</td></tr>
    {{end}}
  </table>
  {{end}}

  <h3 style="margin-top: 1rem;">Members in Good Standing (&ge;10%) — {{.PersonnelGoodStandingCount}} cumulative members</h3>
  {{if .ShowPersonnelDetails}}
  <table>
    <tr><th>Name</th><th>Total Calls</th><th>%</th><th>On Scene</th><th>Not On Scene</th></tr>
    {{range .PersonnelGoodStanding}}
    <tr><td>{{.Name}}</td><td>{{.TotalCalls}}</td><td>{{printf "%.1f" .Pct}}%</td><td>{{.OnScene}}</td><td>{{.NotOnScene}}</td></tr>
    {{end}}
  </table>
  {{end}}
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
func GenerateHTML(s *ReportStats, periodLabel string, showCharts bool, showPersonnelDetails bool) (string, error) {
	chartJSON := "[]"
	if showCharts && len(s.Monthly) > 0 {
		b, _ := json.Marshal(s.Monthly)
		chartJSON = string(b)
	}

	data := struct {
		*ReportStats
		PeriodLabel          string
		ShowCharts           bool
		ShowPersonnelDetails bool
		ChartJSON            template.JS
	}{s, periodLabel, showCharts, showPersonnelDetails, template.JS(chartJSON)}

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
func SaveHTML(s *ReportStats, periodLabel string, showCharts bool, showPersonnelDetails bool) (string, error) {
	html, err := GenerateHTML(s, periodLabel, showCharts, showPersonnelDetails)
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
