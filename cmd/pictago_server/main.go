package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ai-slop-code/pictago/assets"
	"github.com/ai-slop-code/pictago/internal/config"
	"github.com/ai-slop-code/pictago/internal/log"
	v1gw "github.com/ai-slop-code/pictago/internal/proto/v1/grpc/gateway"
	"github.com/ai-slop-code/pictago/internal/user"
	"github.com/ai-slop-code/pictago/pkg/authentication"
	"github.com/ai-slop-code/pictago/pkg/file"
	"github.com/ai-slop-code/pictago/pkg/user_configuration"
	"github.com/ai-slop-code/pictago/pkg/user_info"
	"github.com/ai-slop-code/pictago/pkg/user_management"
	"github.com/jmoiron/sqlx"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type (
	GatewayRegistrar func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error
	GrpcRegistrar    func(srv *grpc.Server)
)

type Service struct {
	Name             string
	GrpcRegistrar    GrpcRegistrar
	GatewayRegistrar GatewayRegistrar
}

var services = []Service{
	{
		Name: "Authentication",
		GrpcRegistrar: func(srv *grpc.Server) {
			v1gw.RegisterAuthenticationServer(srv, authentication.NewAuthenticationServer())
		},
		GatewayRegistrar: v1gw.RegisterAuthenticationHandlerFromEndpoint,
	},
	{
		Name: "FileUpload",
		GrpcRegistrar: func(srv *grpc.Server) {
			v1gw.RegisterFileUploadServer(srv, file.NewFileServer())
		},
		GatewayRegistrar: v1gw.RegisterFileUploadHandlerFromEndpoint,
	},
	{
		Name: "UserConfiguration",
		GrpcRegistrar: func(srv *grpc.Server) {
			v1gw.RegisterUserConfigurationServer(srv, user_configuration.NewUserConfigurationServer())
		},
		GatewayRegistrar: v1gw.RegisterUserConfigurationHandlerFromEndpoint,
	},
	{
		Name: "UserInfo",
		GrpcRegistrar: func(srv *grpc.Server) {
			v1gw.RegisterUserInfoServer(srv, user_info.NewUserInfoServer())
		},
		GatewayRegistrar: v1gw.RegisterUserInfoHandlerFromEndpoint,
	},
	{
		Name: "UserManagement",
		GrpcRegistrar: func(srv *grpc.Server) {
			v1gw.RegisterUserManagementServer(srv, user_management.NewUserManagementServer(user.NewService(user.NewDAO(&sqlx.DB{}))))
		},
		GatewayRegistrar: v1gw.RegisterUserManagementHandlerFromEndpoint,
	},
}

func loadServerConfig(configFileName string) error {
	cfgBytes, err := assets.CfgFs.ReadFile(configFileName)
	if err != nil {
		return err
	}
	return config.LoadConfiguration(cfgBytes)
}

func registerAllGrpcServers(srv *grpc.Server) {
	for _, s := range services {
		s.GrpcRegistrar(srv)
	}
}

func registerAllGatewayHandlers(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	for _, s := range services {
		if err := s.GatewayRegistrar(ctx, mux, endpoint, opts); err != nil {
			return fmt.Errorf("failed to register gateway for %s: %w", s.Name, err)
		}
	}
	return nil
}

func startServers(cfg config.Config) error {
	logger := log.NewDefaultLogger()
	grpcAddr := fmt.Sprintf(":%d", cfg.GrpcServer.Port)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", grpcAddr, err)
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logging.UnaryServerInterceptor(&logger)),
	)
	registerAllGrpcServers(grpcServer)

	go func() {
		logger.Info("gRPC server listening", "address", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Error("gRPC server failed", "error", err)
		}
	}()

	ctx := context.Background()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	if err := registerAllGatewayHandlers(ctx, mux, cfg.GrpcServer.ServerHost(), opts); err != nil {
		return err
	}

	httpAddr := fmt.Sprintf(":%d", cfg.HttpServer.Port)
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("REST gateway listening", "address", httpAddr)
		if err := httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server failed", "error", err)
		}
	}()

	<-stop
	logger.Info("Shutting down servers")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown failed", "error", err)
	}

	grpcServer.GracefulStop()
	logger.Info("Servers stopped")
	return nil
}

func run(configFileName string, logger log.Logger) error {
	if err := loadServerConfig(configFileName); err != nil {
		return err
	}

	cfg := config.GetConfig()
	logger.Info("Starting servers", "grpcPort", cfg.GrpcServer.Port, "httpPort", cfg.HttpServer.Port)
	return startServers(*cfg)
}

func main() {
	logger := log.NewDefaultLogger()
	if err := run(assets.ConfigFileName, logger); err != nil {
		logger.Error("Failed to start servers", "error", err)
		panic(err)
	}
}
