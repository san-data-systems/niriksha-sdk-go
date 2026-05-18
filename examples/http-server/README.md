# http-server example

A minimal order service built with the Go standard library `net/http`, demonstrating how the NirikshaAI SDK instruments an HTTP server with zero-touch automatic tracing via `otelhttp.NewHandler`, a custom child span inside the order-fetch handler, an `orders.created` counter metric, and an `orders.processing_duration_ms` histogram that records end-to-end order creation time.

## Prerequisites

- Go 1.22+
- A NirikshaAI project API key (`nai_…`)

## Run

```bash
export NIRIKSHA_API_KEY=nai_your_key_here
go run .
```

## What you will see in NirikshaAI

- **Traces** — every HTTP request appears as a root span named `http-server`; `GET /orders/{id}` and `POST /orders` each produce a child span (`db.fetchOrder` / `orders.create`) with custom attributes such as `order.id` and `order.source`.
- **Metrics** — the `orders.created` counter increments on each `POST /orders`; the `orders.processing_duration_ms` histogram shows latency distribution for order creation.
- **Logs** — standard `log` output is captured and correlated to the active trace via the OTLP log exporter.
