# grpc-server example

A gRPC `OrderService` server that demonstrates how `otelgrpc.NewServerHandler()` — attached as a `grpc.StatsHandler` — automatically creates traces for every incoming RPC call without any per-handler boilerplate. The `GetOrder` handler shows how to retrieve the active span from the context using `trace.SpanFromContext` and attach custom attributes, while `PlaceOrder` creates an additional child span (`inventory.reserve`) to model a downstream operation. No proto codegen is required to run the example; the service types are defined inline.

## Prerequisites

- Go 1.22+
- A NirikshaAI project API key (`nai_…`)

## Run

```bash
export NIRIKSHA_API_KEY=nai_your_key_here
go run .
```

Use [grpcurl](https://github.com/fullstorydev/grpcurl) or any gRPC client to send requests to `localhost:50051`.

## What you will see in NirikshaAI

- **Traces** — each RPC call becomes a root span carrying gRPC metadata (`rpc.method`, `rpc.service`, status code); `PlaceOrder` produces a child span `inventory.reserve` with `order.item` and `order.quantity` attributes.
- **Metrics** — gRPC request counts and latency histograms are emitted automatically by the `otelgrpc` stats handler.
- **Logs** — server startup and error logs are forwarded via the OTLP log exporter.
