package config

import (
    "errors"
    "io/fs"
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
    APIEnabled bool

    DiscordToken   string
    DiscordGuildID string
}

func Load() (*Config, error) {
    // Load .env file if present; in containers the environment is set directly
    if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
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
        APIEnabled: getEnvOrDefault("API_ENABLED", "true") != "false",

        DiscordToken:   os.Getenv("DISCORD_TOKEN"),
        DiscordGuildID: os.Getenv("DISCORD_GUILD_ID"),
    }, nil
}

func getEnvOrDefault(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}