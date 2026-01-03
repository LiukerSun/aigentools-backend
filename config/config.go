package config

import (
	"fmt"
	"os"

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
	}, nil
}
