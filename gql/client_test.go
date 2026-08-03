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
