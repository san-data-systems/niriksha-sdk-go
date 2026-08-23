# NirikshaAI Go SDK

[![CI](https://github.com/san-data-systems/niriksha-sdk-go/actions/workflows/ci.yml/badge.svg)](https://github.com/san-data-systems/niriksha-sdk-go/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/san-data-systems/niriksha-sdk-go)](https://goreportcard.com/report/github.com/san-data-systems/niriksha-sdk-go)
[![Go Reference](https://pkg.go.dev/badge/github.com/san-data-systems/niriksha-sdk-go.svg)](https://pkg.go.dev/github.com/san-data-systems/niriksha-sdk-go)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go version](https://img.shields.io/github/go-mod/go-version/san-data-systems/niriksha-sdk-go.svg)](https://go.dev/)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/san-data-systems/niriksha-sdk-go.svg)](https://pkg.go.dev/github.com/san-data-systems/niriksha-sdk-go)

The official Go SDK for [NirikshaAI](https://nirikshaai.com) — AI-native observability for logs, metrics, traces, and LLM/agent telemetry.

Under the hood this is a thin wrapper around the [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/). It configures OTLP gRPC exporters, registers the global `TracerProvider`, `MeterProvider`, and `LoggerProvider`, and exposes NirikshaAI-specific helpers (evals, prompt management). You can use the standard OTEL API at any time alongside it.

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration Reference](#configuration-reference)
- [What Init() Sets Up](#what-init-sets-up)
- [Traces](#traces)
- [Metrics](#metrics)
- [Logs](#logs)
- [Trace Context Propagation](#trace-context-propagation-http)
- [Inline Guard](#inline-guard)
- [Eval Submission](#eval-submission)
- [Prompt Management](#prompt-management)
- [Graceful Shutdown](#graceful-shutdown)
- [gRPC Service Example](#grpc-service-example)
- [Using with an Existing OTEL Setup](#using-with-an-existing-otel-setup)
- [Contributing](#contributing)
- [Security](#security)
- [Changelog](#changelog)

---

## Installation

```bash
go get github.com/san-data-systems/niriksha-sdk-go
```

**Minimum Go version:** 1.22

---

## Quick Start

```go
package main

import (
	"context"
	"log"
	"time"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func main() {
	ctx := context.Background()

	// SaaS — REST API and OTLP gateway are on separate hosts
	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint:     "https://app.niriksha.ai",
		OTLPEndpoint: "grpc-ingest.niriksha.ai:443",
		APIKey:       "nai_...",
		ServiceName:  "my-go-service",
		Environment:  "production",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := shutdown(shutdownCtx); err != nil {
			log.Printf("telemetry shutdown: %v", err)
		}
	}()

	// Use standard OTEL tracer
	tracer := otel.Tracer("my-service")
	ctx, span := tracer.Start(ctx, "process-request")
	defer span.End()

	// Use standard OTEL meter
	meter := nirikshaai.Meter("my-service")
	counter, _ := meter.Int64Counter(
		"requests.total",
		metric.WithDescription("Total HTTP requests"),
	)
	counter.Add(ctx, 1, metric.WithAttributes(attribute.String("status", "ok")))
}
```

Your `APIKey` is a project-scoped key (prefixed `nai_`). It encodes which org and project your telemetry belongs to — you do not need to pass org or project IDs separately.

---

## Configuration Reference

| Field | Type | Default | Description |
|---|---|---|---|
| `Endpoint` | `string` | *(required)* | NirikshaAI REST/control-plane base URL. SaaS: `https://app.niriksha.ai`. Private Cloud: `https://niriksha.internal` |
| `OTLPEndpoint` | `string` | `""` | Override the gRPC OTLP address (`host:port`, no scheme). SaaS: `grpc-ingest.niriksha.ai:443`. Derived from `Endpoint` if empty. |
| `APIKey` | `string` | *(required)* | Project-scoped API key with `nai_` prefix |
| `ServiceName` | `string` | `"go-service"` | Value of the `service.name` OTEL resource attribute |
| `Environment` | `string` | `"production"` | Value of the `deployment.environment` OTEL resource attribute |
| `EnableMetrics` | `bool` | `true` | Export OTLP metrics on a 60-second periodic interval |
| `EnableLogs` | `bool` | `true` | Export OTLP logs |
| `OTLPPort` | `int` | `4317` | OTLP gRPC port. Ignored when `OTLPEndpoint` is set explicitly. |
| `Insecure` | `bool` | `false` | Send gRPC without TLS. Use when TLS is terminated at an ingress in front of the gateway. |
| `TLSSkipVerify` | `bool` | `false` | Use TLS but skip server certificate validation. Dev/staging only. |
| `CACertFile` | `string` | `""` | Path to a PEM CA certificate for verifying the gateway's TLS cert. Use for private CAs. |
| `GuardEndpoint` | `string` | *derived* | Base URL of the guard endpoint — the gateway's HTTP listener. Derived from `OTLPEndpoint`, or `Endpoint` when that is empty. See [Inline Guard](#inline-guard) |
| `GuardFailMode` | `GuardFailMode` | `GuardFailOpen` | Behaviour when the guard is unreachable: `GuardFailOpen`, `GuardFailClosed`, or `GuardFailSecretsClosed` |
| `GuardMode` | `string` | `""` | Default mode for every guard call: `"monitor"` or `"block"` |

### Private Cloud examples

```go
// TLS with trusted or custom CA
nirikshaai.Init(ctx, nirikshaai.Options{
    Endpoint:     "https://niriksha.internal",
    OTLPEndpoint: "niriksha.internal:4317",
    APIKey:       "nai_...",
    ServiceName:  "my-service",
    CACertFile:   "/etc/ssl/niriksha-ca.crt", // omit for system roots
})

// Skip TLS verification (dev/staging only)
nirikshaai.Init(ctx, nirikshaai.Options{
    Endpoint:      "https://niriksha.internal",
    OTLPEndpoint:  "niriksha.internal:4317",
    APIKey:        "nai_...",
    TLSSkipVerify: true,
})

// Plaintext gRPC (TLS terminated at ingress)
nirikshaai.Init(ctx, nirikshaai.Options{
    Endpoint:     "https://niriksha.internal",
    OTLPEndpoint: "niriksha.internal:4317",
    APIKey:       "nai_...",
    Insecure:     true,
})
```

---

## What Init() Sets Up

After a successful call to `Init()`, the following global providers are registered:

| Provider | Accessor | Export mechanism |
|---|---|---|
| `TracerProvider` | `otel.GetTracerProvider()` | OTLP gRPC, batched |
| `MeterProvider` | `otel.GetMeterProvider()` | OTLP gRPC, 60s periodic |
| `LoggerProvider` | `global.GetLoggerProvider()` | OTLP gRPC, batched |

All three share the same OTLP endpoint and carry the `X-API-Key` header on every export request.

---

## Traces

Use `otel.Tracer()` from `go.opentelemetry.io/otel` to get a tracer. Spans are automatically batched and exported over OTLP gRPC.

```go
package orders

import (
	"context"
	"errors"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("order-service")

func ProcessOrder(ctx context.Context, orderID string) error {
	ctx, span := tracer.Start(ctx, "process-order",
		trace.WithAttributes(
			attribute.String("order.id", orderID),
		),
	)
	defer span.End()

	if err := validateOrder(ctx, orderID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	total, err := calculateTotal(ctx, orderID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetAttributes(
		attribute.String("order.status", "validated"),
		attribute.Float64("order.total", total),
	)
	return nil
}

func validateOrder(ctx context.Context, orderID string) error {
	// Child span — automatically linked to the parent via context
	_, span := tracer.Start(ctx, "validate-order")
	defer span.End()

	if orderID == "" {
		return errors.New("orderID must not be empty")
	}
	return nil
}
```

---

## Metrics

Use `nirikshaai.Meter()` (a convenience wrapper for `otel.GetMeterProvider().Meter()`) or call `otel.GetMeterProvider().Meter()` directly.

```go
package orders

import (
	"context"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter = nirikshaai.Meter("order-service")

	// Counter — monotonically increasing
	ordersTotal, _ = meter.Int64Counter(
		"orders.total",
		metric.WithDescription("Total orders processed"),
		metric.WithUnit("{order}"),
	)

	// Histogram — for latency, sizes, and distributions
	processingLatency, _ = meter.Float64Histogram(
		"order.processing.duration",
		metric.WithDescription("Time to process an order end-to-end"),
		metric.WithUnit("ms"),
	)

	// UpDownCounter — for values that go up and down
	activeOrders, _ = meter.Int64UpDownCounter(
		"orders.active",
		metric.WithDescription("Orders currently being processed"),
	)
)

func ProcessOrder(ctx context.Context, orderID string) error {
	activeOrders.Add(ctx, 1)
	defer activeOrders.Add(ctx, -1)

	start := time.Now()
	defer func() {
		processingLatency.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	// ... processing logic ...

	ordersTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("region", "us-east"),
		attribute.String("status", "success"),
	))
	return nil
}
```

### Observable instruments (polled on export)

```go
// Observable gauge — value is read on each collection cycle
activeConnGauge, _ := meter.Int64ObservableGauge(
	"db.connections.active",
	metric.WithDescription("Active database connections in the pool"),
)

meter.RegisterCallback(
	func(ctx context.Context, o metric.Observer) error {
		o.ObserveInt64(activeConnGauge, int64(pool.ActiveConnections()),
			metric.WithAttributes(attribute.String("db", "postgres")),
		)
		return nil
	},
	activeConnGauge,
)
```

---

## Logs

### Using the OTEL logger (low-level)

```go
import (
	"go.opentelemetry.io/otel/log/global"
	otellog "go.opentelemetry.io/otel/log"
)

logger := global.Logger("order-service")

var r otellog.Record
r.SetBody(otellog.StringValue("order processed"))
r.AddAttributes(
	otellog.String("order.id", orderID),
	otellog.Float64("order.total", total),
)
logger.Emit(ctx, r)
```

### Using the slog bridge (recommended)

The `otelslog` bridge lets you use Go's idiomatic structured logging API while exporting records over OTEL:

```go
import (
	"context"
	"log/slog"

	"go.opentelemetry.io/contrib/bridges/otelslog"
)

// Set up slog to export via OTEL — do this once at startup, after Init()
handler := otelslog.NewHandler("order-service")
slog.SetDefault(slog.New(handler))

// All slog calls are exported to NirikshaAI with trace/span ID correlation
slog.Info("order processed",
	"order_id", orderID,
	"amount", total,
	"region", "us-east",
)

slog.Warn("inventory low",
	"sku", "WIDGET-001",
	"remaining", 3,
)

slog.Error("payment failed",
	"error", err,
	"order_id", orderID,
	"error_code", "CARD_DECLINED",
)

// With context — attaches the active span's trace ID to the log record
slog.InfoContext(ctx, "processing started", "order_id", orderID)
```

Log records are batched and exported over OTLP gRPC to the same endpoint as traces and metrics.

---

## Trace Context Propagation (HTTP)

When calling downstream services, inject the active trace context into outbound request headers so distributed traces are stitched together end-to-end.

### Outbound HTTP request (client side)

```go
import (
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

func callDownstreamService(ctx context.Context, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	// Inject W3C TraceContext and Baggage headers
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	return http.DefaultClient.Do(req)
}
```

### Inbound HTTP request (server side)

```go
func orderHandler(w http.ResponseWriter, r *http.Request) {
	// Extract trace context from incoming headers
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	// Start a child span — it will be linked to the upstream trace
	ctx, span := tracer.Start(ctx, "handle-order-request")
	defer span.End()

	// ... handler logic ...
}
```

### Register the global propagator once at startup

`Init()` registers `TraceContext` and `Baggage` propagators automatically. If you need to add custom propagators:

```go
import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Call after nirikshaai.Init()
otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
	// Add any additional propagators here
))
```

---

## Inline Guard

Everything else in this SDK records what happened. The guard is enforcement: it
checks text **before** it reaches the model, so a prompt injection can be refused
and a leaked credential stripped rather than merely reported afterwards.

```go
verdict, err := nirikshaai.GuardCheck(ctx, userPrompt, "input")
if errors.Is(err, nirikshaai.ErrGuardBlocked) {
    return fmt.Errorf("request refused: %w", err)
}

// On a redact verdict this returns the rewritten text; otherwise the original.
resp, err := llm.Complete(ctx, verdict.SafeText(userPrompt))
```

### Verdicts

| Action | What it means | What you should do |
|---|---|---|
| `GuardAllow` | Nothing found | Proceed |
| `GuardTag` | Something found, not reliable enough to act on | **Proceed.** Record it |
| `GuardRedact` | Sensitive content found and removed | Proceed **with `verdict.SafeText(text)`** |
| `GuardBlock` | High-confidence attack | Do not send |

**Only a block returns an error.** A redact verdict returns `nil`, because a
customer who asked for PII stripping wants their data protected, not their
application broken. The error wraps `ErrGuardBlocked` and names the rules that
fired, and the verdict is returned alongside it — so a caller gets the reason
without a second guard call:

```go
verdict, err := nirikshaai.GuardCheck(ctx, prompt, "input")
if errors.Is(err, nirikshaai.ErrGuardBlocked) {
    log.Warn("blocked", "risk", verdict.RiskScore, "findings", verdict.Findings)
}
```

The verdict also carries `RiskScore`, `RiskSeverity`, `Findings`, `Reasons`,
`PolicySource` and `PolicyEnforced` — the last two tell you whether your org's
AIDR policy or the product default produced the verdict. Only the former is
binding, and `GuardMode: "monitor"` cannot lift a block your org's policy
mandates.

### Tool calls

The check that can actually prevent an action, rather than describe it after the
fact:

```go
if _, err := nirikshaai.GuardCheckTool(ctx, "bash", map[string]string{"cmd": cmd}); err != nil {
    if errors.Is(err, nirikshaai.ErrGuardBlocked) {
        return errors.New("tool call refused")
    }
}
run(cmd)
```

Covers file destruction, shell execution, destructive SQL, credential access,
network egress, and **a credential appearing in a tool argument** — the concrete
exfiltration path when an agent is persuaded to pass a key to an outbound tool.

`args` may be any JSON-marshalable value or a pre-serialised string. If it cannot
be marshalled the tool **name** is still checked, and a warning says why the
arguments were not — an unmarshalable argument must not silently become an
unchecked tool call.

### Whole conversations

```go
items := make([]nirikshaai.GuardBatchItem, 0, len(messages))
for _, m := range messages {
    items = append(items, nirikshaai.GuardBatchItem{Text: m.Content, Direction: "input"})
}
action, verdicts, err := nirikshaai.GuardCheckBatch(ctx, items)
```

A per-string API is an N+1 for a multi-turn message array, which is every real
chat application. Up to 32 items; the returned action is the most severe of the
set, because one blocked message means the conversation must not be sent.

### When the guard is unreachable

| `GuardFailMode` | Behaviour |
|---|---|
| `GuardFailOpen` *(default)* | Allow the text through |
| `GuardFailClosed` | Block everything |
| `GuardFailSecretsClosed` | Allow everything **except** locally-detectable credentials |

Fail-open is the default because a guard outage must not take down your
application — but it is **never silent**. Every fall-back logs a warning, sets
`verdict.FailedOpen`, and increments a `guard.fail_open` counter. A silent
fail-open is a security hole wearing a reliability costume: the control appears to
work right up until the moment it is needed.

`GuardFailSecretsClosed` is the mode worth using in production. Ten
prefix-anchored secret formats are embedded in the SDK — AWS, GitHub, Slack,
Stripe, Google, OpenAI, Anthropic, PEM private keys, NirikshaAI's own — so a
server outage stops credential exfiltration locally while everything else still
flows. `GuardFailClosed` is correct only for a hard compliance boundary; for
everyone else it converts a guard outage into an application outage.

`LocalSecretFindings` is exported, so you can run the same check yourself — on
data you are about to log, say — without depending on the guard being reachable.

An unknown `GuardFailMode` makes `Init` return an error rather than silently
defaulting, so a deployment cannot believe it is fail-closed when it is not.

### Where the guard lives

The guard endpoint is served by the **OTLP gateway**, not the REST API. In SaaS
those are different hosts, so the URL is derived from `OTLPEndpoint` when you set
it, and from `Endpoint` when you do not:

| `Endpoint` | `OTLPEndpoint` | Derived guard URL |
|---|---|---|
| `https://niriksha.internal` | *(empty)* | `https://niriksha.internal` |
| `https://app.niriksha.ai` | `grpc-ingest.niriksha.ai:443` | `https://grpc-ingest.niriksha.ai:443` |
| *(any)* | `niriksha.internal:4317` | `http://niriksha.internal:4318` |

The last row translates the gateway's default gRPC port to its default HTTP port.
A non-default port is used as configured, since guessing would be worse than
reusing what you already set. Set `GuardEndpoint` explicitly for anything this
does not cover — a wrong value shows up as "guard unreachable" on every call.

Requests time out after 3 seconds with **no retry**: this is on the critical path
in front of your model call, and retrying would turn a 3-second timeout into a
9-second one. The fail mode is a better answer than a slower one.

---

## Eval Submission

Submit evaluation results for LLM responses. Evals are linked to a specific trace so results appear in the NirikshaAI LLM Traces view alongside the originating span.

### Single eval

```go
err := nirikshaai.SubmitEval(ctx, nirikshaai.EvalInput{
	TraceID:     "your-trace-id",   // 32-character hex trace ID
	MetricName:  "faithfulness",
	Score:       0.92,               // float64 in [0, 1]
	Label:       "pass",             // "pass" | "fail" | any custom label
	Explanation: "Response accurately reflects the source documents",
	EvalType:    "llm_judge",        // "llm_judge" | "rule_based" | "human"
})
if err != nil {
	log.Printf("eval submission failed: %v", err)
}
```

### Batch eval

```go
err = nirikshaai.SubmitEvalsBatch(ctx, []nirikshaai.EvalInput{
	{
		TraceID:    "abc123...",
		MetricName: "toxicity",
		Score:      0.01,
		Label:      "pass",
		EvalType:   "rule_based",
	},
	{
		TraceID:     "abc123...",
		MetricName:  "relevance",
		Score:       0.88,
		Label:       "pass",
		EvalType:    "llm_judge",
		Explanation: "Response addresses the user's question directly",
	},
})
```

### Obtaining the trace ID from an active span

```go
func handleLLMRequest(ctx context.Context, question string) (string, error) {
	ctx, span := tracer.Start(ctx, "llm-call")
	defer span.End()

	answer, err := callLLM(ctx, question)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	// Get trace ID for eval submission
	traceID := span.SpanContext().TraceID().String() // 32-character hex string

	// Submit eval asynchronously
	go func() {
		score := runFaithfulnessJudge(question, answer)
		label := "pass"
		if score < 0.8 {
			label = "fail"
		}
		_ = nirikshaai.SubmitEval(context.Background(), nirikshaai.EvalInput{
			TraceID:    traceID,
			MetricName: "faithfulness",
			Score:      score,
			Label:      label,
			EvalType:   "llm_judge",
		})
	}()

	return answer, nil
}
```

---

## Prompt Management

Fetch versioned prompt templates from the NirikshaAI prompt vault. Variable substitution is performed server-side before the rendered text is returned.

### Fetch the latest deployed version

```go
prompt, err := nirikshaai.GetPrompt(ctx, "customer-support-system", nil)
if err != nil {
	return err
}
fmt.Printf("name: %s\n", prompt.Name)
fmt.Printf("version: %d\n", prompt.Version)
fmt.Printf("text:\n%s\n", prompt.Text)
```

### Fetch a specific version with variable substitution

```go
v := 3
prompt, err := nirikshaai.GetPrompt(ctx, "product-description", &nirikshaai.GetPromptOptions{
	Version: &v,
	Variables: map[string]string{
		"product_name": "Widget Pro",
		"category":     "Electronics",
		"price":        "$49.99",
	},
})
if err != nil {
	return err
}
// All {{variables}} in the template have been replaced server-side
fmt.Println(prompt.Text)
```

### List all available prompts

```go
prompts, err := nirikshaai.ListPrompts(ctx)
if err != nil {
	return err
}
for _, p := range prompts {
	fmt.Printf("%s (v%d) — %s\n", p.Name, p.Version, p.Description)
}
```

### Use a fetched prompt in an LLM call

```go
func generateProductDescription(ctx context.Context, productName, category string) (string, error) {
	ctx, span := tracer.Start(ctx, "generate-description")
	defer span.End()

	prompt, err := nirikshaai.GetPrompt(ctx, "product-description", &nirikshaai.GetPromptOptions{
		Variables: map[string]string{
			"product_name": productName,
			"category":     category,
		},
	})
	if err != nil {
		return "", fmt.Errorf("fetch prompt: %w", err)
	}

	span.SetAttribute("prompt.version", prompt.Version)
	span.SetAttribute("prompt.name", prompt.Name)

	// Use the rendered prompt text in your LLM call
	return callLLM(ctx, prompt.Text)
}
```

---

## Graceful Shutdown

`Init()` returns a `ShutdownFunc` that flushes all pending spans, metrics, and log records before the process exits. Always call it, either via `defer` or in a signal handler.

```go
shutdown, err := nirikshaai.Init(ctx, opts)
if err != nil {
	log.Fatal(err)
}

// Recommended: wrap in a timeout to avoid blocking indefinitely
defer func() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := shutdown(shutdownCtx); err != nil {
		log.Printf("telemetry shutdown error: %v", err)
	}
}()
```

### With OS signal handling

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
)

func main() {
	ctx := context.Background()

	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint:    "https://app.niriksha.ai",
		APIKey:      "nai_...",
		ServiceName: "my-go-service",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Start your application...
	go runServer()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := shutdown(shutdownCtx); err != nil {
		log.Printf("telemetry flush error: %v", err)
	}
	log.Println("done")
}
```

---

## gRPC Service Example

Use `otelgrpc.NewServerHandler()` to instrument a gRPC server. All unary and streaming RPCs will be traced automatically.

```go
package main

import (
	"context"
	"log"
	"net"
	"time"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/myorg/myservice/gen/proto"
)

type orderServer struct {
	pb.UnimplementedOrderServiceServer
}

func (s *orderServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.Order, error) {
	// ctx already carries the active span injected by otelgrpc
	return fetchOrder(ctx, req.OrderId)
}

func main() {
	ctx := context.Background()

	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint:    "http://localhost:4317",
		APIKey:      "nai_...",
		ServiceName: "order-grpc-server",
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdown(shutdownCtx)
	}()

	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	pb.RegisterOrderServiceServer(srv, &orderServer{})
	reflection.Register(srv)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("gRPC server listening on :50051")
	log.Fatal(srv.Serve(lis))
}
```

### gRPC client with trace propagation

```go
conn, err := grpc.NewClient(
	"downstream-service:50051",
	grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	grpc.WithTransportCredentials(insecure.NewCredentials()),
)
if err != nil {
	log.Fatal(err)
}
defer conn.Close()

client := pb.NewOrderServiceClient(conn)

// Trace context is propagated to the server automatically via otelgrpc
resp, err := client.GetOrder(ctx, &pb.GetOrderRequest{OrderId: orderID})
```

---

## Using with an Existing OTEL Setup

If you already have an OpenTelemetry setup in your application, you can skip calling `Init()` and use the eval and prompt helpers by configuring the SDK client directly:

```go
// Configure only the HTTP client used for eval/prompt API calls
// (no OTEL provider setup is performed)
nirikshaai.Configure(nirikshaai.ClientOptions{
	Endpoint: "https://app.niriksha.ai",
	APIKey:   "nai_...",
})

// All helpers are now usable with your existing OTEL providers
err := nirikshaai.SubmitEval(ctx, nirikshaai.EvalInput{...})
prompt, err := nirikshaai.GetPrompt(ctx, "my-prompt", nil)
prompts, err := nirikshaai.ListPrompts(ctx)
```

---

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- Development setup (Go 1.22+, golangci-lint, govulncheck)
- Branching model (feature branches from `develop`)
- Commit message format (conventional commits for auto-versioning)
- Pull request process and testing requirements

---

## Security

For reporting security vulnerabilities, see [SECURITY.md](SECURITY.md). Please do not open public issues for security problems.

---

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a complete release history.

---

## License

Apache 2.0 — see [LICENSE](LICENSE).
