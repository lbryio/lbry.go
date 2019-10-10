package blobex

import (
	"fmt"
	"net"

	"github.com/lbryio/lbry.go/v2/extras/errors"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Server struct {
	pricePerKB uint64
}

func ListenAndServe(port int) (*grpc.Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, errors.Prefix("failed to listen", err)
	}
	grpcServer := grpc.NewServer()
	RegisterBlobExchangeServer(grpcServer, &Server{})
	// determine whether to use TLS
	err = grpcServer.Serve(listener)
	return grpcServer, err
}

func (s *Server) PriceCheck(ctx context.Context, r *PriceCheckRequest) (*PriceCheckResponse, error) {
	return &PriceCheckResponse{
		DeweysPerKB: s.pricePerKB,
	}, nil
}

func (s *Server) DownloadCheck(context.Context, *HashesRequest) (*HashesResponse, error) {
	return nil, nil
}

func (s *Server) Download(BlobExchange_DownloadServer) error {
	return nil
}

func (s *Server) UploadCheck(context.Context, *HashesRequest) (*HashesResponse, error) {
	return nil, nil
}

func (s *Server) Upload(BlobExchange_UploadServer) error {
	return nil
}
