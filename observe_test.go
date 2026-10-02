package nirikshaai

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func attrsOf(s sdktrace.ReadOnlySpan) map[string]attribute.Value {
	out := map[string]attribute.Value{}
	for _, kv := range s.Attributes() {
		out[string(kv.Key)] = kv.Value
	}
	return out
}

func withExporter(t *testing.T) *tracetest.InMemoryExporter {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(prev); _ = tp.Shutdown(context.Background()) })
	return exp
}

// Observe makes a run; Span makes typed children; Log lands as an event.
func TestObserveSpanLog(t *testing.T) {
	exp := withExporter(t)
	err := Observe(context.Background(), "research-agent", func(ctx context.Context) error {
		Log(ctx, "info", "started", map[string]any{"q": "x"})
		plan, err := SpanResult(ctx, "plan", SpanLLM, func(ctx context.Context) (string, error) { return "p", nil }, WithModel("gpt-4o"))
		if err != nil || plan != "p" {
			return errors.New("plan failed")
		}
		return Span(ctx, "search", SpanTool, func(ctx context.Context) error { return nil })
	})
	if err != nil {
		t.Fatal(err)
	}
	spans := map[string]sdktrace.ReadOnlySpan{}
	for _, s := range exp.GetSpans().Snapshots() {
		spans[s.Name()] = s
	}
	root := spans["research-agent"]
	if root == nil || root.Parent().IsValid() {
		t.Fatalf("root span missing or has a parent: %+v", root)
	}
	ra := attrsOf(root)
	if !ra["niriksha.run"].AsBool() || ra["gen_ai.operation.name"].AsString() != "invoke_agent" || ra["gen_ai.agent.name"].AsString() != "research-agent" {
		t.Fatalf("root attrs: %v", ra)
	}
	if root.Status().Code != codes.Ok {
		t.Fatalf("root status %v", root.Status())
	}
	if len(root.Events()) != 1 || root.Events()[0].Name != "started" {
		t.Fatalf("log event: %+v", root.Events())
	}
	pa := attrsOf(spans["plan"])
	if pa["gen_ai.operation.name"].AsString() != "chat" || pa["gen_ai.request.model"].AsString() != "gpt-4o" {
		t.Fatalf("plan attrs: %v", pa)
	}
	if spans["plan"].Parent().SpanID() != root.SpanContext().SpanID() {
		t.Fatal("plan must be a child of the run")
	}
	sa := attrsOf(spans["search"])
	if sa["gen_ai.operation.name"].AsString() != "execute_tool" || sa["gen_ai.tool.name"].AsString() != "search" {
		t.Fatalf("search attrs: %v", sa)
	}
}

func TestObserveRecordsErrors(t *testing.T) {
	exp := withExporter(t)
	boom := errors.New("boom")
	if err := Observe(context.Background(), "failing", func(context.Context) error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("error must be returned, got %v", err)
	}
	s := exp.GetSpans().Snapshots()[0]
	if s.Status().Code != codes.Error {
		t.Fatalf("status %v", s.Status())
	}
	found := false
	for _, e := range s.Events() {
		if e.Name == "exception" {
			found = true
		}
	}
	if !found {
		t.Fatal("exception event missing")
	}
}

func TestToAttributes(t *testing.T) {
	got := attribute.NewSet(toAttributes(map[string]any{"cfg": map[string]int{"k": 1}, "tags": []string{"a"}, "n": nil, "ok": true, "f": 1.5})...)
	if v, _ := got.Value("cfg"); v.AsString() != `{"k":1}` {
		t.Fatalf("cfg %v", v)
	}
	if _, ok := got.Value("n"); ok {
		t.Fatal("nil must be dropped")
	}
	if v, _ := got.Value("ok"); !v.AsBool() {
		t.Fatal("bool kept")
	}
}
