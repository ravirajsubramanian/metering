package main

import (
    "os"
    "context"
    "log"
    "net"
    "sync"
    "fmt"
    "github.com/google/uuid"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/reflection"
    pb "github.com/ravirajsubramanian/metering/common"
)

func (s *server) getAlbum(id string) (*pb.Album, error) {
    album, exists := s.albums[id]
    if !exists {
        return nil, status.Errorf(codes.NotFound, "Album with ID %s not found", id)
    }
    return &pb.Album{
        Id:     album.Id,
        Title:  album.Title,
        Artist: album.Artist,
        Price:  album.Price,
    }, nil
}

func stringOrElse(primary, fallback string) string {
    if primary != "" {
        return primary
    }
    return fallback
}

func floatOrElse(primary, secondary float64) float64 {
    if primary == nil {
        return primary
    }
    return secondary
}

func (s *server) updateAlbum(id string, title string, artist string, price float64) (*pb.Album, error){
    album, exists := s.albums[id]
    if !exists {
        return nil, status.Errorf(codes.NotFound, "Album with ID %s not found", id)
    }
    s.albums[id] = &pb.Album{
        Id: album.Id,
        Title: stringOrElse(title, album.Title),
        Artist: stringOrElse(artist, album.Artist),
        Price:  floatOrElse(price, album.Price),
    }
    return &pb.Album{
        Id: s.albums[id].Id,
        Title: s.albums[id].Title,
        Artist: s.albums[id].Artist,
        Price: s.albums[id].Price,
    }, nil
}

func (s *server) deleteAlbum(id string) (string, error){
    _, exists := s.albums[id]
    if !exists {
        return "", status.Errorf(codes.NotFound, "Album with ID %s not found", id)
    }
    delete(s.albums, id)
    return id, nil
}

type server struct {
	pb.UnimplementedAlbumServiceServer
	mu    sync.RWMutex
	albums map[string]*pb.Album
}

func (s *server) Create(ctx context.Context, req *pb.CreateRequest) (*pb.CreateResponse, error){
    if req.GetTitle() == "" {
        return nil, status.Error(codes.InvalidArgument, "title required")
    }
    if req.GetArtist() == "" {
        return nil, status.Error(codes.InvalidArgument, "artist required")
    }
    if req.GetPrice() < 0 {
        return nil, status.Error(codes.InvalidArgument, "price required")
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    id := uuid.New().String()
    album := &pb.Album{
    		Id: id,
    		Title: req.GetTitle(),
    		Artist: req.GetArtist(),
    		Price: req.GetPrice(),
    	}
    s.albums[id] = album
    return &pb.CreateResponse{ Album: album }, nil
}

func (s *server) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pickedAlbum, err := s.getAlbum(req.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.ReadResponse{ Album: pickedAlbum }, nil
}

func (s *server) Update(ctx context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    updatedAlbum, err := s.updateAlbum(req.GetId(), req.GetTitle(), req.GetArtist(), req.GetPrice())
    if err != nil {
        return nil, err
    }
    return &pb.UpdateResponse{ Album: updatedAlbum }, nil
}

func (s *server) Delete(ctx context.Context, req *pb.DeleteRequest)(*pb.DeleteResponse, error){
    s.mu.Lock()
    defer s.mu.Unlock()

    deletedId, err := s.deleteAlbum(req.GetId())
    if err != nil {
        return nil, err
    }
    return &pb.DeleteResponse{ Id: deletedId }, nil;
}

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterAlbumServiceServer(s, &server{
		albums: make(map[string]*pb.Album),
	})

    if os.Getenv("ENV") != "production" {
        reflection.Register(s)
    }

	fmt.Println("gRPC Server with reflection running on port :50051...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
