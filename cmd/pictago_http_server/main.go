package main

import (
	"fmt"
	"net"

	"github.com/ai-slop-code/pictago/assets"
	"github.com/ai-slop-code/pictago/internal/config"
	"github.com/ai-slop-code/pictago/internal/log"
	v1gw "github.com/ai-slop-code/pictago/internal/proto/v1/grpc/gateway"
	"github.com/ai-slop-code/pictago/pkg/authentication"
	"github.com/ai-slop-code/pictago/pkg/file"
	"github.com/ai-slop-code/pictago/pkg/user_configuration"
	"github.com/ai-slop-code/pictago/pkg/user_info"
	"github.com/ai-slop-code/pictago/pkg/user_management"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
)

func loadServerConfig(configFileName string) error {
	cfgBytes, err := assets.CfgFs.ReadFile(configFileName)
	if err != nil {
		return err
	}
	return config.LoadConfiguration(cfgBytes)
}

func startServer(cfg config.Config) error {
	logger := log.NewDefaultLogger()

	lis, err := net.Listen("tcp", cfg.GrpcServer.ServerHost())
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.GrpcServer.ServerHost(), err)
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryServerInterceptor(&logger),
		),
	)

	v1gw.RegisterAuthenticationServer(server, authentication.NewAuthenticationServer())
	v1gw.RegisterFileUploadServer(server, file.NewFileServer())
	v1gw.RegisterUserConfigurationServer(server, user_configuration.NewUserConfigurationServer())
	v1gw.RegisterUserInfoServer(server, user_info.NewUserInfoServer())
	v1gw.RegisterUserManagementServer(server, user_management.NewUserManagementServer())

	logger.Info("gRPC server listening", "address", cfg.GrpcServer.ServerHost())

	return server.Serve(lis)
}

func main() {
	if err := loadServerConfig(assets.ConfigFileName); err != nil {
		panic(err)
	}

	cfg := config.GetConfig()

	// Initialize logger
	logger := log.NewDefaultLogger()
	logger.Info(
		"Starting gRPC server",
		"grpcHost", cfg.GrpcServer.Host,
		"grpcPort", cfg.GrpcServer.Port,
	)

	// Start gRPC server
	if err := startServer(*cfg); err != nil {
		logger.Error("Server exited with error", "error", err)
	}
}
