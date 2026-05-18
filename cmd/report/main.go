package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		enapi.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
	)

	now := time.Now()
	p := tea.NewProgram(newModel(client, now), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
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
