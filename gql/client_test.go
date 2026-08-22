package gql

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

// With SDT_READONLY=1 set, Execute must reject a mutation before making any
// HTTP request. Dummy credentials are used deliberately: the guard sits ahead
// of request(), so on the happy path no network call is made at all. If the
// guard regresses, Execute falls through to request() and this test fails on
// the error-message assertion rather than with the read-only error.
func TestExecuteBlocksMutationWithSDTReadOnly(t *testing.T) {
	t.Setenv("SDT_READONLY", "1")

	client := NewClient("test", "dummy-token")
	_, err := client.Execute("mutation { metafieldsDelete(metafields: []) { userErrors { message } } }")
	if err == nil {
		t.Fatal("expected error for mutation when SDT_READONLY=1, got nil")
	}

	if !strings.Contains(err.Error(), "Mutation not allowed in read-only mode") {
		t.Fatalf("expected read-only error, got: %v", err)
	}
}

func TestContainsMutation(t *testing.T) {
	if containsMutation("query { shop { name } }") {
		t.Error("query should not be detected as a mutation")
	}

	if !containsMutation("mutation { metafieldsDelete(metafields: []) { userErrors { message } } }") {
		t.Error("mutation should be detected")
	}
}

// testServer returns a Client pointed at a counting httptest server. The
// handler responds with status, body pairs, cycling past the end.
func testServer(t *testing.T, pairs [][2]string) (*Client, *int) {
	t.Helper()

	hits := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i := hits
		hits++
		if i >= len(pairs) {
			i = len(pairs) - 1
		}
		w.WriteHeader(statusCode(t, pairs[i][0]))
		fmt.Fprint(w, pairs[i][1])
	}))
	t.Cleanup(server.Close)

	return &Client{endpoint: server.URL, token: "test-token"}, &hits
}

func statusCode(t *testing.T, status string) int {
	t.Helper()

	code, err := strconv.Atoi(status)
	if err != nil {
		t.Fatalf("invalid status %q", status)
	}
	return code
}

func TestRequestRetriesTransientFailures(t *testing.T) {
	client, hits := testServer(t, [][2]string{
		{"500", "internal error"},
		{"500", "internal error"},
		{"200", `{"data":{"shop":{"name":"test"}}}`},
	})

	_, err := client.Execute("query { shop { name } }")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if *hits != 3 {
		t.Errorf("expected 3 attempts after two 500s, got %d", *hits)
	}
}

func TestRequestExhaustsRetries(t *testing.T) {
	client, hits := testServer(t, [][2]string{{"500", "internal error"}})

	_, err := client.Execute("query { shop { name } }")
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if *hits != maxAttempts {
		t.Errorf("expected %d attempts, got %d", maxAttempts, *hits)
	}
}

func TestRequestHonorsRetryAfter(t *testing.T) {
	client, hits := testServer(t, [][2]string{{"429", "throttled"}})

	_, err := client.Execute("query { shop { name } }")
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if *hits != maxAttempts {
		t.Errorf("expected %d attempts, got %d", maxAttempts, *hits)
	}
}

func TestRequestDoesNotRetryMutation(t *testing.T) {
	client, hits := testServer(t, [][2]string{{"500", "internal error"}})

	_, err := client.Execute("mutation { metafieldsDelete(metafields: []) { userErrors { message } } }")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if *hits != 1 {
		t.Errorf("expected a single attempt for a mutation, got %d", *hits)
	}
}

func TestRequestDoesNotRetryClientError(t *testing.T) {
	client, hits := testServer(t, [][2]string{{"400", "bad request"}})

	_, err := client.Execute("query { shop { name } }")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if *hits != 1 {
		t.Errorf("expected a single attempt for a 400, got %d", *hits)
	}
}

func TestRetryDelay(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}

	resp.Header.Set("Retry-After", "5")
	if got := retryDelay(resp); got != 5*time.Second {
		t.Errorf("expected 5s from Retry-After, got %s", got)
	}

	resp.Header.Set("Retry-After", "not-a-number")
	if got := retryDelay(resp); got != 0 {
		t.Errorf("expected 0 for unparseable Retry-After, got %s", got)
	}

	resp.Header.Del("Retry-After")
	if got := retryDelay(resp); got != 0 {
		t.Errorf("expected 0 without Retry-After, got %s", got)
	}
}

func TestNewClientDefaultAPIVersion(t *testing.T) {
	old := DefaultAPIVersion
	defer func() { DefaultAPIVersion = old }()

	// explicit option wins
	client := NewClient("test", "token", map[string]interface{}{"version": "2026-07"})
	if !strings.HasSuffix(client.endpoint, "/admin/api/2026-07/graphql.json") {
		t.Errorf("explicit version not used: %s", client.endpoint)
	}

	// falls back to DefaultAPIVersion
	DefaultAPIVersion = "2026-04"
	client = NewClient("test", "token")
	if !strings.HasSuffix(client.endpoint, "/admin/api/2026-04/graphql.json") {
		t.Errorf("DefaultAPIVersion not applied: %s", client.endpoint)
	}

	// versionless when neither set
	DefaultAPIVersion = ""
	client = NewClient("test", "token")
	if !strings.HasSuffix(client.endpoint, "/admin/api/graphql.json") {
		t.Errorf("expected versionless endpoint: %s", client.endpoint)
	}
}
