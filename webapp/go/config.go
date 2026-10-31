package isureal

import (
	"net"
	"net/url"
	"os"

	"github.com/jackc/pgx/v5"
)

type Config struct {
	Region     string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	LogLevel   string
}

func ConfigFromEnv() *Config {
	return &Config{
		Region:     envOrDefault("ISUREAL_REGION", "asia"),
		DBHost:     envOrDefault("ISUREAL_DB_HOST", "127.0.0.1"),
		DBPort:     envOrDefault("ISUREAL_DB_PORT", "5432"),
		DBUser:     envOrDefault("ISUREAL_DB_USER", "isureal"),
		DBPassword: envOrDefault("ISUREAL_DB_PASSWORD", "isureal"),
		DBName:     envOrDefault("ISUREAL_DB_NAME", "isureal"),
		LogLevel:   envOrDefault("ISUREAL_LOG_LEVEL", "debug"),
	}
}

func (c *Config) DatabaseDSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.DBUser, c.DBPassword),
		Host:   net.JoinHostPort(c.DBHost, c.DBPort),
		Path:   "/" + c.DBName,
		// DB is ~100ms away from s2/s3; skip the per-connection prepare round trip (#21)
		// No TLS: the manual does not ask for it, and the connection still goes through the router (#60)
		RawQuery: "default_query_exec_mode=exec&sslmode=disable",
	}
	return u.String()
}

// binaryImage goes first in the args of statements that carry image bytes, so bytea is sent and received in binary
// instead of \x hex text (#60). Not the default mode: on s2/s3 each new statement and connection costs an extra
// ~100ms describe round trip (#21). Not cache_statement: a named prepared statement can fail with "cached plan must
// not change result type" after /api/initialize alters a table; this one re-parses an unnamed statement every time.
const binaryImage = pgx.QueryExecModeCacheDescribe

func envOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
