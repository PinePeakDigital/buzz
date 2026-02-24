package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMakeAPIRequestGET tests a GET request with {user} placeholder replacement and query params
func TestMakeAPIRequestGET(t *testing.T) {
	var capturedURL string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `[{"slug":"pushups"}]`)
	}))
	defer mockServer.Close()

	config := &Config{
		Username:  "alice",
		AuthToken: "tok123",
		BaseURL:   mockServer.URL,
	}

	body, status, err := makeAPIRequest(config, "GET", "/api/v1/users/{user}/goals.json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(capturedURL, "/api/v1/users/alice/goals.json") {
		t.Errorf("path not replaced; URL = %q", capturedURL)
	}
	if !strings.Contains(capturedURL, "auth_token=tok123") {
		t.Errorf("auth_token missing from query string; URL = %q", capturedURL)
	}
	if !strings.Contains(string(body), "pushups") {
		t.Errorf("unexpected body: %s", string(body))
	}
}

// TestMakeAPIRequestGETWithParams tests that extra params are merged into the query string
func TestMakeAPIRequestGETWithParams(t *testing.T) {
	var capturedURL string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{}`)
	}))
	defer mockServer.Close()

	config := &Config{
		Username:  "alice",
		AuthToken: "tok123",
		BaseURL:   mockServer.URL,
	}

	_, _, err := makeAPIRequest(config, "GET", "/api/v1/users/alice/goals.json", []string{"filter=frontburner"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(capturedURL, "filter=frontburner") {
		t.Errorf("extra param missing from query string; URL = %q", capturedURL)
	}
}

// TestMakeAPIRequestPOST tests that POST params are sent in the request body
func TestMakeAPIRequestPOST(t *testing.T) {
	var capturedBody string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		capturedBody = r.PostForm.Encode()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"id":"dp1"}`)
	}))
	defer mockServer.Close()

	config := &Config{
		Username:  "alice",
		AuthToken: "tok123",
		BaseURL:   mockServer.URL,
	}

	_, status, err := makeAPIRequest(config, "POST",
		"/api/v1/users/alice/goals/pushups/datapoints.json",
		[]string{"value=5", "comment=test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(capturedBody, "auth_token=tok123") {
		t.Errorf("auth_token missing from body; body = %q", capturedBody)
	}
	if !strings.Contains(capturedBody, "value=5") {
		t.Errorf("value missing from body; body = %q", capturedBody)
	}
}

// TestMakeAPIRequestInvalidParam tests that a missing '=' in a param returns an error
func TestMakeAPIRequestInvalidParam(t *testing.T) {
	config := &Config{
		Username:  "alice",
		AuthToken: "tok",
		BaseURL:   "http://localhost",
	}
	_, _, err := makeAPIRequest(config, "GET", "/api/v1/test", []string{"badparam"})
	if err == nil {
		t.Fatal("expected error for invalid param, got nil")
	}
	if !strings.Contains(err.Error(), "invalid parameter format") {
		t.Errorf("unexpected error message: %v", err)
	}
}

// TestMakeAPIRequestUserPlaceholder tests {user} replacement in path
func TestMakeAPIRequestUserPlaceholder(t *testing.T) {
	var capturedPath string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		fmt.Fprintln(w, `{}`)
	}))
	defer mockServer.Close()

	config := &Config{
		Username:  "bob",
		AuthToken: "tok",
		BaseURL:   mockServer.URL,
	}

	if _, _, err := makeAPIRequest(config, "GET", "/api/v1/users/{user}/goals.json", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedPath != "/api/v1/users/bob/goals.json" {
		t.Errorf("path = %q, want /api/v1/users/bob/goals.json", capturedPath)
	}
}

// TestMakeAPIRequestNonJSONResponse tests that non-JSON responses are returned as-is
func TestMakeAPIRequestNonJSONResponse(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Internal Server Error")
	}))
	defer mockServer.Close()

	config := &Config{
		Username:  "alice",
		AuthToken: "tok",
		BaseURL:   mockServer.URL,
	}

	body, status, err := makeAPIRequest(config, "GET", "/api/v1/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", status)
	}
	if !strings.Contains(string(body), "Internal Server Error") {
		t.Errorf("unexpected body: %s", string(body))
	}
}

// TestMakeAPIRequestJSONPrettyPrint verifies the response body is valid JSON
func TestMakeAPIRequestJSONPrettyPrint(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"slug":"pushups","title":"Do pushups"}`)
	}))
	defer mockServer.Close()

	config := &Config{
		Username:  "alice",
		AuthToken: "tok",
		BaseURL:   mockServer.URL,
	}

	body, _, err := makeAPIRequest(config, "GET", "/api/v1/users/alice/goals/pushups.json", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Errorf("response is not valid JSON: %v", err)
	}
	if parsed["slug"] != "pushups" {
		t.Errorf("slug = %v, want pushups", parsed["slug"])
	}
}

// TestMakeAPIRequestPUT tests that PUT params are sent in the request body and not in the query string
func TestMakeAPIRequestPUT(t *testing.T) {
	var capturedBody string
	var capturedQuery string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		capturedQuery = r.URL.RawQuery
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		capturedBody = r.PostForm.Encode()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"id":"dp1"}`)
	}))
	defer mockServer.Close()

	config := &Config{
		Username:  "alice",
		AuthToken: "tok123",
		BaseURL:   mockServer.URL,
	}

	_, status, err := makeAPIRequest(config, "PUT",
		"/api/v1/users/alice/goals/pushups/datapoints.json",
		[]string{"value=5", "comment=test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(capturedBody, "auth_token=tok123") {
		t.Errorf("auth_token missing from body; body = %q", capturedBody)
	}
	if !strings.Contains(capturedBody, "value=5") {
		t.Errorf("value missing from body; body = %q", capturedBody)
	}
	if strings.Contains(capturedQuery, "value=5") {
		t.Errorf("value should not be in query string; query = %q", capturedQuery)
	}
}

// TestMakeAPIRequestDELETE tests that DELETE params are added to the query string
func TestMakeAPIRequestDELETE(t *testing.T) {
	var capturedQuery string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"id":"dp1"}`)
	}))
	defer mockServer.Close()

	config := &Config{
		Username:  "alice",
		AuthToken: "tok123",
		BaseURL:   mockServer.URL,
	}

	_, status, err := makeAPIRequest(config, "DELETE",
		"/api/v1/users/alice/goals/pushups/datapoints/dp1.json",
		nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(capturedQuery, "auth_token=tok123") {
		t.Errorf("auth_token missing from query string; query = %q", capturedQuery)
	}
}

// TestMakeAPIRequestPATCH tests that PATCH params are sent in the request body and not in the query string
func TestMakeAPIRequestPATCH(t *testing.T) {
	var capturedBody string
	var capturedQuery string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		capturedQuery = r.URL.RawQuery
		if err := r.ParseForm(); err != nil {
			t.Fatalf("failed to parse form: %v", err)
		}
		capturedBody = r.PostForm.Encode()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"id":"dp1"}`)
	}))
	defer mockServer.Close()

	config := &Config{
		Username:  "alice",
		AuthToken: "tok123",
		BaseURL:   mockServer.URL,
	}

	_, status, err := makeAPIRequest(config, "PATCH",
		"/api/v1/users/alice/goals/pushups/datapoints.json",
		[]string{"value=5", "comment=test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if !strings.Contains(capturedBody, "auth_token=tok123") {
		t.Errorf("auth_token missing from body; body = %q", capturedBody)
	}
	if !strings.Contains(capturedBody, "value=5") {
		t.Errorf("value missing from body; body = %q", capturedBody)
	}
	if strings.Contains(capturedQuery, "value=5") {
		t.Errorf("value should not be in query string; query = %q", capturedQuery)
	}
}

// TestMakeAPIRequestUserPlaceholderSpecialChars tests {user} replacement with URL-unsafe characters
func TestMakeAPIRequestUserPlaceholderSpecialChars(t *testing.T) {
	var capturedRequestURI string
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// RequestURI contains the raw, unmodified request-target (path + query)
		capturedRequestURI = r.RequestURI
		fmt.Fprintln(w, `{}`)
	}))
	defer mockServer.Close()

	config := &Config{
		Username:  "bob smith",
		AuthToken: "tok",
		BaseURL:   mockServer.URL,
	}

	if _, _, err := makeAPIRequest(config, "GET", "/api/v1/users/{user}/goals.json", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// url.PathEscape encodes spaces as %20; verify the path segment was encoded
	if !strings.Contains(capturedRequestURI, "/api/v1/users/bob%20smith/goals.json") {
		t.Errorf("request URI = %q, expected to contain /api/v1/users/bob%%20smith/goals.json", capturedRequestURI)
	}
}
