// Package config предоставляет функции для загрузки конфигурации приложения
package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

// PostgresConfig содержит настройки подключения к PostgreSQL
type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// HTTPServerConfig содержит настройки HTTP-сервера
type HTTPServerConfig struct {
	Host string
	Port string
}

// KafkaConfig содержит настройки подключения к Kafka
type KafkaConfig struct {
	Brokers  string
	Topic    string
	GroupID  string
	DLQTopic string
}

// CacheConfig содержит настройки in-memory кеша
type CacheConfig struct {
	DefaultTTL time.Duration
	MaxSize    int
}

// TelemetryConfig настройки телеметрии (метрики и трассировка)
type TelemetryConfig struct {
	OTLPEndpoint string // URL для OTLP экспортера
	MetricsPort  string // порт для экспорта метрик Prometheus
}

// Config объединяет все настройки приложения
type Config struct {
	Postgres       PostgresConfig
	HTTPServer     HTTPServerConfig
	Kafka          KafkaConfig
	Cache          CacheConfig
	MigrationsPath string // Путь к папке с миграциями
	Telemetry      TelemetryConfig
}

// LoadConfig загружает конфигурацию из YAML-файла с помощью Viper
func LoadConfig(path string) *Config {
	viper.SetConfigFile(path)
	viper.AutomaticEnv() // поддержка переменных окружения
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	var cfg Config
	cfg.Postgres = PostgresConfig{
		Host:     viper.GetString("postgresql.host"),
		Port:     viper.GetString("postgresql.port"),
		User:     viper.GetString("postgresql.user"),
		Password: viper.GetString("postgresql.password"),
		Database: viper.GetString("postgresql.database"),
	}
	cfg.HTTPServer = HTTPServerConfig{
		Host: viper.GetString("http_server.host"),
		Port: viper.GetString("http_server.port"),
	}
	cfg.Kafka = KafkaConfig{
		Brokers: viper.GetString("kafka.brokers"),
		Topic:   viper.GetString("kafka.topic"),
		GroupID: viper.GetString("kafka.group_id"),
	}

	cfg.Cache = CacheConfig{
		DefaultTTL: viper.GetDuration("cache.default_ttl"),
		MaxSize:    viper.GetInt("cache.max_size"),
	}
	cfg.MigrationsPath = viper.GetString("migrations_path")

	cfg.Telemetry = TelemetryConfig{
		OTLPEndpoint: viper.GetString("telemetry.otlp_endpoint"),
		MetricsPort:  viper.GetString("telemetry.metrics_port"),
	}
	return &cfg
}
