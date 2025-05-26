package config

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	HTTP HTTPConfig
	JWT  JWTConfig
}

type HTTPConfig struct {
	Port string
}
type JWTConfig struct {
	TTL    int64
	Secret string
}

func LoadConfig() (*Config, error) {
	// Загрузка .env файла, если он существует
	_ = godotenv.Load()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		log.Printf("Warning: %s\n", err)
	}

	var config Config

	// HTTP конфигурация
	config.HTTP.Port = viper.GetString("HTTP_PORT")
	if config.HTTP.Port == "" {
		config.HTTP.Port = "8080"
	}

	config.JWT.Secret = viper.GetString("JWT_SECRET")
	if config.JWT.Secret == "" {
		config.JWT.Secret = "secret"
	}

	config.JWT.TTL = viper.GetInt64("JWT_TTL")
	if config.JWT.TTL == 0 {
		config.JWT.TTL = 600
	}

	return &config, nil
}
