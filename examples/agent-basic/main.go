// A small agent wrapped with Observe / Span / Log.
//
// Run it against any NirikshaAI install and it shows up on LLM → Runs within a
// few seconds, with a timeline, a repeated-tool finding (we call `search` four
// times in a row on purpose) and one error.
//
//	NIRIKSHA_ENDPOINT=https://app.niriksha.ai NIRIKSHA_API_KEY=nai_... go run .
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/san-data-systems/niriksha-sdk-go"
)

func callModel(_ context.Context, prompt string) (string, error) {
	time.Sleep(200 * time.Millisecond)
	if len(prompt) > 40 {
		prompt = prompt[:40]
	}
	return "answer to: " + prompt, nil
}

func search(_ context.Context, query string) ([]string, error) {
	time.Sleep(100 * time.Millisecond)
	if rand.Float64() < 0.2 { //nolint:gosec // example
		return nil, errors.New("search backend timed out")
	}
	return []string{"doc about " + query}, nil
}

func main() {
	ctx := context.Background()
	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint:     os.Getenv("NIRIKSHA_ENDPOINT"),
		APIKey:       os.Getenv("NIRIKSHA_API_KEY"),
		ServiceName:  "example-research-agent",
		OTLPEndpoint: os.Getenv("NIRIKSHA_OTLP_ENDPOINT"),
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = shutdown(ctx) }()

	question := "Why is the checkout service slow?"
	err = nirikshaai.Observe(ctx, "research-agent", func(ctx context.Context) error {
		nirikshaai.Log(ctx, "info", "agent started", map[string]any{"question": question})
		plan, err := nirikshaai.SpanResult(ctx, "plan", nirikshaai.SpanLLM, func(ctx context.Context) (string, error) {
			return callModel(ctx, "plan for "+question)
		}, nirikshaai.WithModel("example-model"))
		if err != nil {
			return err
		}
		var docs []string
		for i := 0; i < 4; i++ { // four consecutive calls → a possible_loop finding at the default threshold
			found, err := nirikshaai.SpanResult(ctx, "search", nirikshaai.SpanTool, func(ctx context.Context) ([]string, error) {
				return search(ctx, fmt.Sprintf("%s #%d", plan, i))
			})
			if err != nil {
				nirikshaai.Log(ctx, "warn", "search failed, continuing", map[string]any{"error": err.Error()})
				continue
			}
			docs = append(docs, found...)
		}
		answer, err := nirikshaai.SpanResult(ctx, "answer", nirikshaai.SpanLLM, func(ctx context.Context) (string, error) {
			return callModel(ctx, strings.Join(docs, " "))
		}, nirikshaai.WithModel("example-model"))
		if err != nil {
			return err
		}
		fmt.Println(answer)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
