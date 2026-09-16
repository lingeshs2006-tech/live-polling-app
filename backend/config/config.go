package config

import (
	"os"
)

type Config struct {
	Port          string
	MongoURI      string
	MongoDatabase string
	RedisAddr     string
	RedisPassword string
	JWTSecret     string
	ClientOrigin  string
}

func Load() *Config {
	return &Config{
		Port:          getEnv("PORT", "8080"),
		MongoURI:      getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase: getEnv("MONGO_DB", "livepoll"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		JWTSecret:     getEnv("JWT_SECRET", "super-secret-change-me-in-prod"),
		ClientOrigin:  getEnv("CLIENT_ORIGIN", "http://localhost:5173"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}