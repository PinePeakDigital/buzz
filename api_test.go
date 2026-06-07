package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	status, body, err := client.APIRequest(context.Background(), http.MethodGet, "users/me.json", url.Values{"associations": {"true"}})
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

// TestAPIRequestAuthTokenAlwaysWins verifies that a caller-supplied auth_token
// param cannot override the configured credential.
func TestAPIRequestAuthTokenAlwaysWins(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(&Config{Username: "alice", AuthToken: "secret", BaseURL: server.URL})
	if _, _, err := client.APIRequest(context.Background(), http.MethodGet, "users/me.json", url.Values{"auth_token": {"attacker"}}); err != nil {
		t.Fatalf("APIRequest returned error: %v", err)
	}
	if !strings.Contains(gotQuery, "auth_token=secret") {
		t.Errorf("expected stored auth_token to win, got query: %s", gotQuery)
	}
	if strings.Contains(gotQuery, "attacker") {
		t.Errorf("caller-supplied auth_token leaked into request: %s", gotQuery)
	}
}

// TestAPIRequestPathCannotOverrideAuthToken verifies that an auth_token embedded
// in the path's query string cannot smuggle in a second token — the stored
// credential is the only one sent.
func TestAPIRequestPathCannotOverrideAuthToken(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(&Config{Username: "alice", AuthToken: "secret", BaseURL: server.URL})
	if _, _, err := client.APIRequest(context.Background(), http.MethodGet, "users/me.json?auth_token=attacker", nil); err != nil {
		t.Fatalf("APIRequest returned error: %v", err)
	}
	if strings.Contains(gotQuery, "attacker") {
		t.Errorf("path-embedded auth_token leaked into request: %s", gotQuery)
	}
	if strings.Count(gotQuery, "auth_token=") != 1 || !strings.Contains(gotQuery, "auth_token=secret") {
		t.Errorf("expected exactly one auth_token=secret, got query: %s", gotQuery)
	}
}

// TestAPIRequestDELETEUsesQuery verifies that DELETE (like GET) carries params
// in the query string and sends no body or content-type.
func TestAPIRequestDELETEUsesQuery(t *testing.T) {
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
	if _, _, err := client.APIRequest(context.Background(), http.MethodDelete, "users/me/goals/read/datapoints/123.json", url.Values{"foo": {"bar"}}); err != nil {
		t.Fatalf("APIRequest returned error: %v", err)
	}
	if !strings.Contains(gotQuery, "foo=bar") || !strings.Contains(gotQuery, "auth_token=secret") {
		t.Errorf("expected foo and auth_token in query, got: %s", gotQuery)
	}
	if gotBody != "" {
		t.Errorf("expected empty body for DELETE, got: %s", gotBody)
	}
	if gotContentType != "" {
		t.Errorf("expected no content-type for DELETE, got: %s", gotContentType)
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
	if _, _, err := client.APIRequest(context.Background(), http.MethodPost, "users/me/goals/read/datapoints.json", url.Values{"value": {"1"}}); err != nil {
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
		fn           func(method, path string, params url.Values) (int, []byte, error)
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
			fn:         func(m, p string, params url.Values) (int, []byte, error) { return 200, []byte(`{"a":1}`), nil },
			wantExit:   0,
			wantStdout: "\"a\": 1",
			wantMethod: "GET",
			wantPath:   "users/me.json",
		},
		{
			name: "POST with data passes method and params",
			args: []string{"-X", "post", "-d", "value=1", "users/me/goals/read/datapoints.json"},
			fn: func(m, p string, params url.Values) (int, []byte, error) {
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
			fn: func(m, p string, params url.Values) (int, []byte, error) {
				return 200, []byte("plain text"), nil
			},
			wantExit:   0,
			wantStdout: "plain text",
		},
		{
			name: "non-2xx exits nonzero but prints body",
			args: []string{"users/nope.json"},
			fn: func(m, p string, params url.Values) (int, []byte, error) {
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
			name:       "invalid data - no equals",
			args:       []string{"-d", "novalue", "a.json"},
			wantExit:   1,
			wantStderr: "Invalid --data",
		},
		{
			name:       "invalid data - empty key",
			args:       []string{"-d", "=value", "a.json"},
			wantExit:   1,
			wantStderr: "Invalid --data",
		},
		{
			name:     "empty body prints nothing and exits 0",
			args:     []string{"a.json"},
			fn:       func(m, p string, params url.Values) (int, []byte, error) { return 200, []byte{}, nil },
			wantExit: 0,
		},
		{
			name:       "client error",
			args:       []string{"a.json"},
			fn:         func(m, p string, params url.Values) (int, []byte, error) { return 0, nil, io.ErrUnexpectedEOF },
			wantExit:   1,
			wantStderr: "Error:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sawMethod, sawPath string
			var sawParams url.Values
			client := &FakeClient{}
			if tt.fn != nil {
				client.APIRequestFunc = func(method, path string, params url.Values) (int, []byte, error) {
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
			if tt.wantParamKey != "" && sawParams.Get(tt.wantParamKey) != tt.wantParamVal {
				t.Errorf("param %q = %q, want %q", tt.wantParamKey, sawParams.Get(tt.wantParamKey), tt.wantParamVal)
			}
		})
	}
}

// TestRunAPICommandRepeatedData verifies that repeating -d with the same key
// preserves every value rather than dropping earlier ones.
func TestRunAPICommandRepeatedData(t *testing.T) {
	var sawParams url.Values
	client := &FakeClient{
		APIRequestFunc: func(method, path string, params url.Values) (int, []byte, error) {
			sawParams = params
			return 200, []byte(`{}`), nil
		},
	}

	var stdout, stderr strings.Builder
	exit := runAPICommand([]string{"-d", "tags=a", "-d", "tags=b", "x.json"}, client, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0 (stderr: %s)", exit, stderr.String())
	}
	got := sawParams["tags"]
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Errorf("expected tags=[a b], got %v", got)
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
