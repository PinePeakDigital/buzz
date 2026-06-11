package main

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

// TestAPIStatusErrorMessage pins the message format every HTTPClient method now
// relies on after the request-helper refactor: "API returned status N", with
// the trimmed body appended only when the server sent one. Existing per-method
// tests assert this via strings.Contains; this locks the exact format directly.
func TestAPIStatusErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  *apiStatusError
		want string
	}{
		{
			name: "status only when body empty",
			err:  &apiStatusError{status: http.StatusInternalServerError, body: ""},
			want: "API returned status 500",
		},
		{
			name: "status and body when present",
			err:  &apiStatusError{status: http.StatusUnprocessableEntity, body: `{"errors":"bad"}`},
			want: `API returned status 422: {"errors":"bad"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestAPIStatusErrorAsTarget confirms apiStatusError survives fmt.Errorf
// wrapping and is recoverable via errors.As — the mechanism FetchGoal and
// FetchGoalRawJSON use to turn a 404 into "goal not found".
func TestAPIStatusErrorAsTarget(t *testing.T) {
	wrapped := fmt.Errorf("failed to fetch goal: %w", &apiStatusError{status: http.StatusNotFound})

	var se *apiStatusError
	if !errors.As(wrapped, &se) {
		t.Fatalf("errors.As did not recover *apiStatusError from %v", wrapped)
	}
	if se.status != http.StatusNotFound {
		t.Errorf("recovered status = %d, want %d", se.status, http.StatusNotFound)
	}
}
