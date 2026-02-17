package config

import "os"

type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPassword     string
	DBName         string
	JWTSecret      string
	BotAPIKey      string
	ServerPort     string
	WebhookBaseURL string
	PollInterval   string
}

func Load() *Config {
	return &Config{
		DBHost:         getEnv("DB_HOST", "localhost"),
		DBPort:         getEnv("DB_PORT", "5432"),
		DBUser:         getEnv("DB_USER", "postgres"),
		DBPassword:     getEnv("DB_PASSWORD", "postgres"),
		DBName:         getEnv("DB_NAME", "quizgame"),
		JWTSecret:      getEnv("JWT_SECRET", "super-secret-key-change-me"),
		BotAPIKey:      getEnv("BOT_API_KEY", "bot-api-key-change-me"),
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		WebhookBaseURL: getEnv("WEBHOOK_BASE_URL", ""),
		PollInterval:   getEnv("POLL_INTERVAL", "2"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
