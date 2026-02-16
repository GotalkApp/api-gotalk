package config

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	App    AppConfig
	DB     DBConfig
	Redis  RedisConfig
	JWT    JWTConfig
	MinIO  MinIOConfig
	CORS   CORSConfig
	SMTP   SMTPConfig
	Google GoogleConfig
}

type AppConfig struct {
	Env  string
	Port string
}

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// DSN returns the PostgreSQL connection string
func (d DBConfig) DSN() string {
	return "host=" + d.Host +
		" user=" + d.User +
		" password=" + d.Password +
		" dbname=" + d.Name +
		" port=" + d.Port +
		" sslmode=" + d.SSLMode +
		" TimeZone=Asia/Ho_Chi_Minh"
}

// URL returns the PostgreSQL connection URL (for golang-migrate)
func (d DBConfig) URL() string {
	return "postgres://" + d.User + ":" + d.Password +
		"@" + d.Host + ":" + d.Port +
		"/" + d.Name + "?sslmode=" + d.SSLMode
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
}

// Addr returns the Redis address
func (r RedisConfig) Addr() string {
	return r.Host + ":" + r.Port
}

type JWTConfig struct {
	Secret string
	Expiry time.Duration
}

type MinIOConfig struct {
	Endpoint  string
	PublicURL string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type CORSConfig struct {
	Origins []string
}

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
}

type GoogleConfig struct {
	ClientID     string
	ClientSecret string
}

// Load reads configuration from .env file and environment variables
func Load() *Config {
	// Load .env file (ignore error if not exists - e.g. in Docker)
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️  No .env file found, reading from environment variables")
	}

	jwtExpiry, err := time.ParseDuration(getEnv("JWT_EXPIRY", "24h"))
	if err != nil {
		jwtExpiry = 24 * time.Hour
	}

	return &Config{
		App: AppConfig{
			Env:  getEnv("APP_ENV", "development"),
			Port: getEnv("APP_PORT", "8080"),
		},
		DB: DBConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "gotalk"),
			Password: getEnv("DB_PASSWORD", "gotalk"),
			Name:     getEnv("DB_NAME", "gotalk"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "default-secret"),
			Expiry: jwtExpiry,
		},
		MinIO: MinIOConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			PublicURL: getEnv("MINIO_PUBLIC_URL", ""),
			AccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    getEnv("MINIO_BUCKET", "gotalk-media"),
			UseSSL:    getEnv("MINIO_USE_SSL", "false") == "true",
		},
		CORS: CORSConfig{
			Origins: strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000"), ","),
		},
		SMTP: SMTPConfig{
			Host:     getEnv("SMTP_HOST", "mailpit"),
			Port:     getEnv("SMTP_PORT", "1025"),
			Username: getEnv("SMTP_USERNAME", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
			From:     getEnv("SMTP_FROM", "noreply@gotalk.local"),
			FromName: getEnv("SMTP_FROM_NAME", "GoTalk"),
		},
		Google: GoogleConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		},
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
