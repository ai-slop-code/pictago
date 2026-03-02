package user_info

import (
	"context"

	pb "github.com/ai-slop-code/pictago/internal/proto/v1/grpc/gateway"
)

type server struct {
	pb.UnimplementedUserInfoServer
}

func (s *server) GetUserInfo(context.Context, *pb.GetUserInfoRequest) (*pb.GetUserInfoResponse, error) {
	res := pb.GetUserInfoResponse{}
	return &res, nil
}

func NewUserInfoServer() pb.UserInfoServer {
	return &server{}
}
