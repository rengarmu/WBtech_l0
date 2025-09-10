package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type MainConfig struct {
	Kafka KfkConfig  `yaml:"kafka"`
	Psql  PsqlConfig `yaml:"psql"`
	HTTP  HTTPConfig `yaml:"http"`
}

type KfkConfig struct {
	KafkaBrokers string `yaml:"brokers"`
	Topic        string `yaml:"topic"`
	KafkaGroupID string `yaml:"group_id"`
}

type PsqlConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type HTTPConfig struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

// LoadConfig заполняет структуру MainConfig данными из файла config.yaml
func LoadConfig() (*MainConfig, error) {
	cfg := &MainConfig{}
	file, err := os.ReadFile("wb_tech/config.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	if err := yaml.Unmarshal(file, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return cfg, nil
}
