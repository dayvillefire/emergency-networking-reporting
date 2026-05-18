package enapi

import "fmt"

// APIError represents an API error response.
type APIError struct {
	StatusCode int
	Message    string            `json:"message"`
	Errors     map[string]string `json:"errors"`
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("enapi: %d %s: %v", e.StatusCode, e.Message, e.Errors)
	}
	return fmt.Sprintf("enapi: %d %s", e.StatusCode, e.Message)
}
