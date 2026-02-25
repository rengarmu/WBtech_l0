package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

// Структура для хранения настроек PostgreSQL
type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
}

// Структура для HTTP-сервера
type HTTPServerConfig struct {
	Host string
	Port string
}

// Структура для Kafka
type KafkaConfig struct {
	Brokers  string
	Topic    string
	GroupID  string
	DLQTopic string
}

// Структура для кеша
type CacheConfig struct {
	DefaultTTL time.Duration
	MaxSize    int
}

// Главная структура конфурации
type Config struct {
	Postgres       PostgresConfig
	HTTPServer     HTTPServerConfig
	Kafka          KafkaConfig
	Cache          CacheConfig
	MigrationsPath string // Путь к папке с миграциями
}

// Загружаем конфигурацию из файла config.yaml с помощью Viper
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

	return &cfg
}
