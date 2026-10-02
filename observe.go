package nirikshaai

// Wrap an agent so the platform sees it as a run.
//
//   - Observe wraps an agent entry point in a root span. The platform turns it
//     into an agent run: it appears on LLM → Runs as soon as the first child
//     span arrives, flips to Succeeded or Failed when this span ends, and the
//     health rules (long running, repeated tool, repeated error, no activity,
//     possible loop) evaluate it.
//   - Span is a unit of work inside the run — an LLM call, a tool call, a
//     retrieval or a nested agent. The type drives the timeline.
//   - Log is a structured line attached to the active run.
//
// Plain OpenTelemetry with the GenAI semantic conventions
// (gen_ai.operation.name, gen_ai.agent.name, gen_ai.tool.name), so the same
// spans read in any OTel backend. No LLM library is required.
//
//	shutdown, _ := nirikshaai.Init(ctx, nirikshaai.Options{Endpoint: "https://app.niriksha.ai", APIKey: "nai_…", ServiceName: "research-agent"})
//	defer shutdown(ctx)
//	err := nirikshaai.Observe(ctx, "research-agent", func(ctx context.Context) error {
//		nirikshaai.Log(ctx, "info", "agent started", map[string]any{"question": q})
//		plan, err := nirikshaai.SpanResult(ctx, "plan", nirikshaai.SpanLLM, func(ctx context.Context) (string, error) { return callModel(ctx, q) })
//		...
//	})

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
)

// SpanType is the kind of work a step is; it maps onto gen_ai.operation.name.
type SpanType string

const (
	SpanLLM       SpanType = "llm"
	SpanTool      SpanType = "tool"
	SpanAgent     SpanType = "agent"
	SpanRetrieval SpanType = "retrieval"
)

const (
	observeTracerName = "nirikshaai.observe"
	// runAttr marks the root span of a run so the platform never guesses from
	// a missing parent alone.
	runAttr = "niriksha.run"
)

func operationFor(t SpanType) string {
	switch t {
	case SpanLLM:
		return "chat"
	case SpanAgent:
		return "invoke_agent"
	case SpanRetrieval:
		return "retrieval"
	default:
		return "execute_tool"
	}
}

// SpanOption customises Span and Observe.
type SpanOption func(*spanConfig)

type spanConfig struct {
	model string
	attrs []attribute.KeyValue
}

// WithModel sets gen_ai.request.model on an LLM span.
func WithModel(model string) SpanOption { return func(c *spanConfig) { c.model = model } }

// WithAttributes adds attributes to the span; values that are not OTel
// primitives are JSON-encoded.
func WithAttributes(attrs map[string]any) SpanOption {
	return func(c *spanConfig) { c.attrs = append(c.attrs, toAttributes(attrs)...) }
}

func toAttributes(values map[string]any) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(values))
	for k, v := range values {
		switch x := v.(type) {
		case nil:
			continue
		case string:
			out = append(out, attribute.String(k, x))
		case bool:
			out = append(out, attribute.Bool(k, x))
		case int:
			out = append(out, attribute.Int(k, x))
		case int64:
			out = append(out, attribute.Int64(k, x))
		case float64:
			out = append(out, attribute.Float64(k, x))
		case []string:
			out = append(out, attribute.StringSlice(k, x))
		default:
			b, err := json.Marshal(x)
			if err != nil {
				out = append(out, attribute.String(k, fmt.Sprint(x)))
			} else {
				out = append(out, attribute.String(k, string(b)))
			}
		}
	}
	return out
}

func finishSpan(sp trace.Span, err error) {
	if err != nil {
		sp.RecordError(err)
		sp.SetStatus(codes.Error, err.Error())
	} else {
		sp.SetStatus(codes.Ok, "")
	}
	sp.End()
}

// Observe wraps an agent entry point in a root span — the platform's run.
// fn's error is recorded on the span and returned.
func Observe(ctx context.Context, name string, fn func(ctx context.Context) error, opts ...SpanOption) error {
	cfg := &spanConfig{}
	for _, o := range opts {
		o(cfg)
	}
	attrs := append([]attribute.KeyValue{
		attribute.Bool(runAttr, true),
		attribute.String("gen_ai.operation.name", operationFor(SpanAgent)),
		attribute.String("gen_ai.agent.name", name),
	}, cfg.attrs...)
	ctx, sp := otel.Tracer(observeTracerName).Start(ctx, name, trace.WithSpanKind(trace.SpanKindInternal), trace.WithAttributes(attrs...))
	err := fn(ctx)
	finishSpan(sp, err)
	return err
}

// Span runs fn inside a child span of the current run.
func Span(ctx context.Context, name string, t SpanType, fn func(ctx context.Context) error, opts ...SpanOption) error {
	_, err := SpanResult(ctx, name, t, func(ctx context.Context) (struct{}, error) { return struct{}{}, fn(ctx) }, opts...)
	return err
}

// SpanResult is Span for a function that returns a value.
func SpanResult[T any](ctx context.Context, name string, t SpanType, fn func(ctx context.Context) (T, error), opts ...SpanOption) (T, error) {
	cfg := &spanConfig{}
	for _, o := range opts {
		o(cfg)
	}
	attrs := []attribute.KeyValue{attribute.String("gen_ai.operation.name", operationFor(t))}
	switch t {
	case SpanTool:
		attrs = append(attrs, attribute.String("gen_ai.tool.name", name))
	case SpanAgent:
		attrs = append(attrs, attribute.String("gen_ai.agent.name", name))
	}
	if cfg.model != "" {
		attrs = append(attrs, attribute.String("gen_ai.request.model", cfg.model))
	}
	attrs = append(attrs, cfg.attrs...)
	ctx, sp := otel.Tracer(observeTracerName).Start(ctx, name, trace.WithSpanKind(trace.SpanKindInternal), trace.WithAttributes(attrs...))
	v, err := fn(ctx)
	finishSpan(sp, err)
	return v, err
}

// Log attaches a structured line to the active run: an event on the active
// span (so the run page shows it even with log export off) and an OTel log
// record with the trace context when a logger provider is installed.
// level is trace | debug | info | warn | error | fatal.
func Log(ctx context.Context, level, message string, attrs map[string]any) {
	kv := toAttributes(attrs)
	if sp := trace.SpanFromContext(ctx); sp.IsRecording() {
		sp.AddEvent(message, trace.WithAttributes(append([]attribute.KeyValue{attribute.String("log.level", strings.ToLower(level))}, kv...)...))
	}
	var rec otellog.Record
	rec.SetBody(otellog.StringValue(message))
	rec.SetSeverityText(strings.ToUpper(level))
	rec.SetSeverity(severityFor(level))
	for _, a := range kv {
		rec.AddAttributes(otellog.KeyValue{Key: string(a.Key), Value: otellog.StringValue(a.Value.Emit())})
	}
	global.GetLoggerProvider().Logger("nirikshaai.agent").Emit(ctx, rec)
}

func severityFor(level string) otellog.Severity {
	switch strings.ToLower(level) {
	case "trace":
		return otellog.SeverityTrace
	case "debug":
		return otellog.SeverityDebug
	case "warn", "warning":
		return otellog.SeverityWarn
	case "error":
		return otellog.SeverityError
	case "fatal":
		return otellog.SeverityFatal
	default:
		return otellog.SeverityInfo
	}
}
