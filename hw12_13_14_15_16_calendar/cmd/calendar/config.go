package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	storageTypeMemory = "memory"
	storageTypeSQL    = "sql"
)

type Config struct {
	Logger  LoggerConf  `yaml:"logger"`
	HTTP    HTTPConf    `yaml:"http"`
	Storage StorageConf `yaml:"storage"`
}

type LoggerConf struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

type HTTPConf struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type StorageConf struct {
	Type string `yaml:"type"`
	DSN  string `yaml:"dsn"`
}

func NewConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if err := config.validate(); err != nil {
		return Config{}, fmt.Errorf("validate config: %w", err)
	}

	return config, nil
}

func (c Config) validate() error {
	switch c.Logger.Level {
	case "error", "warn", "info", "debug":
	default:
		return fmt.Errorf("unsupported logger level %q", c.Logger.Level)
	}
	if c.Logger.File == "" {
		return fmt.Errorf("logger file is required")
	}

	if c.HTTP.Host == "" {
		return fmt.Errorf("http host is required")
	}

	if c.HTTP.Port < 1 || c.HTTP.Port > 65535 {
		return fmt.Errorf("http port must be between 1 and 65535")
	}

	switch c.Storage.Type {
	case storageTypeMemory:
	case storageTypeSQL:
		if c.Storage.DSN == "" {
			return fmt.Errorf("storage dsn is required for sql storage")
		}
	default:
		return fmt.Errorf("unsupported storage type %q", c.Storage.Type)
	}

	return nil
}
