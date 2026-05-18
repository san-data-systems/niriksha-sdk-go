# gin example

A product catalog REST API built with the Gin framework, showing how `otelgin.Middleware` automatically creates a root span for every incoming request and propagates the trace context through Gin's `*gin.Context`. Custom child spans add structured attributes (`product.id`, `product.name`, `products.count`) to individual handlers, and a `products.created` counter metric is incremented on each successful `POST /products`.

## Prerequisites

- Go 1.22+
- A NirikshaAI project API key (`nai_…`)

## Run

```bash
export NIRIKSHA_API_KEY=nai_your_key_here
go run .
```

## What you will see in NirikshaAI

- **Traces** — each request to `GET /products` or `POST /products` appears as a Gin-instrumented root span; child spans `catalog.listProducts` and `catalog.createProduct` carry custom span attributes for easy filtering in the NirikshaAI trace explorer.
- **Metrics** — the `products.created` counter increments each time a new product is added via the API.
- **Logs** — Gin's default logger output is captured and linked to the active trace context.
