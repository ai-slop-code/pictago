package main

import (
	"github.com/ai-slop-code/pictago/assets"
	"github.com/ai-slop-code/pictago/internal/config"
	"github.com/ai-slop-code/pictago/internal/log"
)

func loadServerConfig(configFileName string) error {
	cfgBytes, err := assets.CfgFs.ReadFile(configFileName)
	if err != nil {
		return err
	}
	if err := config.LoadConfiguration(cfgBytes); err != nil {
		return err
	}
	return nil
}

func init() {
	if err := loadServerConfig(assets.ConfigFileName); err != nil {
		panic(err)
	}
}

var (
	logger = log.NewDefaultLogger()
	cfg    = config.GetConfig()
)

func main() {
	logger.Info(
		"Starting server",
		"grpcPort", cfg.GrpcServer.Port,
		"grpcHost", cfg.GrpcServer.Host,
		"httpPort", cfg.HttpServer.Port,
		"httpHost", cfg.HttpServer.Host,
	)

}
