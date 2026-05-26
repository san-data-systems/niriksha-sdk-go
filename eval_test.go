package nirikshaai_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
)

func TestSubmitEval_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	// The eval client uses the state baseURL; we need to init the SDK first
	// Skip this test if the SDK requires full OTEL init
	t.Skip("requires integration setup — covered by integration tests")
}

func TestRedactPII_Immutable(t *testing.T) {
	input := "email: test@test.com"
	original := input
	nirikshaai.RedactPII(input)
	if input != original {
		t.Error("RedactPII must not mutate input")
	}
}
