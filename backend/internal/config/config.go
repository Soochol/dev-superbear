package config

import "os"

type Config struct {
	Port               string
	DatabaseURL        string
	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string
	AllowedOrigins     []string
	Env                string
	KISAppKey          string
	KISAppSecret       string
	KISBaseURL         string
	DARTApiKey         string
}

func Load() *Config {
	return &Config{
		Port:               getEnv("PORT", "8080"),
		DatabaseURL:        getEnv("DATABASE_URL", "postgresql://nexus:nexus@localhost:5432/nexus?sslmode=disable"),
		JWTSecret:          getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		AllowedOrigins:     []string{getEnv("ALLOWED_ORIGIN", "http://localhost:3000")},
		Env:                getEnv("APP_ENV", "development"),
		KISAppKey:          getEnv("KIS_APP_KEY", ""),
		KISAppSecret:       getEnv("KIS_APP_SECRET", ""),
		KISBaseURL:         getEnv("KIS_BASE_URL", "https://openapi.koreainvestment.com:9443"),
		DARTApiKey:         getEnv("DART_API_KEY", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
