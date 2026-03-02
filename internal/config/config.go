package config

import (
	"bytes"
	"fmt"
	"strings"
	"sync"

	"github.com/ai-slop-code/pictago/internal/log"
	"github.com/ai-slop-code/pictago/internal/utils"
	"github.com/spf13/viper"
)

var (
	mutex    sync.Mutex
	Cfg      *Config
	logger   = log.NewDefaultLogger()
	validate = utils.NewValidator()
)

type ServerConfig struct {
	Port int    `mapstructure:"port" validate:"required,min=1,max=65535,notblank"`
	Host string `mapstructure:"host" validate:"required,hostname|ip,notblank"`
}

func (cfg *ServerConfig) ServerHost() string {
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
}

type GrpcServerConfig struct {
	ServerConfig `mapstructure:",squash"`
}

type HttpServerConfig struct {
	ServerConfig `mapstructure:",squash"`
}

type DatabaseConfig struct {
	ServerConfig  `mapstructure:",squash"`
	Username      string `mapstructure:"username" validate:"required,min=1,max=32"`
	Password      string `mapstructure:"password" validate:"required,min=1,max=32"`
	Driver        string `mapstructure:"driver" validate:"required,oneof=mysql postgres"`
	MigrationsDir string `mapstructure:"migrations_dir" validate:"required,min=1,max=32"`
}

func (cfg *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"%s://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Driver,
		cfg.Username,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Driver,
	)
}

type Config struct {
	GrpcServer GrpcServerConfig `mapstructure:"grpc" validate:"required"`
	HttpServer HttpServerConfig `mapstructure:"http" validate:"required"`
	Database   DatabaseConfig   `mapstructure:"database" validate:"required"`
}

// GetConfig returns the current configuration.
// It is safe for concurrent use.
func GetConfig() *Config {
	mutex.Lock()
	defer mutex.Unlock()
	return Cfg
}

func LoadConfiguration(config []byte) error {
	mutex.Lock()
	defer mutex.Unlock()
	viper.SetConfigType("toml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := viper.ReadConfig(bytes.NewBuffer(config)); err != nil {
		logger.Error("Failed to read config", "error", err)
		return err
	}
	loadedConfig := Config{}
	if err := viper.Unmarshal(&loadedConfig); err != nil {
		logger.Error("Failed to unmarshal configuration", "error", err)
		return err
	}
	if err := validate.Struct(loadedConfig); err != nil {
		logger.Error("Configuration validation failed", "error", err)
		return err
	}
	Cfg = &loadedConfig
	return nil
}
