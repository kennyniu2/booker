package config

import (
    "os"
    "github.com/joho/godotenv"
)

type Config struct {
    DBHost     string
    DBPort     string
    DBUser     string
    DBPassword string
    DBName     string
    SecretKey  string
    Port       string
}

func Load() (*Config, error) {
    // Load .env file
    err := godotenv.Load()
    if err != nil {
        return nil, err
    }

    return &Config{
        DBHost:     os.Getenv("DB_HOST"),
        DBPort:     os.Getenv("DB_PORT"),
        DBUser:     os.Getenv("DB_USER"),
        DBPassword: os.Getenv("DB_PASSWORD"),
        DBName:     os.Getenv("DB_NAME"),
        SecretKey:  os.Getenv("SECRET_KEY"),
        Port:       getEnvOrDefault("PORT", "8080"),
    }, nil
}

func getEnvOrDefault(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}