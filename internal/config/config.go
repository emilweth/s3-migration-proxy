package config

import (
	"strings"

	"github.com/spf13/viper"
)

type S3Config struct {
	BucketName      string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	Protocol        string
}

type HTTPConfig struct {
	Port int
}

type Configuration struct {
	S3 struct {
		Source             S3Config
		Target             S3Config
		CacheErrorDuration int
	}
	HTTP HTTPConfig
}

func LoadConfig(filePath string) (*Configuration, error) {
	// Define the key replacer: replace dots with underscores
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// AutomaticEnv will tell viper to look for any environment variables
	// that match the keys in the config file.
	viper.AutomaticEnv()

	// Set config file path and type
	viper.SetConfigFile(filePath)
	viper.SetConfigType("yaml")

	// Read the configuration file
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Configuration
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
