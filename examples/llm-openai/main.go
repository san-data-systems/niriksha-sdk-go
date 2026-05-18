package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("llm-openai-example")

func callChatCompletion(ctx context.Context, client *openai.Client, prompt string) (string, error) {
	ctx, span := tracer.Start(ctx, "openai.chat.completion",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	span.SetAttributes(
		attribute.String("llm.provider", "openai"),
		attribute.String("llm.model", openai.GPT4oMini),
		attribute.Int("llm.prompt.length", len(prompt)),
	)

	start := time.Now()
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	latencyMs := float64(time.Since(start).Milliseconds())
	content := resp.Choices[0].Message.Content

	span.SetAttributes(
		attribute.Int("llm.usage.prompt_tokens", resp.Usage.PromptTokens),
		attribute.Int("llm.usage.completion_tokens", resp.Usage.CompletionTokens),
		attribute.Float64("llm.latency_ms", latencyMs),
	)
	span.SetStatus(codes.Ok, "")

	// Extract the trace ID and submit an eval for this LLM call.
	traceID := span.SpanContext().TraceID().String()
	evalErr := nirikshaai.SubmitEval(ctx, nirikshaai.EvalInput{
		TraceID:     traceID,
		MetricName:  "response_quality",
		Score:       1.0,
		Label:       "pass",
		Explanation: fmt.Sprintf("completion returned %d tokens in %.0fms", resp.Usage.CompletionTokens, latencyMs),
		EvalType:    "rule_based",
	})
	if evalErr != nil {
		log.Printf("warn: SubmitEval: %v", evalErr)
	}

	return content, nil
}

func main() {
	ctx := context.Background()
	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint:      "https://app.niriksha.ai",
		OTLPEndpoint:  "ingest.niriksha.ai:4317",
		APIKey:        os.Getenv("NIRIKSHA_API_KEY"),
		ServiceName:   "llm-openai-example",
		Environment:   "production",
		EnableMetrics: true,
		EnableLogs:    true,
	})
	if err != nil {
		log.Fatalf("nirikshaai.Init: %v", err)
	}
	defer shutdown(ctx)

	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	prompt := "Summarise the benefits of distributed tracing in two sentences."
	reply, err := callChatCompletion(ctx, client, prompt)
	if err != nil {
		log.Fatalf("chat completion: %v", err)
	}
	fmt.Println("Response:", reply)
}
