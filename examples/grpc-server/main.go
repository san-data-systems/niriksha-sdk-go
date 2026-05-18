// Package main demonstrates gRPC server instrumentation with the NirikshaAI SDK.
//
// Proto definition (inline, no codegen required):
//
//	service OrderService {
//	    rpc GetOrder  (GetOrderRequest)  returns (OrderResponse);
//	    rpc PlaceOrder(PlaceOrderRequest) returns (OrderResponse);
//	}
package main

import (
	"context"
	"log"
	"net"
	"os"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	otelgrpc "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
)

// ── Minimal hand-rolled types (replace with protoc-generated code in production) ──

type GetOrderRequest struct{ ID string }
type PlaceOrderRequest struct{ Item string; Quantity int32 }
type OrderResponse struct{ ID string; Status string }

// OrderServiceServer is the interface a generated proto server would embed.
type OrderServiceServer interface {
	GetOrder(context.Context, *GetOrderRequest) (*OrderResponse, error)
	PlaceOrder(context.Context, *PlaceOrderRequest) (*OrderResponse, error)
}

// orderServer implements OrderServiceServer.
type orderServer struct{}

func (s *orderServer) GetOrder(ctx context.Context, req *GetOrderRequest) (*OrderResponse, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("order.id", req.ID))

	if req.ID == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}
	return &OrderResponse{ID: req.ID, Status: "pending"}, nil
}

func (s *orderServer) PlaceOrder(ctx context.Context, req *PlaceOrderRequest) (*OrderResponse, error) {
	_, span := otel.Tracer("order-service-grpc").Start(ctx, "inventory.reserve")
	defer span.End()
	span.SetAttributes(
		attribute.String("order.item", req.Item),
		attribute.Int("order.quantity", int(req.Quantity)),
	)
	return &OrderResponse{ID: "ord_456", Status: "placed"}, nil
}

// registerOrderService manually wires the server (stands in for proto-generated RegisterXxxServer).
func registerOrderService(s *grpc.Server, srv OrderServiceServer) {
	// In a real service this is generated: pb.RegisterOrderServiceServer(s, srv)
	// Here we log the registration to keep the example self-contained.
	log.Printf("OrderService registered: %T", srv)
	_ = srv
	_ = s
}

func main() {
	ctx := context.Background()
	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint:      "https://app.niriksha.ai",
		OTLPEndpoint:  "ingest.niriksha.ai:4317",
		APIKey:        os.Getenv("NIRIKSHA_API_KEY"),
		ServiceName:   "order-service-grpc",
		Environment:   "production",
		EnableMetrics: true,
		EnableLogs:    true,
	})
	if err != nil {
		log.Fatalf("nirikshaai.Init: %v", err)
	}
	defer shutdown(ctx)

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	)
	registerOrderService(srv, &orderServer{})

	log.Println("order-service-grpc listening on :50051")
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
