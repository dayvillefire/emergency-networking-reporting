package enapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PaginatedResponse is a generic paginated API response.
type PaginatedResponse[T any] struct {
	Total       int `json:"total"`
	PerPage     int `json:"perPage"`
	CurrentPage int `json:"currentPage"`
	Data        []T `json:"data"`
}

// QueryOption sets a query parameter for paginated non-v/ endpoints (crew-schedules).
type QueryOption func(url.Values)

// WithPerPage sets the number of records per page.
func WithPerPage(n int) QueryOption {
	return func(v url.Values) { v.Set("perPage", strconv.Itoa(n)) }
}

// WithPage sets the page number.
func WithPage(n int) QueryOption {
	return func(v url.Values) { v.Set("page", strconv.Itoa(n)) }
}

// WithStart filters crew schedules from this date onward.
func WithStart(t time.Time) QueryOption {
	return func(v url.Values) { v.Set("start", t.Format(time.RFC3339)) }
}

// WithEnd filters crew schedules up to this date.
func WithEnd(t time.Time) QueryOption {
	return func(v url.Values) { v.Set("end", t.Format(time.RFC3339)) }
}

// VQueryOption sets query parameters for v/ endpoints.
type VQueryOption func(url.Values)

// VWithPerPage sets the number of records per page.
func VWithPerPage(n int) VQueryOption {
	return func(v url.Values) { v.Set("perPage", strconv.Itoa(n)) }
}

// VWithPage sets the page number.
func VWithPage(n int) VQueryOption {
	return func(v url.Values) { v.Set("page", strconv.Itoa(n)) }
}

// VWithUpdatedAfter filters records updated after this date.
func VWithUpdatedAfter(t time.Time) VQueryOption {
	return func(v url.Values) { v.Set("updatedAfter", t.Format(time.RFC3339)) }
}

// VWithStatus filters records by status.
func VWithStatus(status string) VQueryOption {
	return func(v url.Values) { v.Set("status", status) }
}

// Time is a wrapper around time.Time that handles API time formats.
type Time struct{ time.Time }

// UnmarshalJSON implements json.Unmarshaler.
func (t *Time) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		t.Time = time.Time{}
		return nil
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000000Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if parsed, err := time.Parse(f, s); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return nil
}

// IntBool handles JSON booleans represented as 0/1 integers or "True"/"False" strings.
type IntBool bool

// UnmarshalJSON implements json.Unmarshaler.
func (b *IntBool) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		*b = false
		return nil
	}
	switch strings.ToLower(s) {
	case "1", "true":
		*b = true
	case "0", "false":
		*b = false
	default:
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("enapi: cannot unmarshal %q into IntBool", s)
		}
		*b = n != 0
	}
	return nil
}

// FlexString handles JSON fields that can be either strings or numbers.
type FlexString string

// UnmarshalJSON implements json.Unmarshaler.
func (s *FlexString) UnmarshalJSON(data []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*s = FlexString(str)
		return nil
	}
	// Try number
	var num json.Number
	if err := json.Unmarshal(data, &num); err == nil {
		*s = FlexString(num.String())
		return nil
	}
	return fmt.Errorf("enapi: cannot unmarshal %s into FlexString", string(data))
}

// Date is a wrapper around time.Time for date-only fields.
type Date struct{ time.Time }

// UnmarshalJSON implements json.Unmarshaler.
func (d *Date) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	if s == "" || s == "null" {
		d.Time = time.Time{}
		return nil
	}
	formats := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
	}
	for _, f := range formats {
		if parsed, err := time.Parse(f, s); err == nil {
			d.Time = parsed
			return nil
		}
	}
	return nil
}
