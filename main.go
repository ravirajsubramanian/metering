package main

import (
	"fmt"
	"log"
	"net"
	"os"

	pb "github.com/ravirajsubramanian/metering/common"
	"github.com/ravirajsubramanian/metering/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func addr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	return ":50051"
}

func main() {
	lis, err := net.Listen("tcp", addr())
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterAlbumServiceServer(s, service.NewServer())

	if os.Getenv("ENV") != "production" {
		reflection.Register(s)
	}

	fmt.Printf("gRPC Server with reflection running on port %s...\n", addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
