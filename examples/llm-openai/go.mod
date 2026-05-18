module github.com/san-data-systems/niriksha-sdk-go/examples/llm-openai

go 1.22

require (
	github.com/san-data-systems/niriksha-sdk-go v0.0.0
	github.com/sashabaranov/go-openai v1.28.0
	go.opentelemetry.io/otel v1.28.0
	go.opentelemetry.io/otel/trace v1.28.0
)

replace github.com/san-data-systems/niriksha-sdk-go => ../..
