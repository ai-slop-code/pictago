package user_management

import (
	"context"

	pb "github.com/ai-slop-code/pictago/internal/proto/v1/grpc/gateway"
)

type server struct {
	pb.UnimplementedUserManagementServer
}

func (s *server) GetUsers(context.Context, *pb.GetUsersRequest) (*pb.GetUsersResponse, error) {
	res := pb.GetUsersResponse{}
	return &res, nil
}

func (s *server) CreateUser(context.Context, *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	res := pb.CreateUserResponse{}
	return &res, nil
}

func (s *server) DeleteUser(context.Context, *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	res := pb.DeleteUserResponse{}
	return &res, nil
}

var _ pb.UserManagementServer = &server{}

func NewUserManagementServer() pb.UserManagementServer {
	return &server{}
}
