// Package nirikshaai is the NirikshaAI Go SDK — full-stack observability
// for any Go service via OpenTelemetry (traces, metrics, and logs).
//
// # SaaS quick start
//
//	import "github.com/san-data-systems/niriksha-sdk-go"
//
//	func main() {
//	    shutdown, err := nirikshaai.Init(context.Background(), nirikshaai.Options{
//	        Endpoint:    "https://app.niriksha.ai",
//	        APIKey:      "nai_...",
//	        ServiceName: "my-go-service",
//	    })
//	    if err != nil { log.Fatal(err) }
//	    defer shutdown(context.Background())
//	}
//
// # Private Cloud — TLS with trusted certificate
//
//	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
//	    Endpoint:    "https://niriksha.internal",
//	    OTLPEndpoint: "niriksha.internal:4317", // explicit gRPC address
//	    APIKey:      "nai_...",
//	    ServiceName: "my-service",
//	})
//
// # Private Cloud — self-signed CA
//
//	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
//	    Endpoint:    "https://niriksha.internal",
//	    OTLPEndpoint: "niriksha.internal:4317",
//	    APIKey:      "nai_...",
//	    ServiceName: "my-service",
//	    CACertFile:  "/etc/ssl/niriksha-ca.crt",
//	})
//
// # Private Cloud — skip TLS verification (dev/staging only)
//
//	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
//	    Endpoint:       "https://niriksha.internal",
//	    OTLPEndpoint:   "niriksha.internal:4317",
//	    APIKey:         "nai_...",
//	    ServiceName:    "my-service",
//	    TLSSkipVerify:  true,
//	})
//
// # Private Cloud — plaintext gRPC (gateway TLS terminated at ingress)
//
//	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
//	    Endpoint:     "https://niriksha.internal",
//	    OTLPEndpoint: "niriksha.internal:4317",
//	    APIKey:       "nai_...",
//	    ServiceName:  "my-service",
//	    Insecure:     true,
//	})
package nirikshaai

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/log/global"
	otelmetric "go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	googlegrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Version is the current SDK release.
const Version = "0.1.0"

// Options controls NirikshaAI SDK initialisation.
type Options struct {
	// Endpoint is the NirikshaAI REST/control-plane base URL.
	//   SaaS:          "https://app.niriksha.ai"
	//   Private Cloud: "https://niriksha.internal"
	// The SDK derives the gRPC OTLP address from this URL unless OTLPEndpoint
	// is set explicitly.
	Endpoint string

	// OTLPEndpoint overrides the gRPC OTLP address (host:port, no scheme).
	// Use this when your REST API and OTLP gateway are on different hosts —
	// for example on NirikshaAI SaaS:
	//   Endpoint:     "https://app.niriksha.ai"
	//   OTLPEndpoint: "ingest.niriksha.ai:4317"
	// If empty, the SDK builds the address from Endpoint's hostname + OTLPPort.
	OTLPEndpoint string

	// APIKey is the project-scoped API key (prefix nai_).
	APIKey string

	// ServiceName sets the service.name resource attribute (default: "my-service").
	ServiceName string

	// Environment sets the deployment.environment attribute (default: "production").
	Environment string

	// EnableMetrics exports OTLP metrics (default: true).
	EnableMetrics bool

	// EnableLogs exports OTLP logs (default: true).
	EnableLogs bool

	// OTLPPort is the OTLP gRPC port used when deriving the gRPC address from
	// Endpoint (ignored when OTLPEndpoint is set explicitly). Default: 4317.
	OTLPPort int

	// Insecure sends gRPC traffic without TLS. Use when TLS is terminated at
	// an ingress layer in front of the NirikshaAI gateway (common in private
	// cloud deployments where the pod-to-gateway path is already encrypted by
	// the cluster's service mesh).
	Insecure bool

	// TLSSkipVerify uses TLS but skips server certificate validation.
	// Useful for self-signed certs in dev/staging environments.
	// Do NOT use in production.
	TLSSkipVerify bool

	// CACertFile is the path to a PEM-encoded CA certificate used to verify
	// the NirikshaAI gateway's TLS certificate. Use for private CAs.
	// Mutually exclusive with TLSSkipVerify and Insecure.
	CACertFile string
}

// global state shared with eval and prompt helpers
var _state struct {
	baseURL    string
	apiKey     string
	extraAttrs []attribute.KeyValue
}

// _initialized is set to true after a successful Init call.
var _initialized bool

// ShutdownFunc flushes and stops all providers. Call it with defer in main.
type ShutdownFunc func(ctx context.Context) error

// SetGlobalAttributes sets extra resource attributes that will be included in
// the OpenTelemetry resource created by Init. Must be called before Init.
func SetGlobalAttributes(attrs ...attribute.KeyValue) {
	_state.extraAttrs = attrs
}

// IsInitialized reports whether Init has been called successfully.
func IsInitialized() bool { return _initialized }

// buildDialOpts returns the gRPC dial options based on the TLS configuration
// in opts.
func buildDialOpts(useTLS bool, opts Options) ([]googlegrpc.DialOption, error) {
	if !useTLS || opts.Insecure {
		return []googlegrpc.DialOption{
			googlegrpc.WithTransportCredentials(insecure.NewCredentials()),
		}, nil
	}
	if opts.TLSSkipVerify {
		creds := credentials.NewTLS(&tls.Config{InsecureSkipVerify: true}) //nolint:gosec
		return []googlegrpc.DialOption{googlegrpc.WithTransportCredentials(creds)}, nil
	}
	if opts.CACertFile != "" {
		creds, err := credentials.NewClientTLSFromFile(opts.CACertFile, "")
		if err != nil {
			return nil, fmt.Errorf("nirikshaai: load CA cert %q: %w", opts.CACertFile, err)
		}
		return []googlegrpc.DialOption{googlegrpc.WithTransportCredentials(creds)}, nil
	}
	// Default: system TLS roots.
	return nil, nil
}

// Init configures global OpenTelemetry providers (traces, metrics, logs) to
// export to NirikshaAI. Returns a ShutdownFunc that should be deferred.
func Init(ctx context.Context, opts Options) (ShutdownFunc, error) {
	if opts.ServiceName == "" {
		opts.ServiceName = "my-service"
	}
	if opts.Environment == "" {
		opts.Environment = "production"
	}
	if opts.OTLPPort == 0 {
		opts.OTLPPort = 4317
	}
	if !opts.EnableMetrics && !opts.EnableLogs {
		opts.EnableMetrics = true
		opts.EnableLogs = true
	}

	u, err := url.Parse(opts.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("nirikshaai: invalid endpoint %q: %w", opts.Endpoint, err)
	}
	useTLS := strings.HasPrefix(opts.Endpoint, "https")

	grpcAddr := opts.OTLPEndpoint
	if grpcAddr == "" {
		grpcAddr = fmt.Sprintf("%s:%d", u.Hostname(), opts.OTLPPort)
	}

	dialOpts, err := buildDialOpts(useTLS, opts)
	if err != nil {
		return nil, err
	}

	headers := map[string]string{"x-api-key": opts.APIKey}

	baseAttrs := []attribute.KeyValue{
		semconv.ServiceName(opts.ServiceName),
		semconv.DeploymentEnvironment(opts.Environment),
		attribute.String("telemetry.sdk.language", "go"),
		attribute.String("telemetry.sdk.version", Version),
	}
	baseAttrs = append(baseAttrs, _state.extraAttrs...)

	res, err := resource.New(ctx,
		resource.WithAttributes(baseAttrs...),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("nirikshaai: create resource: %w", err)
	}

	// ── Traces ────────────────────────────────────────────────────────────────
	traceExp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(grpcAddr),
		otlptracegrpc.WithHeaders(headers),
		otlptracegrpc.WithDialOption(dialOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("nirikshaai: trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	var shutdowns []func(context.Context) error
	shutdowns = append(shutdowns, tp.Shutdown)

	// ── Metrics ───────────────────────────────────────────────────────────────
	var mp *sdkmetric.MeterProvider
	if opts.EnableMetrics {
		metricExp, mErr := otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithEndpoint(grpcAddr),
			otlpmetricgrpc.WithHeaders(headers),
			otlpmetricgrpc.WithDialOption(dialOpts...),
		)
		if mErr == nil {
			mp = sdkmetric.NewMeterProvider(
				sdkmetric.WithResource(res),
				sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
			)
			otel.SetMeterProvider(mp)
			shutdowns = append(shutdowns, mp.Shutdown)
		}
	}
	_ = mp

	// ── Logs ──────────────────────────────────────────────────────────────────
	if opts.EnableLogs {
		logExp, lErr := otlploggrpc.New(ctx,
			otlploggrpc.WithEndpoint(grpcAddr),
			otlploggrpc.WithHeaders(headers),
			otlploggrpc.WithDialOption(dialOpts...),
		)
		if lErr == nil {
			lp := sdklog.NewLoggerProvider(
				sdklog.WithResource(res),
				sdklog.WithProcessor(sdklog.NewBatchProcessor(logExp)),
			)
			global.SetLoggerProvider(lp)
			shutdowns = append(shutdowns, lp.Shutdown)
		}
	}

	_state.baseURL = strings.TrimRight(opts.Endpoint, "/")
	_state.apiKey = opts.APIKey
	_initialized = true

	return func(ctx context.Context) error {
		var lastErr error
		for _, fn := range shutdowns {
			if err := fn(ctx); err != nil {
				lastErr = err
			}
		}
		return lastErr
	}, nil
}

// Flush forces all pending spans, metrics, and log records to be exported.
// Call this before process exit in short-lived or serverless environments.
func Flush(ctx context.Context) error {
	var lastErr error
	if tp, ok := otel.GetTracerProvider().(interface {
		ForceFlush(context.Context) error
	}); ok {
		if err := tp.ForceFlush(ctx); err != nil {
			lastErr = err
		}
	}
	if mp, ok := otel.GetMeterProvider().(interface {
		ForceFlush(context.Context) error
	}); ok {
		if err := mp.ForceFlush(ctx); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Meter returns the global meter for this service.
// Use after calling Init().
func Meter(name string) otelmetric.Meter {
	return otel.Meter(name)
}
