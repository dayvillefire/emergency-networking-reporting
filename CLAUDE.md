# CLAUDE.md

## Project

Emergency Networking Reporting — Go API wrapper (`enapi/`) and Bubble Tea reporting TUI (`cmd/report/`).

Base API URL: `https://app.emergencynetworking.com/department-api`
Auth: Bearer token from `.env` as `API_TOKEN`
Module: `github.com/dayvillefire/emergency-networking-reporting`

## Build & Run

```
make                  # Build Linux + Windows binaries
go build ./...        # Check compilation
go run ./cmd/report   # Launch reporting TUI
go run ./cmd/test     # Integration test (exercises all GET endpoints)
```

## Architecture

```
enapi/
  enapi.go              Client struct, NewClient, functional options, HTTP helpers
  types.go              PaginatedResponse[T], Time, Date, IntBool, FlexString, query options
  errors.go             APIError type
  apparatus.go          Apparatus CRUD
  users.go              ListUsers
  crew_schedules.go     CrewSchedule paginated list + create
  dispatch_tickets.go   DispatchTicket CRUD
  v_endpoints.go        10 NERIS/advanced endpoints + all response types

cmd/report/
  main.go               Entry point, token reading
  stats.go              ReportStats struct, 9 compute* functions, classification
  tui.go                Bubble Tea model (7 states), spinner, picklists
  html.go               HTML template, GenerateHTML, SaveHTML, displayName
```

## Key Patterns

- **Client**: `enapi.NewClient(enapi.WithToken(t), enapi.WithHTTPClient(...))`
- **Pagination**: `client.ListNerisIncidents(ctx, enapi.VWithPerPage(100), enapi.VWithPage(n))`
- **Query options**: `QueryOption` / `VQueryOption` are `func(url.Values)`
- **Custom types**: `IntBool` (0/1/True/False), `FlexString` (string or number), `Time` (flexible time format unmarshaling)
- **Statistics**: All compute functions in `cmd/report/stats.go`; `isEMS()` classifies by `IncidentDispatchedAs` prefix `[M]` and medical keywords

## Dependencies

- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/bubbles` — spinner component
- `github.com/charmbracelet/lipgloss` — terminal styling
- No external dependencies in `enapi/` (stdlib only)
