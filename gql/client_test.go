package gql

import (
	"strings"
	"testing"
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
