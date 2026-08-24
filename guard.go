package nirikshaai

// Inline AI security guard — check text before it reaches the model.
//
// The rest of this SDK is observability: it records what happened. This file is
// enforcement. It calls the gateway's synchronous /v1/guard endpoint, which
// returns a verdict in single-digit milliseconds, so a prompt injection can be
// refused and a leaked credential stripped before the provider call is made.
//
// Three deliberate differences from how comparable SDKs behave, each because the
// obvious choice is worse:
//
//  1. Redact returns the rewritten text and no error. Only Block returns
//     ErrGuardBlocked. Erroring on both means a customer who asked for PII
//     stripping gets their application broken instead of their data protected.
//
//  2. Fail-open stays the default but stops being silent. A guard outage must not
//     take down the caller's application, so an unreachable server allows the
//     text through — but it warns and increments guard.fail_open every time. A
//     silent fail-open is a security hole wearing a reliability costume: the
//     control appears to work right up until it is needed.
//
//  3. GuardFailSecretsClosed is available and is the mode a security buyer will
//     actually accept. The secret patterns are embedded here, so when the server
//     is unreachable, credential exfiltration is still blocked locally while
//     everything else fails open.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/san-data-systems/niriksha-sdk-go/internal/logger"
)

// GuardAction is what the caller should do with the text.
type GuardAction string

const (
	// GuardAllow means nothing was found.
	GuardAllow GuardAction = "allow"
	// GuardTag means something was found that is not reliable enough to act on.
	// Proceed; record it.
	GuardTag GuardAction = "tag"
	// GuardRedact means sensitive content was found and removed. Proceed with
	// Verdict.SafeText.
	GuardRedact GuardAction = "redact"
	// GuardBlock means the text must not be sent.
	GuardBlock GuardAction = "block"
)

// GuardFailMode is what the guard does when it cannot reach the server.
type GuardFailMode string

const (
	// GuardFailOpen allows everything when the guard is unreachable. Default.
	GuardFailOpen GuardFailMode = "open"
	// GuardFailClosed blocks everything when the guard is unreachable. Correct
	// for a hard compliance boundary, and a guaranteed outage for everyone else.
	GuardFailClosed GuardFailMode = "closed"
	// GuardFailSecretsClosed allows everything except locally-detectable
	// credentials. The only fail mode that is both safe and survivable.
	GuardFailSecretsClosed GuardFailMode = "secrets_closed"
)

// ErrGuardBlocked is returned when the verdict is block.
//
// Wrapped with the rules that fired, so a caller gets the reason from the error
// without a second guard call. Compare with errors.Is.
var ErrGuardBlocked = errors.New("blocked by NirikshaAI guard")

// guardTimeout bounds a guard call. This sits on the caller's critical path: a
// guard that hangs is worse than one that is absent, because the absent one
// fails fast.
const guardTimeout = 3 * time.Second

// maxGuardBatch mirrors the server's cap, so an oversized batch fails here with
// a clear message rather than as a 400 from the gateway.
const maxGuardBatch = 32

// GuardFinding is one detection.
type GuardFinding struct {
	Category   string  `json:"category,omitempty"`
	Severity   string  `json:"severity,omitempty"`
	Rule       string  `json:"rule,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
	Start      int     `json:"start,omitempty"`
	End        int     `json:"end,omitempty"`
	// Local reports that the finding came from this SDK's embedded patterns
	// rather than from the server.
	Local bool `json:"local,omitempty"`
}

// GuardVerdict is the guard's decision about one piece of text.
type GuardVerdict struct {
	Action   GuardAction    `json:"action"`
	Findings []GuardFinding `json:"findings,omitempty"`
	// Reasons names low-precision detections that were recorded but did not
	// influence the action.
	Reasons []string `json:"reasons,omitempty"`
	// Redacted is the rewritten text, present only when Action is redact.
	Redacted     string `json:"redacted,omitempty"`
	RiskScore    int    `json:"risk_score"`
	RiskSeverity string `json:"risk_severity,omitempty"`
	// PolicySource is "project" when an operator's stored AIDR policy was
	// applied, "default" when the product default was. Only "project" is
	// binding.
	PolicySource string `json:"policy_source,omitempty"`
	// PolicyEnforced reports that the block comes from the operator's policy
	// rather than rule precision alone — which also means monitor mode will not
	// lift it.
	PolicyEnforced bool `json:"policy_enforced,omitempty"`
	// FailedOpen reports that the guard was unreachable and the configured fail
	// mode decided the outcome. Never silently true: a warning is logged
	// whenever it is set.
	FailedOpen bool `json:"-"`
}

// SafeText returns the text that is safe to send: the redaction when there was
// one, the original otherwise.
//
// Exists so no caller has to write the equivalent conditional, which is easy to
// get wrong in the direction that forwards the secret. Also returns the original
// when a malformed redact verdict carries no text, rather than silently
// substituting an empty prompt.
func (v GuardVerdict) SafeText(original string) string {
	if v.Action == GuardRedact && v.Redacted != "" {
		return v.Redacted
	}
	return original
}

// ── configuration, set by Init ──────────────────────────────────────────────

var _guard struct {
	url      string
	failMode GuardFailMode
	mode     string
}

var _guardClient = &http.Client{Timeout: guardTimeout}

// configureGuard is called by Init.
func configureGuard(url string, failMode GuardFailMode, mode string) error {
	switch failMode {
	case GuardFailOpen, GuardFailClosed, GuardFailSecretsClosed:
	case "":
		failMode = GuardFailOpen
	default:
		// Rejected at Init rather than defaulted, so a deployment cannot believe
		// it is fail-closed when it is not.
		return fmt.Errorf(
			"nirikshaai: GuardFailMode must be one of %q, %q or %q, got %q",
			GuardFailOpen, GuardFailClosed, GuardFailSecretsClosed, failMode)
	}
	_guard.url = strings.TrimRight(url, "/")
	_guard.failMode = failMode
	_guard.mode = mode
	return nil
}

// GuardConfigured reports whether the guard has a URL to call.
//
// Exported so a caller can branch, rather than discovering the guard is inert
// only from a log line.
func GuardConfigured() bool {
	return _guard.url != "" && _state.apiKey != ""
}

// ── public API ──────────────────────────────────────────────────────────────

// GuardCheck evaluates one prompt or completion.
//
// direction is "input" for a prompt heading to the model or "output" for a
// completion coming back; empty defaults to "input". Injection and jailbreak
// rules apply to input only; secrets and PII to both.
//
// Returns ErrGuardBlocked (wrapped, with the rules that fired) when the verdict
// is block. The verdict is returned in that case too, so a caller can inspect it
// without a second call.
func GuardCheck(ctx context.Context, text, direction string) (GuardVerdict, error) {
	if direction == "" {
		direction = "input"
	}
	v := evaluateGuard(ctx, map[string]any{"text": text, "direction": direction}, text)
	return v, blockedError(v)
}

// GuardCheckTool evaluates a tool call before executing it.
//
// This is the check that can actually prevent an action — an rm -rf, an unscoped
// DELETE, a credential read, an outbound request carrying a key. Observing the
// tool call afterwards cannot.
//
// args may be any JSON-marshalable value, or a pre-serialised string.
func GuardCheckTool(ctx context.Context, name string, args any) (GuardVerdict, error) {
	argsJSON := ""
	switch a := args.(type) {
	case nil:
	case string:
		argsJSON = a
	default:
		if b, err := json.Marshal(a); err == nil {
			argsJSON = string(b)
		} else {
			// Unmarshalable arguments must not silently become an unchecked tool
			// call. The name is still worth checking, and the error says why the
			// arguments were not.
			logger.Warn("guard: could not serialise tool arguments; checking the name only")
		}
	}

	payload := map[string]any{"tool": map[string]any{"name": name, "arguments": argsJSON}}
	// The fallback text for the secrets-closed local check includes the
	// arguments: a credential passed to an outbound tool is the concrete
	// exfiltration path, so it is exactly what must still be inspected when the
	// server is unreachable.
	v := evaluateGuard(ctx, payload, name+"\n"+argsJSON)
	return v, blockedError(v)
}

// GuardBatchItem is one entry in a batch request.
type GuardBatchItem struct {
	Text      string `json:"text"`
	Direction string `json:"direction,omitempty"`
}

// GuardCheckBatch evaluates a whole message array in one request.
//
// A per-string API is an N+1 for a multi-turn conversation, which is every real
// chat application. At most 32 items.
//
// The returned action is the most severe of the set: one blocked message means
// the conversation must not be sent. ErrGuardBlocked is returned when that
// aggregate is block.
func GuardCheckBatch(ctx context.Context, items []GuardBatchItem) (GuardAction, []GuardVerdict, error) {
	if len(items) == 0 {
		return GuardAllow, nil, fmt.Errorf("nirikshaai: GuardCheckBatch requires at least one item")
	}
	if len(items) > maxGuardBatch {
		return GuardAllow, nil, fmt.Errorf(
			"nirikshaai: GuardCheckBatch accepts at most %d items, got %d", maxGuardBatch, len(items))
	}

	if !GuardConfigured() {
		verdicts := make([]GuardVerdict, 0, len(items))
		for _, it := range items {
			verdicts = append(verdicts, failVerdict(it.Text))
		}
		return worstGuardAction(verdicts), verdicts, nil
	}

	type batchItem struct {
		Text      string `json:"text"`
		Direction string `json:"direction,omitempty"`
		Mode      string `json:"mode,omitempty"`
	}
	payload := struct {
		Items []batchItem `json:"items"`
	}{Items: make([]batchItem, 0, len(items))}
	for _, it := range items {
		payload.Items = append(payload.Items, batchItem{
			Text: it.Text, Direction: it.Direction, Mode: _guard.mode,
		})
	}

	var body struct {
		Action  string         `json:"action"`
		Results []GuardVerdict `json:"results"`
	}
	if !postGuard(ctx, _guard.url+"/v1/guard/batch", payload, &body) {
		verdicts := make([]GuardVerdict, 0, len(items))
		for _, it := range items {
			verdicts = append(verdicts, failVerdict(it.Text))
		}
		return worstGuardAction(verdicts), verdicts, nil
	}

	action := GuardAction(body.Action)
	if action == "" {
		action = worstGuardAction(body.Results)
	}
	if action == GuardBlock {
		for _, v := range body.Results {
			if v.Action == GuardBlock {
				return action, body.Results, blockedError(v)
			}
		}
		return action, body.Results, fmt.Errorf("%w: policy", ErrGuardBlocked)
	}
	return action, body.Results, nil
}

// ── internals ───────────────────────────────────────────────────────────────

func blockedError(v GuardVerdict) error {
	if v.Action != GuardBlock {
		return nil
	}
	rules := make([]string, 0, len(v.Findings))
	for _, f := range v.Findings {
		if f.Rule != "" {
			rules = append(rules, f.Rule)
		}
	}
	if len(rules) == 0 {
		return fmt.Errorf("%w: policy", ErrGuardBlocked)
	}
	return fmt.Errorf("%w: %s", ErrGuardBlocked, strings.Join(rules, ", "))
}

func evaluateGuard(ctx context.Context, payload map[string]any, fallbackText string) GuardVerdict {
	if !GuardConfigured() {
		return failVerdict(fallbackText)
	}
	if _guard.mode != "" {
		payload["mode"] = _guard.mode
	}
	var v GuardVerdict
	if !postGuard(ctx, _guard.url+"/v1/guard", payload, &v) {
		return failVerdict(fallbackText)
	}
	if v.Action == "" {
		v.Action = GuardAllow
	}
	return v
}

// postGuard sends one request. Reports whether the guard could be consulted.
//
// Deliberately no retry. This is a synchronous call in front of the caller's LLM
// request: retrying turns a 3-second timeout into a 9-second one, and the fail
// mode is a better answer than a slower one.
func postGuard(ctx context.Context, url string, payload, out any) bool {
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		logger.Warn("guard: refusing non-http(s) URL")
		return false
	}
	b, err := json.Marshal(payload)
	if err != nil {
		logger.Warn("guard: could not encode request")
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		logger.Warn("guard: could not build request")
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", _state.apiKey)

	resp, err := _guardClient.Do(req)
	if err != nil {
		logger.Warn("guard: unreachable: " + err.Error())
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		// 4xx is a client bug — a bad key, a malformed body — and means the guard
		// has never worked, rather than that it is briefly down. Worth the louder
		// level.
		msg := fmt.Sprintf("guard: %s returned %d", url, resp.StatusCode)
		if resp.StatusCode < 500 {
			logger.Error(msg)
		} else {
			logger.Warn(msg)
		}
		return false
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		logger.Warn("guard: could not decode response: " + err.Error())
		return false
	}
	return true
}

// failVerdict applies the configured fail mode.
//
// Never silent. Every fail-open trip logs and is countable, because a security
// control that quietly stops working is worse than one that was never installed
// — the second is at least known to be absent.
func failVerdict(text string) GuardVerdict {
	countGuardFailOpen()

	switch _guard.failMode {
	case GuardFailClosed:
		logger.Warn("guard: unreachable and fail mode is closed — blocking")
		return GuardVerdict{Action: GuardBlock, FailedOpen: true}
	case GuardFailSecretsClosed:
		if findings := LocalSecretFindings(text); len(findings) > 0 {
			logger.Warn(fmt.Sprintf(
				"guard: unreachable; blocking locally on %d secret pattern(s)", len(findings)))
			return GuardVerdict{Action: GuardBlock, Findings: findings, FailedOpen: true}
		}
	case GuardFailOpen:
	}

	logger.Warn(fmt.Sprintf(
		"guard: unreachable — allowing text through (fail mode %q). Text is NOT being checked.",
		_guard.failMode))
	return GuardVerdict{Action: GuardAllow, FailedOpen: true}
}

var _guardFailCounter metric.Int64Counter

// countGuardFailOpen increments guard.fail_open.
//
// The counter is a diagnostic: failing to record it must never turn a guard
// outage into an application error, so every failure here is swallowed after the
// first attempt.
func countGuardFailOpen() {
	if _guardFailCounter == nil {
		c, err := otel.Meter("nirikshaai.guard").Int64Counter(
			"guard.fail_open",
			metric.WithDescription("Guard calls that could not reach the server"),
		)
		if err != nil {
			return
		}
		_guardFailCounter = c
	}
	_guardFailCounter.Add(context.Background(), 1,
		metric.WithAttributes(attribute.String("fail_mode", string(_guard.failMode))))
}

func worstGuardAction(verdicts []GuardVerdict) GuardAction {
	rank := map[GuardAction]int{GuardAllow: 0, GuardTag: 1, GuardRedact: 2, GuardBlock: 3}
	worst := GuardAllow
	for _, v := range verdicts {
		if rank[v.Action] > rank[worst] {
			worst = v.Action
		}
	}
	return worst
}

// ── local secret patterns, for GuardFailSecretsClosed ───────────────────────
//
// A deliberately small, prefix-anchored subset of the server's set. The point is
// not parity — the server has eighteen patterns, entropy gating and a placeholder
// denylist — but that the highest-confidence, zero-false-positive formats are
// still caught with no network call. Every one is a vendor's own key prefix, so a
// match is near-certain and a non-match is cheap.

var localSecretPatterns = []struct {
	rule string
	re   *regexp.Regexp
}{
	{"aws_access_key", regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{"github_token", regexp.MustCompile(`\b(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{36,}\b`)},
	{"github_pat", regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{22,}\b`)},
	{"slack_token", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`)},
	{"stripe_secret_key", regexp.MustCompile(`\b(?:sk|rk)_live_[A-Za-z0-9]{20,}\b`)},
	{"google_api_key", regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`)},
	{"anthropic_api_key", regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{20,}\b`)},
	{"openai_api_key", regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_-]{20,}\b`)},
	{"private_key_block", regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |PGP |DSA )?PRIVATE KEY-----`)},
	{"niriksha_api_key", regexp.MustCompile(`\bnai_(?:plat_)?[A-Za-z0-9]{20,}\b`)},
}

// localSecretPlaceholders are documentation values that match a real pattern.
// Without this the local check would block on a README, and the first person it
// inconveniences would switch the mode off.
var localSecretPlaceholders = map[string]bool{
	"akiaiosfodnn7example": true,
	"aws_access_key_id":    true,
}

// LocalSecretFindings detects embedded secret formats without calling the server.
//
// Exported so a caller can run the same check itself — for example on data it is
// about to log — without depending on the guard being reachable.
func LocalSecretFindings(text string) []GuardFinding {
	if text == "" {
		return nil
	}
	var findings []GuardFinding
	for _, p := range localSecretPatterns {
		for _, loc := range p.re.FindAllStringIndex(text, -1) {
			if localSecretPlaceholders[strings.ToLower(text[loc[0]:loc[1]])] {
				continue
			}
			findings = append(findings, GuardFinding{
				Category:   "secret",
				Severity:   "critical",
				Rule:       p.rule,
				Confidence: 0.95,
				Start:      loc[0],
				End:        loc[1],
				Local:      true,
			})
		}
	}
	return findings
}
