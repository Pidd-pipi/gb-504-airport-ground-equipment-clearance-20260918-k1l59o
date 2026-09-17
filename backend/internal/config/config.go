package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv             string
	ServerPort         string
	DBHost             string
	DBPort             string
	DBName             string
	DBUser             string
	DBPassword         string
	DBSSLMode          string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	JWTSecret          string
	JWTExpireHours     int
	RateLimitPerMinute int
	CORSOrigins        []string
	SeedDemoData       bool
}

func Load() *Config {
	return &Config{
		AppEnv:             getEnv("APP_ENV", "development"),
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		DBHost:             getEnv("DB_HOST", "127.0.0.1"),
		DBPort:             getEnv("DB_PORT", "5432"),
		DBName:             getEnv("DB_NAME", "ground_clearance"),
		DBUser:             getEnv("DB_USER", "ground_clearance"),
		DBPassword:         getEnv("DB_PASSWORD", "ground_clearance_pwd"),
		DBSSLMode:          getEnv("DB_SSLMODE", "disable"),
		RedisAddr:          getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvIntAllowZero("REDIS_DB", 0),
		JWTSecret:          getEnv("JWT_SECRET", "change_me_to_a_long_random_string"),
		JWTExpireHours:     getEnvInt("JWT_EXPIRE_HOURS", 24),
		RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 240),
		CORSOrigins:        parseCSV(getEnv("APP_CORS_ORIGINS", "http://localhost:18504,http://127.0.0.1:18504")),
		SeedDemoData:       getEnvBool("SEED_DEMO_DATA", false),
	}
}

func (c *Config) Validate() error {
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if strings.EqualFold(c.AppEnv, "production") {
		secret := strings.ToLower(c.JWTSecret)
		if strings.Contains(secret, "change_me") || strings.Contains(secret, "replace_with") {
			return fmt.Errorf("JWT_SECRET placeholder is forbidden in production")
		}
		if c.SeedDemoData {
			return fmt.Errorf("SEED_DEMO_DATA must be false in production")
		}
	}
	if len(c.CORSOrigins) == 0 {
		return fmt.Errorf("APP_CORS_ORIGINS must contain at least one origin")
	}
	for _, origin := range c.CORSOrigins {
		if origin == "*" {
			return fmt.Errorf("wildcard CORS origin is forbidden")
		}
	}
	return nil
}

func (c *Config) DBDSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Shanghai",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

func (c *Config) JWTExpireDuration() time.Duration {
	return time.Duration(c.JWTExpireHours) * time.Hour
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, err := strconv.Atoi(os.Getenv(key))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func getEnvIntAllowZero(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseCSV(value string) []string {
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
