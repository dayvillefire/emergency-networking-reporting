package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dayvillefire/emergency-networking-reporting/enapi"
)

func main() {
	token := readToken()
	if token == "" {
		fmt.Fprintln(os.Stderr, "API_TOKEN not found in .env or environment")
		os.Exit(1)
	}

	client := enapi.NewClient(
		enapi.WithToken(token),
		enapi.WithHTTPClient(&http.Client{Timeout: 30 * time.Second}),
	)

	ctx := context.Background()
	opts := []enapi.VQueryOption{enapi.VWithPerPage(5)}

	tests := []struct {
		name string
		fn   func() (any, error)
	}{
		{"GET /apparatus", func() (any, error) { return client.ListApparatus(ctx) }},
		{"GET /apparatus/{apparatus}", func() (any, error) {
			list, err := client.ListApparatus(ctx)
			if err != nil {
				return nil, err
			}
			if len(list) == 0 {
				return nil, fmt.Errorf("no apparatus found to fetch by ID")
			}
			return client.GetApparatus(ctx, list[0].ID)
		}},
		{"GET /crew-schedules", func() (any, error) {
			return client.ListCrewSchedules(ctx, enapi.WithPerPage(2))
		}},
		{"GET /dispatch-tickets", func() (any, error) { return client.ListDispatchTickets(ctx) }},
		{"GET /users", func() (any, error) { return client.ListUsers(ctx) }},
		{"GET /v/events", func() (any, error) { return client.ListEvents(ctx, opts...) }},
		{"GET /v/fire-billing", func() (any, error) { return client.ListFireBilling(ctx, opts...) }},
		{"GET /v/hydrants", func() (any, error) { return client.ListHydrants(ctx, opts...) }},
		{"GET /v/incidents", func() (any, error) { return client.ListIncidents(ctx, opts...) }},
		{"GET /v/inspections", func() (any, error) { return client.ListInspections(ctx, opts...) }},
		{"GET /v/inventory", func() (any, error) { return client.ListInventory(ctx, opts...) }},
		{"GET /v/neris-billing", func() (any, error) { return client.ListNerisBilling(ctx, opts...) }},
		{"GET /v/neris-incidents", func() (any, error) { return client.ListNerisIncidents(ctx, opts...) }},
		{"GET /v/properties", func() (any, error) { return client.ListProperties(ctx, opts...) }},
		{"GET /v/training", func() (any, error) { return client.ListTraining(ctx, opts...) }},
	}

	passed, failed := 0, 0
	for _, t := range tests {
		result, err := t.fn()
		if err != nil {
			fmt.Printf("FAIL [%s]: %v\n", t.name, err)
			failed++
			continue
		}
		pretty, _ := json.MarshalIndent(result, "", "  ")
		if len(pretty) > 500 {
			pretty = append(pretty[:500], []byte("\n  ...")...)
		}
		fmt.Printf("OK   [%s]: %s\n", t.name, string(pretty))
		passed++
	}

	fmt.Printf("\n---\nResults: %d passed, %d failed, %d total\n", passed, failed, len(tests))
	if failed > 0 {
		os.Exit(1)
	}
}

func readToken() string {
	// Try .env file first (relative to working directory)
	data, err := os.ReadFile(".env")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "API_TOKEN=") {
				return strings.TrimPrefix(line, "API_TOKEN=")
			}
		}
	}
	// Fall back to environment variable
	return os.Getenv("API_TOKEN")
}
