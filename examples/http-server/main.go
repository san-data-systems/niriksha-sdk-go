package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tracer        = otel.Tracer("order-service")
	orderCounter  metric.Int64Counter
	orderDuration metric.Float64Histogram
)

func initMetrics() {
	meter := nirikshaai.Meter("order-service")
	var err error
	orderCounter, err = meter.Int64Counter("orders.created",
		metric.WithDescription("Total number of orders created"))
	if err != nil {
		log.Fatalf("counter: %v", err)
	}
	orderDuration, err = meter.Float64Histogram("orders.processing_duration_ms",
		metric.WithDescription("Order processing time in milliseconds"),
		metric.WithUnit("ms"))
	if err != nil {
		log.Fatalf("histogram: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "db.fetchOrder")
	defer span.End()

	orderID := r.PathValue("id")
	span.SetAttributes(attribute.String("order.id", orderID))

	_ = ctx // context propagated to span
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": orderID, "status": "pending"})
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx, span := tracer.Start(r.Context(), "orders.create")
	defer func() {
		span.End()
		orderDuration.Record(ctx, float64(time.Since(start).Milliseconds()))
	}()

	span.SetAttributes(attribute.String("order.source", r.Header.Get("User-Agent")))
	orderCounter.Add(ctx, 1)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{"id": "ord_123", "status": "created"})
}

func main() {
	ctx := context.Background()
	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint:      "https://app.niriksha.ai",
		OTLPEndpoint:  "ingest.niriksha.ai:4317",
		APIKey:        os.Getenv("NIRIKSHA_API_KEY"),
		ServiceName:   "order-service",
		Environment:   "production",
		EnableMetrics: true,
		EnableLogs:    true,
	})
	if err != nil {
		log.Fatalf("nirikshaai.Init: %v", err)
	}
	defer shutdown(ctx)

	initMetrics()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /orders/{id}", getOrderHandler)
	mux.HandleFunc("POST /orders", createOrderHandler)

	handler := otelhttp.NewHandler(mux, "http-server",
		otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents))

	log.Println("order-service listening on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
