# Emergency Networking Reporting

API wrapper and reporting tools for [Emergency Networking](https://app.emergencynetworking.com) department data.

## Components

### enapi — Go API Client

Zero-dependency Go wrapper for the Emergency Networking Department API.

```go
client := enapi.NewClient(enapi.WithToken(token))
apparatus, err := client.ListApparatus(ctx)
incidents, err := client.ListNerisIncidents(ctx, enapi.VWithPerPage(100))
```

Covers all 18 endpoints: apparatus, crew schedules, dispatch tickets, users, and 10 NERIS/advanced endpoints.

### en-report — CLI Reporting Tool

Interactive Bubble Tea TUI that generates HTML reports for a selected time period.

```
make build
./en-report
```

The TUI walks through department selection, period selection (year / quarter / month), then fetches incident data and produces a styled HTML and PDF report covering call volume, response times, personnel, mutual aid, concurrency, special incident types, and unit responses.

### en-responder-stats — Responder Participation CLI

Headless CLI that outputs responder call numbers and percentages across three rolling windows (last month, quarter, year), tranched into >30%, >20%, and >10% participation.

```
make build-responder
./en-responder-stats                          # text table output
./en-responder-stats --csv                    # CSV output
./en-responder-stats --department "Station 1" # filter by station
```

## Setup

Copy `.env.sample` to `.env` and set your API token:

```
API_TOKEN=eyJ...
```

## Build

```
make          # Linux + Windows binaries
make build    # Linux only
make windows  # Windows cross-compile
make clean    # Remove binaries
```

## Project Structure

```
enapi/                API client library
internal/shared/      Shared utilities (token, normalization, personnel stats)
cmd/report/           Reporting TUI (stats, html, pdf, tui, main)
cmd/responder-stats/  Responder participation CLI
cmd/test/             Integration test harness
```
