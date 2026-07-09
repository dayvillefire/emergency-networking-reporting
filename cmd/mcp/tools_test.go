package main

import (
	"net/url"
	"testing"
)

func TestVOpts(t *testing.T) {
	opts, err := vOpts(listVInput{Page: 2, PerPage: 50, UpdatedAfter: "2026-01-02T15:04:05Z", Status: "open"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := url.Values{}
	for _, o := range opts {
		o(got)
	}
	want := map[string]string{
		"page":         "2",
		"perPage":      "50",
		"updatedAfter": "2026-01-02T15:04:05Z",
		"status":       "open",
	}
	for k, v := range want {
		if got.Get(k) != v {
			t.Errorf("%s = %q, want %q", k, got.Get(k), v)
		}
	}
}

func TestVOptsBadTimestamp(t *testing.T) {
	if _, err := vOpts(listVInput{UpdatedAfter: "not-a-timestamp"}); err == nil {
		t.Fatal("expected error for malformed updated_after")
	}
}

func TestVOptsEmpty(t *testing.T) {
	opts, err := vOpts(listVInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) != 0 {
		t.Errorf("expected no options, got %d", len(opts))
	}
}
