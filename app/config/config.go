package config

// PROPRIETARY AND CONFIDENTIAL
// This code contains trade secrets and confidential material of Finimen Sniper / FSC.
// Any unauthorized use, disclosure, or duplication is strictly prohibited.
// © 2025 Finimen Sniper / FSC. All rights reserved.

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Environment EnvironmentConfig
	Server      ServerConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	RateLimit   RateLimitConfig
	Email       EmailConfig
}

type EnvironmentConfig struct {
	Current string
}

type ServerConfig struct {
	Port         string
	CookieSecure bool
}

type DatabaseConfig struct {
	Path string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	SecretKey string
}

type RateLimitConfig struct {
	MaxRequests int
	Window      time.Duration
}

type EmailConfig struct {
	SMTHost  string
	SMTPort  string
	Username string
	Password string
	From     string
}

func LoadConfig() (config Config, err error) {
	viper.SetConfigName("app")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, err
		}
	}

	viper.AutomaticEnv()

	viper.SetDefault("environment.current", "development")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("database.path", "./burgers.db")
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("jwt.secretkey", "your_default_secret_change_in_production")
	viper.SetDefault("ratelimit.maxrequests", 100)
	viper.SetDefault("ratelimit.window", time.Minute)

	err = viper.Unmarshal(&config)
	if err != nil {
		return config, err
	}

	if config.JWT.SecretKey == "your_default_secret_change_in_production" {
		log.Println("WARNING: Using default JWT secret key. This is insecure for production.")
	}

	return config, nil
}
