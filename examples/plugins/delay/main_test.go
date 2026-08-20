package main

import "testing"

func TestFibonacci(t *testing.T) {
	t.Parallel()

	tests := []struct {
		attempt int
		want    string
	}{
		{attempt: 1, want: "50ms"},
		{attempt: 2, want: "50ms"},
		{attempt: 3, want: "100ms"},
		{attempt: 5, want: "250ms"},
	}

	for _, tt := range tests {
		if got := fibonacci(tt.attempt, "50ms"); got != tt.want {
			t.Fatalf("attempt %d: got %s, want %s", tt.attempt, got, tt.want)
		}
	}

	if got := fibonacci(1, "nope"); got != "0s" {
		t.Fatalf("broken step: got %s", got)
	}
}
