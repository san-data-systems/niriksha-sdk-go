package nirikshaai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/san-data-systems/niriksha-sdk-go/internal/logger"
)

// EvalInput is a single evaluation result.
type EvalInput struct {
	TraceID      string            `json:"trace_id"`
	MetricName   string            `json:"metric_name"`
	Score        float64           `json:"score"`
	Label        string            `json:"label,omitempty"`
	Explanation  string            `json:"explanation,omitempty"`
	EvalType     string            `json:"eval_type,omitempty"`
	ExperimentID string            `json:"experiment_id,omitempty"` // optional — groups evals into an experiment
	Confidence   float64           `json:"confidence,omitempty"`    // optional — evaluator confidence in [0,1], 0 means unset
	Metadata     map[string]string `json:"metadata,omitempty"`      // optional — arbitrary context key/values
	EvalTime     time.Time         `json:"eval_time,omitempty"`     // optional — when the eval was performed; zero = server time
}

// evalPayload is used when building the JSON body to control conditional fields.
type evalPayload struct {
	TraceID      string            `json:"trace_id"`
	MetricName   string            `json:"metric_name"`
	Score        float64           `json:"score"`
	Label        string            `json:"label,omitempty"`
	Explanation  string            `json:"explanation,omitempty"`
	EvalType     string            `json:"eval_type,omitempty"`
	ExperimentID string            `json:"experiment_id,omitempty"`
	Confidence   float64           `json:"confidence,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	EvalTime     string            `json:"eval_time,omitempty"`
}

func toPayload(e EvalInput) evalPayload {
	p := evalPayload{
		TraceID:     e.TraceID,
		MetricName:  e.MetricName,
		Score:       e.Score,
		Label:       e.Label,
		Explanation: e.Explanation,
		EvalType:    e.EvalType,
	}
	if e.ExperimentID != "" {
		p.ExperimentID = e.ExperimentID
	}
	if e.Confidence > 0 {
		p.Confidence = e.Confidence
	}
	if e.Metadata != nil {
		p.Metadata = e.Metadata
	}
	if !e.EvalTime.IsZero() {
		p.EvalTime = e.EvalTime.UTC().Format(time.RFC3339)
	}
	return p
}

var _evalClient = &http.Client{Timeout: 15 * time.Second}

// SubmitEval submits a single evaluation result for a trace.
// The project is resolved from the API key — no org/project IDs needed.
func SubmitEval(ctx context.Context, e EvalInput) error {
	if e.EvalType == "" {
		e.EvalType = "rule_based"
	}
	return doPost(ctx, _state.baseURL+"/api/v1/sdk/evals", toPayload(e))
}

// SubmitEvalsBatch submits multiple evals in a single request.
func SubmitEvalsBatch(ctx context.Context, evals []EvalInput) error {
	payloads := make([]evalPayload, len(evals))
	for i := range evals {
		if evals[i].EvalType == "" {
			evals[i].EvalType = "rule_based"
		}
		payloads[i] = toPayload(evals[i])
	}
	return doPost(ctx, _state.baseURL+"/api/v1/sdk/evals/batch", map[string]any{"evals": payloads})
}

func doPost(ctx context.Context, url string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	const maxAttempts = 3
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			logger.Warn("retrying eval POST", "url", url, "attempt", attempt+1, "max_attempts", maxAttempts, "err", lastErr)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", _state.apiKey)

		resp, err := _evalClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("nirikshaai: eval POST %s status %d", url, resp.StatusCode)
			continue
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("nirikshaai: eval POST %s status %d", url, resp.StatusCode)
		}
		return nil
	}

	logger.Error("eval POST failed", "url", url, "attempts", maxAttempts, "err", lastErr)
	return lastErr
}
