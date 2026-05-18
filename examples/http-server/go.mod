module github.com/san-data-systems/niriksha-sdk-go/examples/http-server

go 1.22

require (
	github.com/san-data-systems/niriksha-sdk-go v0.0.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.53.0
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/metric v1.28.0
)

replace github.com/san-data-systems/niriksha-sdk-go => ../..
