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
	Environment         EnvironmentConfig         `mapstructure:"environment"`
	Server              ServerConfig              `mapstructure:"server"`
	Database            DatabaseConfig            `mapstructure:"database"`
	DatabaseConnections DatabaseConnectionsConfig `mapstructure:"database_connections"`
	Redis               RedisConfig               `mapstructure:"redis"`
	JWT                 JWTConfig                 `mapstructure:"jwt"`
	RateLimit           RateLimitConfig           `mapstructure:"ratelimit"`
	Email               EmailConfig               `mapstructure:"email"`
	Tracing             Tracing                   `mapstructure:"tracing"`
}

type EnvironmentConfig struct {
	Current string `mapstructure:"current"`
}

type ServerConfig struct {
	Port         string `mapstructure:"port"`
	CookieSecure bool   `mapstructure:"cookiesecure"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
}

type DatabaseConfig struct {
	Path     string `mapstructure:"path"`
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}

type DatabaseConnectionsConfig struct {
	MaxOpenConns    int `mapstructure:"max_open_conns"`
	MaxIdleConns    int `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int `mapstructure:"max_life_time"`
	ConnMaxIdleTime int `mapstructure:"max_idle_time"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	SecretKey string `mapstructure:"secretkey"`
}

type RateLimitConfig struct {
	MaxRequests int           `mapstructure:"maxrequests"`
	Window      time.Duration `mapstructure:"window"`
}

type EmailConfig struct {
	SMTHost  string `mapstructure:"smtHost"`
	SMTPort  string `mapstructure:"smtPort"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

type Tracing struct {
	Enabled     bool   `mapstructure:"enabled"`
	ServiceName string `mapstructure:"service_name"`
	Endpoint    string `mapstructure:"endpoint"` // Jaeger endpoint
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
