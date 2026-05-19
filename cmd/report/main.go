package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dayvillefire/emergency-networking-reporting/enapi"
)

func main() {
	flag.Parse()

	token := readToken()
	if token == "" {
		fmt.Fprintln(os.Stderr, "API_TOKEN not found in .env or environment")
		os.Exit(1)
	}

	client := enapi.NewClient(
		enapi.WithToken(token),
		enapi.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
	)

	// Build name mapping from user data (ID/personnel_id → "Last, First")
	nameMap := buildNameMap(client)

	now := time.Now()
	p := tea.NewProgram(newModel(client, now, nameMap), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildNameMap(client *enapi.Client) map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	users, err := client.ListUsers(ctx)
	if err != nil {
		return nil
	}
	m := make(map[string]string, len(users))
	for _, u := range users {
		name := fmt.Sprintf("%s, %s", u.LastName, u.FirstName)
		// Map by both database ID and personnel ID
		if u.PersonnelID != "" {
			m[u.PersonnelID] = name
		}
		m[fmt.Sprintf("%d", u.ID)] = name
	}
	return m
}

func readToken() string {
	data, err := os.ReadFile(".env")
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "API_TOKEN=") {
				return strings.TrimPrefix(line, "API_TOKEN=")
			}
		}
	}
	return os.Getenv("API_TOKEN")
}
