package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dayvillefire/emergency-networking-reporting/enapi"
	"github.com/dayvillefire/emergency-networking-reporting/internal/shared"
)

func main() {
	flag.Parse()

	token := shared.ReadToken()
	if token == "" {
		fmt.Fprintln(os.Stderr, "API_TOKEN not found in .env or environment")
		os.Exit(1)
	}

	client := enapi.NewClient(
		enapi.WithToken(token),
		enapi.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
	)

	nameMap := shared.BuildNameMap(client)

	now := time.Now()
	p := tea.NewProgram(newModel(client, now, nameMap), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
