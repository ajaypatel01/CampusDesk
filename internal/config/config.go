package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Auth     AuthConfig
	Email    EmailConfig
	Storage  StorageConfig
	WhatsApp WhatsAppConfig
}

type StorageConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
}

type WhatsAppConfig struct {
	PhoneNumberID string
	AccessToken   string
	APIVersion    string
}

type EmailConfig struct {
	SendGridAPIKey string
	FromEmail      string
	FromName       string
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type DatabaseConfig struct {
	URL string
}

type AuthConfig struct {
	JWTSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %w", err)
	}

	readSec, _ := strconv.Atoi(getEnv("READ_TIMEOUT_SEC", "15"))
	writeSec, _ := strconv.Atoi(getEnv("WRITE_TIMEOUT_SEC", "15"))

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://campusdesk:campusdesk@localhost:5432/campusdesk?sslmode=disable"
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}

	return &Config{
		Server: ServerConfig{
			Host:         getEnv("HOST", "0.0.0.0"),
			Port:         port,
			ReadTimeout:  time.Duration(readSec) * time.Second,
			WriteTimeout: time.Duration(writeSec) * time.Second,
		},
		Database: DatabaseConfig{URL: dbURL},
		Auth:  AuthConfig{JWTSecret: jwtSecret},
		Email: EmailConfig{
			SendGridAPIKey: os.Getenv("SENDGRID_API_KEY"),
			FromEmail:      getEnv("EMAIL_FROM", "noreply@campusdesk.app"),
			FromName:       getEnv("EMAIL_FROM_NAME", "CampusDesk"),
		},
		Storage: StorageConfig{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			Region:          getEnv("S3_REGION", "ap-south-1"),
			Bucket:          os.Getenv("S3_BUCKET"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY"),
			SecretAccessKey: os.Getenv("S3_SECRET_KEY"),
			UseSSL:          getEnv("S3_USE_SSL", "true") == "true",
		},
		WhatsApp: WhatsAppConfig{
			PhoneNumberID: os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
			AccessToken:   os.Getenv("WHATSAPP_ACCESS_TOKEN"),
			APIVersion:    getEnv("WHATSAPP_API_VERSION", "v19.0"),
		},
	}, nil
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
