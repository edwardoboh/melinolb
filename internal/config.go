package internal

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Service  ServiceConfig       `yaml:"service"`
	TLS      *TLSConfig          `yaml:"tls"`
	Logging  LoggingConfig       `yaml:"logging"`
	Admin    AdminConfig         `yaml:"admin"`
	Defaults Defaults            `yaml:"defaults"`
	Backends map[string][]string `yaml:"backends"`
	Routes   []Route             `yaml:"routes"`
}

type ServiceConfig struct {
	Name   string `yaml:"name"`
	Env    string `yaml:"env"`
	Listen string `yaml:"listen"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertPath string `yaml:"cert_path"`
	KeyPath  string `yaml:"key_path"`
}

type LoggingConfig struct {
	Level     string `yaml:"level"`
	AccessLog string `yaml:"access_log"`
}

type AdminConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Listen         string `yaml:"listen"`
	healthEndpoint string `yaml:"health_endpoint"`
	metricEndpoint string `yaml:"metric_endpoint"`
}

type Defaults struct {
	ReadTimeout           string `yaml:"read_timeout"`
	WriteTimeout          string `yaml:"write_timeout"`
	BackendConnectTimeout string `yaml:"backend_connect_timeout"`
	BackendReadTimeout    string `yaml:"backend_read_timeout"`
	ConnectionKeepAlive   string `yaml:"connection_keep_alive"`
	RetryOn5xx            bool   `yaml:"retry_on_5xx"`
	MaxRetries            int    `yaml:"max_retries"`
}

type Route struct {
	Id        string           `yaml:"id"`
	Match     MatchConfig      `yaml:"match"`
	Backend   interface{}      `yaml:"backend"`
	LB        string           `yaml:"lb"` // Load Balancing strategy
	Sticky    *StickyConfig    `yaml:"sticky"`
	Health    *HealthConfig    `yaml:"health"`
	Retry     *RetryConfig     `yaml:"retry"`
	RateLimit *RateLimitConfig `yaml:"rate_limit"`
}

type MatchConfig struct {
	Host    string   `yaml:"host"`
	Path    string   `yaml:"path"`
	Methods []string `yaml:"methods"`
}

type StickyConfig struct {
	Enabled    bool   `yaml:"enabled"`
	CookieName string `yaml:"cookie_name"`
	TTL        int32  `yaml:"ttl"`
}

type HealthConfig struct {
	Path     string `yaml:"path"`
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
}

type RetryConfig struct {
	Enabled       bool   `yaml:"enabled"`
	MaxRetries    int    `yaml:"max_retries"`
	PerTryTimeout string `yaml:"per_try_timeout"`
}

type RateLimitConfig struct {
	Enabled          bool `yaml:"enabled"`
	RequestPerSecond int  `yaml:"requests_per_second"`
	Burst            int  `yaml:"burst"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}
