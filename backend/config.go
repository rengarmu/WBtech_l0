package backend

import (
	"log"

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
	Brokers string
	Topic   string
	GroupID string
}

// Главная структура конфурации
type Config struct {
	Postgres   PostgresConfig
	HTTPServer HTTPServerConfig
	Kafka      KafkaConfig
}

// Загружаем конфигурацию из файла config.yaml с помощью Viper
func LoadConfig(path string) Config {
	viper.SetConfigFile(path)
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

	return cfg
}
