// Package config loads application configuration from .env and process env.
// Strongly typed Config struct; missing required fields cause a fatal startup error.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the top-level application configuration.
type Config struct {
	App     AppConfig
	MySQL   MySQLConfig
	Redis   RedisConfig
	Auth    AuthConfig
	AI      AIConfig
	Mock    MockConfig
	Runner  RunnerConfig
	Logging LoggingConfig
}

type AppConfig struct {
	Env            string
	Host           string
	Port           int
	BaseURL        string
	FrontendOrigin string
}

type MySQLConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	TestDatabase    string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func (m MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
		m.User, m.Password, m.Host, m.Port, m.Database)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
	TestDB   int
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type AuthConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	BcryptCost    int
}

type AIConfig struct {
	Enabled          bool
	Provider         string
	ModelAttributor  string
	ModelCompleter   string
	ModelPrioritizer string
	AnthropicKey     string
	OpenAIKey        string
	DailyLimitUSD    float64
	MonthlyLimitUSD  float64
	MaxTokens        int
	TimeoutSeconds   int
	CacheTTLSeconds  int
}

type MockConfig struct {
	DefaultDelayMs    int
	RecordHits        bool
	BodyTruncateBytes int
}

type RunnerConfig struct {
	DefaultTimeout  time.Duration
	DefaultMode     string
	DefaultConc     int
	MaxConcurrency  int
}

type LoggingConfig struct {
	Level  string
	Format string
}

// Load reads .env (if present) and process env into a Config struct.
func Load() (*Config, error) {
	v := viper.New()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read .env if it exists (does not override real env vars)
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("..")
	v.AddConfigPath("../..")
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read .env: %w", err)
		}
	}

	cfg := &Config{
		App: AppConfig{
			Env:            v.GetString("APP_ENV"),
			Host:           v.GetString("APP_HOST"),
			Port:           v.GetInt("APP_PORT"),
			BaseURL:        v.GetString("APP_BASE_URL"),
			FrontendOrigin: v.GetString("FRONTEND_ORIGIN"),
		},
		MySQL: MySQLConfig{
			Host:            v.GetString("MYSQL_HOST"),
			Port:            v.GetInt("MYSQL_PORT"),
			User:            v.GetString("MYSQL_USER"),
			Password:        v.GetString("MYSQL_PASSWORD"),
			Database:        v.GetString("MYSQL_DATABASE"),
			TestDatabase:    v.GetString("MYSQL_TEST_DATABASE"),
			MaxOpenConns:    v.GetInt("MYSQL_MAX_OPEN_CONNS"),
			MaxIdleConns:    v.GetInt("MYSQL_MAX_IDLE_CONNS"),
			ConnMaxLifetime: time.Duration(v.GetInt("MYSQL_CONN_MAX_LIFETIME")) * time.Second,
		},
		Redis: RedisConfig{
			Host:     v.GetString("REDIS_HOST"),
			Port:     v.GetInt("REDIS_PORT"),
			Password: v.GetString("REDIS_PASSWORD"),
			DB:       v.GetInt("REDIS_DB"),
			TestDB:   v.GetInt("REDIS_TEST_DB"),
		},
		Auth: AuthConfig{
			AccessSecret:  v.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: v.GetString("JWT_REFRESH_SECRET"),
			AccessTTL:     v.GetDuration("JWT_ACCESS_TTL"),
			RefreshTTL:    v.GetDuration("JWT_REFRESH_TTL"),
			BcryptCost:    v.GetInt("BCRYPT_COST"),
		},
		AI: AIConfig{
			Enabled:          v.GetBool("AI_ENABLED"),
			Provider:         v.GetString("AI_PROVIDER"),
			ModelAttributor:  v.GetString("AI_MODEL_ATTRIBUTOR"),
			ModelCompleter:   v.GetString("AI_MODEL_COMPLETER"),
			ModelPrioritizer: v.GetString("AI_MODEL_PRIORITIZER"),
			AnthropicKey:     v.GetString("ANTHROPIC_API_KEY"),
			OpenAIKey:        v.GetString("OPENAI_API_KEY"),
			DailyLimitUSD:    v.GetFloat64("AI_DAILY_LIMIT_USD"),
			MonthlyLimitUSD:  v.GetFloat64("AI_MONTHLY_LIMIT_USD"),
			MaxTokens:        v.GetInt("AI_MAX_TOKENS"),
			TimeoutSeconds:   v.GetInt("AI_TIMEOUT_SECONDS"),
			CacheTTLSeconds:  v.GetInt("AI_CACHE_TTL_SECONDS"),
		},
		Mock: MockConfig{
			DefaultDelayMs:    v.GetInt("MOCK_DEFAULT_DELAY_MS"),
			RecordHits:        v.GetBool("MOCK_RECORD_HITS"),
			BodyTruncateBytes: v.GetInt("MOCK_BODY_TRUNCATE_BYTES"),
		},
		Runner: RunnerConfig{
			DefaultTimeout: time.Duration(v.GetInt("RUNNER_DEFAULT_TIMEOUT_SECONDS")) * time.Second,
			DefaultMode:    v.GetString("RUNNER_DEFAULT_MODE"),
			DefaultConc:    v.GetInt("RUNNER_DEFAULT_CONCURRENCY"),
			MaxConcurrency: v.GetInt("RUNNER_MAX_CONCURRENCY"),
		},
		Logging: LoggingConfig{
			Level:  v.GetString("LOG_LEVEL"),
			Format: v.GetString("LOG_FORMAT"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.App.Port <= 0 || c.App.Port > 65535 {
		return fmt.Errorf("invalid APP_PORT: %d", c.App.Port)
	}
	if c.MySQL.Host == "" || c.MySQL.Database == "" {
		return fmt.Errorf("MYSQL_HOST and MYSQL_DATABASE required")
	}
	if c.Redis.Host == "" {
		return fmt.Errorf("REDIS_HOST required")
	}
	if c.App.Env == "production" {
		if c.Auth.AccessSecret == "" || len(c.Auth.AccessSecret) < 32 {
			return fmt.Errorf("JWT_ACCESS_SECRET must be >= 32 chars in production")
		}
	}
	return nil
}
