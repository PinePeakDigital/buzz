package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMultiClientFetchGoalsDeduplicatesSharedGoals(t *testing.T) {
	serverAlice := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/users/alice/goals.json") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"slug":"shared","title":"Shared Goal"},{"slug":"alice-only","title":"Alice Goal"}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer serverAlice.Close()

	serverBob := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/users/bob/goals.json") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"slug":"shared","title":"Shared Goal"},{"slug":"bob-only","title":"Bob Goal"}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer serverBob.Close()

	cfg := &Config{
		Accounts: []AccountConfig{
			{Username: "alice", AuthToken: "token-a", BaseURL: serverAlice.URL},
			{Username: "bob", AuthToken: "token-b", BaseURL: serverBob.URL},
		},
	}

	goals, err := NewHTTPClient(cfg).FetchGoals(context.Background())
	if err != nil {
		t.Fatalf("FetchGoals() error = %v", err)
	}

	if len(goals) != 3 {
		t.Fatalf("FetchGoals() returned %d goals, want 3 after dedup", len(goals))
	}

	got := map[string]string{}
	for _, goal := range goals {
		got[goal.Slug] = goal.Username
	}

	if got["shared"] != "alice" {
		t.Fatalf("shared goal should come from first configured account, got username %q", got["shared"])
	}
	if got["alice-only"] != "alice" {
		t.Fatalf("alice-only goal username = %q, want alice", got["alice-only"])
	}
	if got["bob-only"] != "bob" {
		t.Fatalf("bob-only goal username = %q, want bob", got["bob-only"])
	}
}

func TestMultiClientFetchGoalRoutesToMatchingAccount(t *testing.T) {
	serverAlice := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/users/alice/goals/bob-goal.json") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer serverAlice.Close()

	serverBob := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/users/bob/goals/bob-goal.json") {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"slug":"bob-goal","title":"Bob Goal"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer serverBob.Close()

	cfg := &Config{
		Accounts: []AccountConfig{
			{Username: "alice", AuthToken: "token-a", BaseURL: serverAlice.URL},
			{Username: "bob", AuthToken: "token-b", BaseURL: serverBob.URL},
		},
	}

	goal, err := NewHTTPClient(cfg).FetchGoal(context.Background(), "bob-goal")
	if err != nil {
		t.Fatalf("FetchGoal() error = %v", err)
	}
	if goal.Username != "bob" {
		t.Fatalf("FetchGoal() username = %q, want bob", goal.Username)
	}
}
