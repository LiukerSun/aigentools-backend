package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBPort        string
	RedisAddr     string
	RedisPort     string
	RedisPassword string
	JWTSecret     string

	// Log configuration
	LogLevel      string
	LogFilename   string
	LogMaxSize    int
	LogMaxBackups int
	LogMaxAge     int
	LogCompress   bool
}

func (c *Config) DSN() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort)
}

func (c *Config) RedisFullAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisAddr, c.RedisPort)
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		// Ignore error if .env file is not found
		if !os.IsNotExist(err) {
			return nil, err
		}
	}

	return &Config{
		DBHost:        os.Getenv("DB_HOST"),
		DBUser:        os.Getenv("DB_USER"),
		DBPassword:    os.Getenv("DB_PASSWORD"),
		DBName:        os.Getenv("DB_NAME"),
		DBPort:        os.Getenv("DB_PORT"),
		RedisAddr:     os.Getenv("REDIS_HOST"),
		RedisPort:     os.Getenv("REDIS_PORT"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		JWTSecret:     os.Getenv("JWT_SECRET"),

		LogLevel:      getEnv("LOG_LEVEL", "INFO"),
		LogFilename:   getEnv("LOG_FILENAME", "logs/app.log"),
		LogMaxSize:    getEnvAsInt("LOG_MAX_SIZE", 100),
		LogMaxBackups: getEnvAsInt("LOG_MAX_BACKUPS", 3),
		LogMaxAge:     getEnvAsInt("LOG_MAX_AGE", 28),
		LogCompress:   getEnvAsBool("LOG_COMPRESS", true),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := strconv.Atoi(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if valueStr, exists := os.LookupEnv(key); exists {
		if value, err := strconv.ParseBool(valueStr); err == nil {
			return value
		}
	}
	return defaultValue
}
