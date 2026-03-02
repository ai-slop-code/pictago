package authentication

import (
	"context"

	pb "github.com/ai-slop-code/pictago/internal/proto/v1/grpc/gateway"
)

type server struct {
	pb.UnimplementedAuthenticationServer
}

func (s *server) Login(context.Context, *pb.LoginRequest) (*pb.LoginResponse, error) {
	res := pb.LoginResponse{}
	return &res, nil
}

func (s *server) Logout(context.Context, *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	res := pb.LogoutResponse{}
	return &res, nil
}

func (s *server) ChangePassword(context.Context, *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	res := pb.ChangePasswordResponse{}
	return &res, nil
}

var _ pb.AuthenticationServer = &server{}

func NewAuthenticationServer() pb.AuthenticationServer {
	return &server{}
}
