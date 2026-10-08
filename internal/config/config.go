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
	// FrontendURL is the base URL of the deployed web app, used to build
	// links that get emailed out (e.g. a password reset link).
	FrontendURL string
}

type StorageConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
}

// WhatsAppConfig configures the WhatsApp Cloud API. Until PhoneNumberID
// and AccessToken are set, WhatsApp sending (including OTP login) is off.
type WhatsAppConfig struct {
	PhoneNumberID string
	AccessToken   string
	APIVersion    string
	// BaseURL is only overridden for local testing against a fake API.
	BaseURL string
	// OTPTemplate is an approved Authentication-category template with a
	// copy-code button; OTPLanguage is its language code.
	OTPTemplate string
	OTPLanguage string
}

type EmailConfig struct {
	ResendAPIKey string
	FromEmail    string
	FromName     string
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
		Auth:     AuthConfig{JWTSecret: jwtSecret},
		Email: EmailConfig{
			// Resend: EMAIL_FROM's domain must be verified in Resend. Email
			// stays off until both RESEND_API_KEY and EMAIL_FROM are set.
			ResendAPIKey: os.Getenv("RESEND_API_KEY"),
			FromEmail:    os.Getenv("EMAIL_FROM"),
			FromName:     getEnv("EMAIL_FROM_NAME", "CampusDesk"),
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
			BaseURL:       getEnv("WHATSAPP_API_BASE_URL", "https://graph.facebook.com"),
			OTPTemplate:   getEnv("WHATSAPP_OTP_TEMPLATE", "campusdesk_otp"),
			OTPLanguage:   getEnv("WHATSAPP_OTP_LANGUAGE", "en"),
		},
		FrontendURL: getEnv("FRONTEND_URL", "https://13-202-93-187.sslip.io"),
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
