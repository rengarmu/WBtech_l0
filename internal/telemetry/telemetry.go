// Package telemetry предоставляет настройки метрик Prometheus и трассировки
package telemetry

import (
	"context"
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// Метрики Prometheus
var (
	OrdersProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_processed_total",
			Help: "Total number of orders processed",
		},
		[]string{"source", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	KafkaMessageProcessDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_message_process_duration_seconds",
			Help:    "Duration of Kafka message processing",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic", "status"},
	)
)

// InitTracer инициализирует OTLP экспортер трейсов.
// Параметр otlpEndpoint должен содержать адрес OTLP collector (например, "localhost:4318").
func InitTracer(serviceName, otlpEndpoint string) func() {
	ctx := context.Background()

	// Создаём OTLP HTTP экспортер
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		otlptracehttp.WithInsecure(), // если collector без TLS (для разработки)
	)
	if err != nil {
		log.Fatalf("failed to create OTLP exporter: %v", err)
	}

	// Ресурс с именем сервиса
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			attribute.String("environment", "production"),
		),
	)
	if err != nil {
		log.Fatalf("failed to create resource: %v", err)
	}

	// Провайдер трассировки
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // для разработки
	)
	otel.SetTracerProvider(tp)

	return func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}
}
