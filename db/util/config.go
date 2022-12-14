package util

import (
	"time"

	"github.com/spf13/viper"
)

// Config will hold all configuration variables - either from file or env vars as read by viper package
// in order to get the values and store them here - we use viper's unmarshaling feature
// viper uses the mapstructure package for unmarshaling values
type Config struct {
	DBDriver             string        `mapstructure:"DB_DRIVER"` // mapstructure tags for unmarshaling values
	DBSource             string        `mapstructure:"DB_SOURCE"`
	HTTPServerAddress    string        `mapstructure:"HTTP_SERVER_ADDRESS"`
	GRPCServerAddress    string        `mapstructure:"GRPC_SERVER_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"TOKEN_SYMMETRIC_KEY"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
}

// LoadConfig reads configuration from file in the path if it exists or overrides the config values with env vars if provided
func LoadConfig(path string) (config Config, err error) {
	// we tell viper where the config file is
	viper.AddConfigPath(path)
	// we tell viper what the config file is named
	viper.SetConfigName("app")
	// we tell viper what file type the config file is
	viper.SetConfigType("env")

	// if corresponding env vars exist, override values in the config file with those of the env vars
	viper.AutomaticEnv()

	// start reading in config values
	err = viper.ReadInConfig() // remember named return variable
	if err != nil {
		return // named return
	}

	// unmarshals the values
	err = viper.Unmarshal(&config)
	return // named return
}
