package main

import (
	"net/http"
	"testing"
)

func TestReplacePort(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		newPort  int
		expected string
	}{
		{
			name:     "replaces existing port",
			host:     "127.0.0.1:3903",
			newPort:  3900,
			expected: "127.0.0.1:3900",
		},
		{
			name:     "appends port when missing",
			host:     "garage.internal",
			newPort:  3900,
			expected: "garage.internal:3900",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := replacePort(tt.host, tt.newPort); got != tt.expected {
				t.Fatalf("replacePort(%q, %d) = %q, want %q", tt.host, tt.newPort, got, tt.expected)
			}
		})
	}
}

func TestIsLifecycleUnsupportedStatus(t *testing.T) {
	tests := []struct {
		name   string
		status int
		want   bool
	}{
		{name: "bad request", status: http.StatusBadRequest, want: true},
		{name: "forbidden", status: http.StatusForbidden, want: true},
		{name: "method not allowed", status: http.StatusMethodNotAllowed, want: true},
		{name: "not implemented", status: http.StatusNotImplemented, want: true},
		{name: "not found", status: http.StatusNotFound, want: false},
		{name: "ok", status: http.StatusOK, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLifecycleUnsupportedStatus(tt.status); got != tt.want {
				t.Fatalf("isLifecycleUnsupportedStatus(%d) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
