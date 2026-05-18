module github.com/san-data-systems/niriksha-sdk-go/examples/grpc-server

go 1.22

require (
	github.com/san-data-systems/niriksha-sdk-go v0.0.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.53.0
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/trace v1.28.0
	google.golang.org/grpc v1.65.0
)

replace github.com/san-data-systems/niriksha-sdk-go => ../..
