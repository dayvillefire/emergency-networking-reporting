package shared

import (
	"os"
	"strings"
)

// ReadToken reads the API token from a .env file in the working directory,
// falling back to the API_TOKEN environment variable.
func ReadToken() string {
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
