package main

import (
	"context"
	"log"
	"net/http"
	"os"

	nirikshaai "github.com/san-data-systems/niriksha-sdk-go"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	tracer          = otel.Tracer("product-catalog")
	productCounter  metric.Int64Counter
)

type Product struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

var products = []Product{
	{ID: "p1", Name: "Widget A", Price: 9.99},
	{ID: "p2", Name: "Widget B", Price: 19.99},
}

func listProducts(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "catalog.listProducts")
	defer span.End()

	span.SetAttributes(attribute.Int("products.count", len(products)))
	_ = ctx

	c.JSON(http.StatusOK, gin.H{"products": products})
}

func createProduct(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "catalog.createProduct")
	defer span.End()

	var p Product
	if err := c.ShouldBindJSON(&p); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	span.SetAttributes(
		attribute.String("product.id", p.ID),
		attribute.String("product.name", p.Name),
	)
	products = append(products, p)
	productCounter.Add(ctx, 1)

	c.JSON(http.StatusCreated, p)
}

func main() {
	ctx := context.Background()
	shutdown, err := nirikshaai.Init(ctx, nirikshaai.Options{
		Endpoint:      "https://app.niriksha.ai",
		OTLPEndpoint:  "grpc-ingest.niriksha.ai:4317",
		APIKey:        os.Getenv("NIRIKSHA_API_KEY"),
		ServiceName:   "product-catalog",
		Environment:   "production",
		EnableMetrics: true,
		EnableLogs:    true,
	})
	if err != nil {
		log.Fatalf("nirikshaai.Init: %v", err)
	}
	defer shutdown(ctx)

	meter := nirikshaai.Meter("product-catalog")
	productCounter, err = meter.Int64Counter("products.created",
		metric.WithDescription("Total products added to the catalog"))
	if err != nil {
		log.Fatalf("counter: %v", err)
	}

	r := gin.Default()
	r.Use(otelgin.Middleware("product-catalog"))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/products", listProducts)
	r.POST("/products", createProduct)

	log.Println("product-catalog listening on :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatal(err)
	}
}
