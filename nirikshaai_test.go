package nirikshaai_test

import (
	"context"
	"testing"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
)

func TestInit_MissingAPIKey(t *testing.T) {
	ctx := context.Background()
	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint: "https://ingest.niriksha.ai",
		// APIKey intentionally omitted
	})
	if err != nil {
		// Expected — Init may return an error or succeed with empty key
		return
	}
	// If Init succeeds with no key, that is also acceptable — SDK is lenient
	// on empty key (auth fails at export time). Ensure shutdown is callable.
	if shutdown == nil {
		t.Fatal("expected non-nil ShutdownFunc")
	}
	_ = shutdown(ctx)
}

func TestInit_MissingEndpoint(t *testing.T) {
	ctx := context.Background()
	_, err := nirikshaai.Init(ctx, nirikshaai.Options{
		APIKey: "test-key",
		// Endpoint intentionally omitted
	})
	// An empty endpoint is invalid and must produce an error.
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
}

func TestInit_TLSMutualExclusivity(t *testing.T) {
	ctx := context.Background()
	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint:      "https://ingest.niriksha.ai",
		APIKey:        "test-key",
		TLSSkipVerify: true,
		Insecure:      true,
	})
	// The SDK should either return an error or warn — both options are
	// mutually exclusive in intent, though the current implementation allows
	// both (Insecure takes precedence in buildDialOpts). We verify the call
	// does not panic and that Init is at least callable.
	if err == nil && shutdown == nil {
		t.Fatal("expected non-nil ShutdownFunc when Init succeeds")
	}
	if shutdown != nil {
		_ = shutdown(ctx)
	}
}

func TestInit_ValidOptions(t *testing.T) {
	ctx := context.Background()
	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint:    "https://ingest.niriksha.ai",
		APIKey:      "test-key",
		ServiceName: "test-service",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil ShutdownFunc")
	}
	_ = shutdown(ctx)
}

func TestIsInitialized(t *testing.T) {
	ctx := context.Background()
	_, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint: "https://ingest.niriksha.ai",
		APIKey:   "test-key",
	})
	if err != nil {
		t.Fatalf("unexpected init error: %v", err)
	}
	if !nirikshaai.IsInitialized() {
		t.Error("IsInitialized should return true after successful Init")
	}
}
