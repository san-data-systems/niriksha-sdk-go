# llm-openai example

A standalone Go script that wraps an OpenAI `gpt-4o-mini` chat completion in an OpenTelemetry span and then calls `nirikshaai.SubmitEval` to attach a quality score to that exact trace. The span captures `llm.provider`, `llm.model`, prompt/completion token counts, and end-to-end latency as attributes; the eval records the trace ID, a `response_quality` metric score, and a human-readable explanation, making the LLM call fully visible in both the NirikshaAI trace explorer and the Evals dashboard.

## Prerequisites

- Go 1.22+
- A NirikshaAI project API key (`nai_…`)
- An OpenAI API key

## Run

```bash
export NIRIKSHA_API_KEY=nai_your_key_here
export OPENAI_API_KEY=sk-your_openai_key_here
go run .
```

## What you will see in NirikshaAI

- **Traces** — a single span `openai.chat.completion` with attributes for model name, token usage (`llm.usage.prompt_tokens`, `llm.usage.completion_tokens`), and `llm.latency_ms`.
- **Evals** — a `response_quality` eval result linked to the trace, visible in the NirikshaAI Evals tab with score, label, and explanation.
- **Metrics** — the SDK's periodic metric exporter emits SDK-level counters alongside your custom spans.
