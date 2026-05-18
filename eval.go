package nirikshaai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// EvalInput is a single evaluation result.
type EvalInput struct {
	TraceID     string  `json:"trace_id"`
	MetricName  string  `json:"metric_name"`
	Score       float64 `json:"score"`
	Label       string  `json:"label,omitempty"`
	Explanation string  `json:"explanation,omitempty"`
	EvalType    string  `json:"eval_type,omitempty"`
}

// SubmitEval submits a single evaluation result for a trace.
// The project is resolved from the API key — no org/project IDs needed.
func SubmitEval(ctx context.Context, e EvalInput) error {
	if e.EvalType == "" {
		e.EvalType = "rule_based"
	}
	return doPost(ctx, _state.baseURL+"/api/v1/sdk/evals", e)
}

// SubmitEvalsBatch submits multiple evals in a single request.
func SubmitEvalsBatch(ctx context.Context, evals []EvalInput) error {
	for i := range evals {
		if evals[i].EvalType == "" {
			evals[i].EvalType = "rule_based"
		}
	}
	return doPost(ctx, _state.baseURL+"/api/v1/sdk/evals/batch", map[string]any{"evals": evals})
}

func doPost(ctx context.Context, url string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", _state.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("nirikshaai: eval POST %s status %d", url, resp.StatusCode)
	}
	return nil
}
