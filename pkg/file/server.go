package file

import (
	"context"

	pb "github.com/ai-slop-code/pictago/internal/proto/v1/grpc/gateway"
)

type server struct {
	pb.UnimplementedFileUploadServer
}

func (s *server) UploadFile(context.Context, *pb.UploadFileRequest) (*pb.UploadFileResponse, error) {
	res := pb.UploadFileResponse{}
	return &res, nil
}

func (s *server) DeleteFile(context.Context, *pb.DeleteFileRequest) (*pb.DeleteFileResponse, error) {
	res := pb.DeleteFileResponse{}
	return &res, nil
}

func (s *server) GetFile(context.Context, *pb.GetFileRequest) (*pb.GetFileResponse, error) {
	res := pb.GetFileResponse{}
	return &res, nil
}

func (s *server) ListFiles(context.Context, *pb.ListFilesRequest) (*pb.ListFilesResponse, error) {
	res := pb.ListFilesResponse{}
	return &res, nil
}

var _ pb.FileUploadServer = &server{}

func NewFileServer() pb.FileUploadServer {
	return &server{}
}
