package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dayvillefire/emergency-networking-reporting/enapi"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// listVInput is the shared argument shape for the paginated v/ endpoints.
type listVInput struct {
	Page         int    `json:"page,omitempty" jsonschema:"page number, 1-based"`
	PerPage      int    `json:"per_page,omitempty" jsonschema:"records per page"`
	UpdatedAfter string `json:"updated_after,omitempty" jsonschema:"only records updated after this RFC3339 timestamp"`
	Status       string `json:"status,omitempty" jsonschema:"filter by status"`
}

// vOpts turns listVInput into enapi VQueryOptions, validating the timestamp.
func vOpts(in listVInput) ([]enapi.VQueryOption, error) {
	var opts []enapi.VQueryOption
	if in.Page > 0 {
		opts = append(opts, enapi.VWithPage(in.Page))
	}
	if in.PerPage > 0 {
		opts = append(opts, enapi.VWithPerPage(in.PerPage))
	}
	if in.UpdatedAfter != "" {
		t, err := time.Parse(time.RFC3339, in.UpdatedAfter)
		if err != nil {
			return nil, fmt.Errorf("updated_after must be RFC3339: %w", err)
		}
		opts = append(opts, enapi.VWithUpdatedAfter(t))
	}
	if in.Status != "" {
		opts = append(opts, enapi.VWithStatus(in.Status))
	}
	return opts, nil
}

// emptyInput is the argument shape for read tools that take no parameters.
type emptyInput struct{}

// crewInput is the argument shape for list_crew_schedules.
type crewInput struct {
	Page    int    `json:"page,omitempty" jsonschema:"page number, 1-based"`
	PerPage int    `json:"per_page,omitempty" jsonschema:"records per page"`
	Start   string `json:"start,omitempty" jsonschema:"filter schedules from this RFC3339 timestamp"`
	End     string `json:"end,omitempty" jsonschema:"filter schedules up to this RFC3339 timestamp"`
}

// getApparatusInput is the argument shape for get_apparatus.
type getApparatusInput struct {
	ID string `json:"id" jsonschema:"apparatus ID"`
}

// registerTools wires every enapi client method to an MCP tool. Mutations are
// registered only when allowWrites is true.
func registerTools(s *mcp.Server, client *enapi.Client, allowWrites bool) {
	// Paginated v/ endpoints (shared listVInput).
	addListV(s, "list_events", "List scheduled events.", client.ListEvents)
	addListV(s, "list_fire_billing", "List fire billing records.", client.ListFireBilling)
	addListV(s, "list_hydrants", "List hydrants.", client.ListHydrants)
	addListV(s, "list_incidents", "List legacy incident reports.", client.ListIncidents)
	addListV(s, "list_inspections", "List property inspections.", client.ListInspections)
	addListV(s, "list_inventory", "List inventory items.", client.ListInventory)
	addListV(s, "list_neris_billing", "List NERIS billing records.", client.ListNerisBilling)
	addListV(s, "list_neris_incidents", "List NERIS incident reports.", client.ListNerisIncidents)
	addListV(s, "list_properties", "List property records.", client.ListProperties)
	addListV(s, "list_training", "List training records.", client.ListTraining)

	// No-argument slice reads.
	addListAll(s, "list_apparatus", "List all apparatus/vehicles.", client.ListApparatus)
	addListAll(s, "list_users", "List all department users/personnel.", client.ListUsers)
	addListAll(s, "list_dispatch_tickets", "List all dispatch tickets.", client.ListDispatchTickets)

	// One-off reads.
	mcp.AddTool(s, &mcp.Tool{Name: "list_crew_schedules", Description: "List crew schedules, optionally filtered by date range."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in crewInput) (*mcp.CallToolResult, any, error) {
			var opts []enapi.QueryOption
			if in.Page > 0 {
				opts = append(opts, enapi.WithPage(in.Page))
			}
			if in.PerPage > 0 {
				opts = append(opts, enapi.WithPerPage(in.PerPage))
			}
			if in.Start != "" {
				t, err := time.Parse(time.RFC3339, in.Start)
				if err != nil {
					return nil, nil, fmt.Errorf("start must be RFC3339: %w", err)
				}
				opts = append(opts, enapi.WithStart(t))
			}
			if in.End != "" {
				t, err := time.Parse(time.RFC3339, in.End)
				if err != nil {
					return nil, nil, fmt.Errorf("end must be RFC3339: %w", err)
				}
				opts = append(opts, enapi.WithEnd(t))
			}
			out, err := client.ListCrewSchedules(ctx, opts...)
			return nil, out, err
		})

	mcp.AddTool(s, &mcp.Tool{Name: "get_apparatus", Description: "Get a single apparatus by ID."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in getApparatusInput) (*mcp.CallToolResult, any, error) {
			out, err := client.GetApparatus(ctx, in.ID)
			return nil, out, err
		})

	if !allowWrites {
		return
	}

	// Mutations. enapi request types embed enapi.Time (a struct wrapping
	// time.Time), which the SDK's schema inference rejects for embedded structs
	// (it panics). So mutation tools take a free-form JSON object, decoded into
	// the typed request via addMutation (enapi.Time's UnmarshalJSON parses the
	// timestamp strings). Field shapes are documented in each description.
	addMutation(s, "create_apparatus",
		"Create an apparatus. Object fields: name (string, required), resource_code (string, required), call_sign (string), identifier (string), identifier_type (string), vehicle_unit_number (string).",
		client.CreateApparatus)
	addMutation(s, "create_crew_schedule",
		"Create a crew schedule. Object fields: start, end (RFC3339 timestamps, required); notes (string); equipment (array of {call_sign, primary_action, secondary_action, users:[{personnel_id, start, end, notes}]}).",
		client.CreateCrewSchedule)
	addMutation(s, "create_dispatch_tickets",
		"Create a dispatch ticket. Object fields: incident_number, initial_call (RFC3339), pcr_number, address, city, state (required); optional longitude, latitude, apartment, zip, district, shift, station, call_type, incident_type (int), notes, phone; units:[{unit, notified (RFC3339), en_route, on_scene, at_patient, left_for_destination, at_destination, completed, crew:[{identifier}]}]; narratives:[{written_at (RFC3339), personnel_name, narrative}].",
		client.CreateDispatchTickets)
	addMutation(s, "update_dispatch_ticket",
		"Update a dispatch ticket's unit timestamps. Object fields: incident_number, unit (required); optional RFC3339 timestamps notified, en_route, on_scene, at_patient, left_for_destination, at_destination, completed.",
		client.UpdateDispatchTicket)
}

// addListV registers a paginated v/ endpoint reader.
func addListV[T any](s *mcp.Server, name, desc string, fn func(context.Context, ...enapi.VQueryOption) (*enapi.PaginatedResponse[T], error)) {
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: desc},
		func(ctx context.Context, _ *mcp.CallToolRequest, in listVInput) (*mcp.CallToolResult, any, error) {
			opts, err := vOpts(in)
			if err != nil {
				return nil, nil, err
			}
			out, err := fn(ctx, opts...)
			return nil, out, err
		})
}

// addListAll registers a no-argument slice reader.
func addListAll[T any](s *mcp.Server, name, desc string, fn func(context.Context) ([]T, error)) {
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: desc},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ emptyInput) (*mcp.CallToolResult, any, error) {
			out, err := fn(ctx)
			return nil, out, err
		})
}

// addMutation registers a write tool. Arguments arrive as a free-form JSON
// object and are decoded into the typed request R before the call.
func addMutation[R any, Out any](s *mcp.Server, name, desc string, call func(context.Context, R) (Out, error)) {
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: desc},
		func(ctx context.Context, _ *mcp.CallToolRequest, in map[string]any) (*mcp.CallToolResult, any, error) {
			raw, err := json.Marshal(in)
			if err != nil {
				return nil, nil, err
			}
			var req R
			if err := json.Unmarshal(raw, &req); err != nil {
				return nil, nil, fmt.Errorf("invalid arguments: %w", err)
			}
			out, err := call(ctx, req)
			return nil, out, err
		})
}
