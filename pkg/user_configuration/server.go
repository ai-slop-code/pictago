package user_configuration

import (
	"context"

	pb "github.com/ai-slop-code/pictago/internal/proto/v1/grpc/gateway"
)

type server struct {
	pb.UnimplementedUserConfigurationServer
}

func (s *server) GetConfig(
	context.Context, *pb.GetConfigRequest,
) (*pb.GetConfigResponse, error) {
	res := pb.GetConfigResponse{}
	return &res, nil
}

func (s *server) UpdateConfig(
	context.Context, *pb.UpdateConfigRequest,
) (*pb.UpdateConfigResponse, error) {
	res := pb.UpdateConfigResponse{}
	return &res, nil
}

func (s *server) GetApiKeys(
	context.Context, *pb.GetApiKeysRequest,
) (*pb.GetApiKeysResponse, error) {
	res := pb.GetApiKeysResponse{}
	return &res, nil
}

func (s *server) CreateApiKey(
	context.Context, *pb.CreateApiKeyRequest,
) (*pb.CreateApiKeyResponse, error) {
	res := pb.CreateApiKeyResponse{}
	return &res, nil
}

func (s *server) DeleteApiKey(
	context.Context, *pb.DeleteApiKeyRequest,
) (*pb.DeleteApiKeyResponse, error) {
	res := pb.DeleteApiKeyResponse{}
	return &res, nil
}

func (s *server) CreateFileCollection(
	context.Context, *pb.CreateFileCollectionRequest,
) (*pb.CreateFileCollectionResponse, error) {
	res := pb.CreateFileCollectionResponse{}
	return &res, nil
}

func (s *server) GetFileCollections(
	context.Context, *pb.GetFileCollectionsRequest,
) (*pb.GetFileCollectionsResponse, error) {
	res := pb.GetFileCollectionsResponse{}
	return &res, nil
}

func (s *server) DeleteFileCollection(
	context.Context, *pb.DeleteFileCollectionRequest,
) (*pb.DeleteFileCollectionResponse, error) {
	res := pb.DeleteFileCollectionResponse{}
	return &res, nil
}

func (s *server) UpdateFileCollection(
	context.Context, *pb.UpdateFileCollectionRequest,
) (*pb.UpdateFileCollectionResponse, error) {
	res := pb.UpdateFileCollectionResponse{}
	return &res, nil
}

func (s *server) GetFileCollection(
	context.Context, *pb.GetFileCollectionRequest,
) (*pb.GetFileCollectionResponse, error) {
	res := pb.GetFileCollectionResponse{}
	return &res, nil
}

var _ pb.UserConfigurationServer = &server{}

func NewUserConfigurationServer() pb.UserConfigurationServer {
	return &server{}
}
