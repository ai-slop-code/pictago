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

func registerServers(srv *grpc.Server) {
	v1gw.RegisterAuthenticationServer(srv, authentication.NewAuthenticationServer())
	v1gw.RegisterFileUploadServer(srv, file.NewFileServer())
	v1gw.RegisterUserConfigurationServer(srv, user_configuration.NewUserConfigurationServer())
	v1gw.RegisterUserInfoServer(srv, user_info.NewUserInfoServer())
	v1gw.RegisterUserManagementServer(srv, user_management.NewUserManagementServer())
}

func startServer(cfg config.Config) error {
	logger := log.NewDefaultLogger()

	lis, err := net.Listen("tcp", cfg.GrpcServer.ServerHost())
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.GrpcServer.ServerHost(), err)
	}

	//goland:noinspection GoUnhandledErrorResult
	defer lis.Close()

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logging.UnaryServerInterceptor(&logger)),
	)

	registerServers(server)

	logger.Info("gRPC server listening", "address", cfg.GrpcServer.ServerHost())

	return server.Serve(lis)
}

func run(configFileName string, logger log.Logger) error {
	if err := loadServerConfig(configFileName); err != nil {
		return err
	}

	cfg := config.GetConfig()
	logger.Info(
		"Starting gRPC server",
		"grpcHost", cfg.GrpcServer.Host,
		"grpcPort", cfg.GrpcServer.Port,
	)
	if err := startServer(*cfg); err != nil {
		return err
	}
	return nil
}

func main() {
	logger := log.NewDefaultLogger()
	if err := run(assets.ConfigFileName, logger); err != nil {
		logger.Error("Failed to start gRPC server", "error", err)
		panic(err)
	}
}
