module github.com/san-data-systems/niriksha-sdk-go/examples/gin

go 1.22

require (
	github.com/san-data-systems/niriksha-sdk-go v0.0.0
	github.com/gin-gonic/gin v1.10.0
	go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin v0.53.0
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/metric v1.28.0
)

replace github.com/san-data-systems/niriksha-sdk-go => ../..
