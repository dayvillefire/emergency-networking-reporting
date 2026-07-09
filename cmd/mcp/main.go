package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/dayvillefire/emergency-networking-reporting/enapi"
	"github.com/dayvillefire/emergency-networking-reporting/internal/shared"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	readOnly := flag.Bool("read-only", false, "expose only read tools (disable create/update)")
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

	server := mcp.NewServer(&mcp.Implementation{Name: "emergency-networking", Version: "v1.0.0"}, nil)
	registerTools(server, client, !*readOnly)

	// Serve MCP over stdin/stdout until the client disconnects. Diagnostics go to
	// stderr only; stdout is reserved for JSON-RPC frames.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
