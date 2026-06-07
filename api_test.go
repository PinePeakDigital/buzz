package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAPIRequestGET verifies that GET requests carry auth_token and params in
// the query string and return the status code and body verbatim.
func TestAPIRequestGET(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"username":"alice"}`)
	}))
	defer server.Close()

	client := NewHTTPClient(&Config{Username: "alice", AuthToken: "secret", BaseURL: server.URL})
	status, body, err := client.APIRequest(context.Background(), http.MethodGet, "users/me.json", map[string]string{"associations": "true"})
	if err != nil {
		t.Fatalf("APIRequest returned error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("expected status 200, got %d", status)
	}
	if string(body) != `{"username":"alice"}` {
		t.Errorf("unexpected body: %s", body)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/api/v1/users/me.json" {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if !strings.Contains(gotQuery, "auth_token=secret") {
		t.Errorf("auth_token missing from query: %s", gotQuery)
	}
	if !strings.Contains(gotQuery, "associations=true") {
		t.Errorf("param missing from query: %s", gotQuery)
	}
}

// TestAPIRequestLeadingSlashAndExistingQuery verifies that a leading slash is
// tolerated and that auth_token is appended to a path that already has a query.
func TestAPIRequestLeadingSlashAndExistingQuery(t *testing.T) {
	var gotPath, gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(&Config{Username: "alice", AuthToken: "secret", BaseURL: server.URL})
	if _, _, err := client.APIRequest(context.Background(), http.MethodGet, "/users/me.json?skinny=true", nil); err != nil {
		t.Fatalf("APIRequest returned error: %v", err)
	}
	if gotPath != "/api/v1/users/me.json" {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if !strings.Contains(gotQuery, "skinny=true") || !strings.Contains(gotQuery, "auth_token=secret") {
		t.Errorf("expected both skinny and auth_token in query, got: %s", gotQuery)
	}
}

// TestAPIRequestPOSTBody verifies that POST sends auth_token and params in the
// urlencoded form body, not the query string.
func TestAPIRequestPOSTBody(t *testing.T) {
	var gotQuery, gotBody, gotContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(&Config{Username: "alice", AuthToken: "secret", BaseURL: server.URL})
	if _, _, err := client.APIRequest(context.Background(), http.MethodPost, "users/me/goals/read/datapoints.json", map[string]string{"value": "1"}); err != nil {
		t.Fatalf("APIRequest returned error: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("expected empty query for POST, got: %s", gotQuery)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("unexpected content type: %s", gotContentType)
	}
	if !strings.Contains(gotBody, "auth_token=secret") || !strings.Contains(gotBody, "value=1") {
		t.Errorf("expected auth_token and value in body, got: %s", gotBody)
	}
}

// TestAPIRequestNon2xxNotError verifies that a non-2xx status is returned to the
// caller rather than turned into an error.
func TestAPIRequestNon2xxNotError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "not found")
	}))
	defer server.Close()

	client := NewHTTPClient(&Config{Username: "alice", AuthToken: "secret", BaseURL: server.URL})
	status, body, err := client.APIRequest(context.Background(), http.MethodGet, "users/nope.json", nil)
	if err != nil {
		t.Fatalf("expected no error for 404, got: %v", err)
	}
	if status != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", status)
	}
	if string(body) != "not found" {
		t.Errorf("unexpected body: %s", body)
	}
}

// TestRunAPICommand exercises the command core against a fake client.
func TestRunAPICommand(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		fn           func(method, path string, params map[string]string) (int, []byte, error)
		wantExit     int
		wantStdout   string // substring; empty means don't check
		wantStderr   string // substring; empty means don't check
		wantMethod   string // asserted inside fn when set
		wantPath     string
		wantParamKey string
		wantParamVal string
	}{
		{
			name:       "simple GET pretty-prints JSON",
			args:       []string{"users/me.json"},
			fn:         func(m, p string, params map[string]string) (int, []byte, error) { return 200, []byte(`{"a":1}`), nil },
			wantExit:   0,
			wantStdout: "\"a\": 1",
			wantMethod: "GET",
			wantPath:   "users/me.json",
		},
		{
			name: "POST with data passes method and params",
			args: []string{"-X", "post", "-d", "value=1", "users/me/goals/read/datapoints.json"},
			fn: func(m, p string, params map[string]string) (int, []byte, error) {
				return 200, []byte(`{}`), nil
			},
			wantExit:     0,
			wantMethod:   "POST",
			wantPath:     "users/me/goals/read/datapoints.json",
			wantParamKey: "value",
			wantParamVal: "1",
		},
		{
			name: "non-JSON body printed raw",
			args: []string{"something.txt"},
			fn: func(m, p string, params map[string]string) (int, []byte, error) {
				return 200, []byte("plain text"), nil
			},
			wantExit:   0,
			wantStdout: "plain text",
		},
		{
			name: "non-2xx exits nonzero but prints body",
			args: []string{"users/nope.json"},
			fn: func(m, p string, params map[string]string) (int, []byte, error) {
				return 404, []byte(`{"error":"x"}`), nil
			},
			wantExit:   1,
			wantStdout: "\"error\": \"x\"",
			wantStderr: "status 404",
		},
		{
			name:       "missing path",
			args:       []string{},
			wantExit:   1,
			wantStderr: "Missing required <path>",
		},
		{
			name:       "too many positionals",
			args:       []string{"a.json", "b.json"},
			wantExit:   1,
			wantStderr: "Too many arguments",
		},
		{
			name:       "invalid method",
			args:       []string{"-X", "BOGUS", "a.json"},
			wantExit:   1,
			wantStderr: "Unsupported method",
		},
		{
			name:       "invalid data",
			args:       []string{"-d", "novalue", "a.json"},
			wantExit:   1,
			wantStderr: "Invalid --data",
		},
		{
			name:       "client error",
			args:       []string{"a.json"},
			fn:         func(m, p string, params map[string]string) (int, []byte, error) { return 0, nil, io.ErrUnexpectedEOF },
			wantExit:   1,
			wantStderr: "Error:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sawMethod, sawPath string
			var sawParams map[string]string
			client := &FakeClient{}
			if tt.fn != nil {
				client.APIRequestFunc = func(method, path string, params map[string]string) (int, []byte, error) {
					sawMethod, sawPath, sawParams = method, path, params
					return tt.fn(method, path, params)
				}
			}

			var stdout, stderr strings.Builder
			exit := runAPICommand(tt.args, client, &stdout, &stderr)

			if exit != tt.wantExit {
				t.Errorf("exit = %d, want %d (stderr: %s)", exit, tt.wantExit, stderr.String())
			}
			if tt.wantStdout != "" && !strings.Contains(stdout.String(), tt.wantStdout) {
				t.Errorf("stdout %q does not contain %q", stdout.String(), tt.wantStdout)
			}
			if tt.wantStderr != "" && !strings.Contains(stderr.String(), tt.wantStderr) {
				t.Errorf("stderr %q does not contain %q", stderr.String(), tt.wantStderr)
			}
			if tt.wantMethod != "" && sawMethod != tt.wantMethod {
				t.Errorf("method = %q, want %q", sawMethod, tt.wantMethod)
			}
			if tt.wantPath != "" && sawPath != tt.wantPath {
				t.Errorf("path = %q, want %q", sawPath, tt.wantPath)
			}
			if tt.wantParamKey != "" && sawParams[tt.wantParamKey] != tt.wantParamVal {
				t.Errorf("param %q = %q, want %q", tt.wantParamKey, sawParams[tt.wantParamKey], tt.wantParamVal)
			}
		})
	}
}

// TestRunAPICommandHelp verifies that --help prints usage and exits 0.
func TestRunAPICommandHelp(t *testing.T) {
	var stdout, stderr strings.Builder
	exit := runAPICommand([]string{"--help"}, &FakeClient{}, &stdout, &stderr)
	if exit != 0 {
		t.Errorf("exit = %d, want 0", exit)
	}
	if !strings.Contains(stdout.String(), "Usage: buzz api") {
		t.Errorf("expected usage on stdout, got: %s", stdout.String())
	}
}
