package service

import (
	"context"
	"sync"

	"github.com/google/uuid"
	pb "github.com/ravirajsubramanian/metering/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Server implements pb.AlbumServiceServer with an in-memory store.
// It is safe for concurrent use.
type Server struct {
	pb.UnimplementedAlbumServiceServer
	mu     sync.RWMutex
	albums map[string]*pb.Album
}

// NewServer returns an empty Server ready for use.
func NewServer() *Server {
	return &Server{albums: make(map[string]*pb.Album)}
}

func copyAlbum(a *pb.Album) *pb.Album {
	if a == nil {
		return nil
	}
	return &pb.Album{
		Id:     a.GetId(),
		Title:  a.GetTitle(),
		Artist: a.GetArtist(),
		Price:  a.GetPrice(),
	}
}

func (s *Server) List(_ context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    albumSlice := make([]*pb.Album, 0, len(s.albums))
    for _, album := range s.albums {
        albumSlice = append(albumSlice, album)
    }

    return &pb.ListResponse{Albums: albumSlice}, nil
}

// Create inserts a new album and returns it with a server-generated ID.
func (s *Server) Create(_ context.Context, req *pb.CreateRequest) (*pb.CreateResponse, error) {
	if req.GetTitle() == "" {
		return nil, status.Error(codes.InvalidArgument, "title required")
	}
	if req.GetArtist() == "" {
		return nil, status.Error(codes.InvalidArgument, "artist required")
	}
	if req.GetPrice() < 0 {
		return nil, status.Error(codes.InvalidArgument, "price must not be negative")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	album := &pb.Album{
		Id:     id,
		Title:  req.GetTitle(),
		Artist: req.GetArtist(),
		Price:  req.GetPrice(),
	}
	s.albums[id] = album
	return &pb.CreateResponse{Album: copyAlbum(album)}, nil
}

// Read returns a single album by ID.
func (s *Server) Read(_ context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	a, ok := s.albums[req.GetId()]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Album with ID %s not found", req.GetId())
	}
	return &pb.ReadResponse{Album: copyAlbum(a)}, nil
}

// Update applies a partial update: only non-nil wrapper fields are changed.
func (s *Server) Update(_ context.Context, req *pb.UpdateRequest) (*pb.UpdateResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	a, ok := s.albums[req.GetId()]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "Album with ID %s not found", req.GetId())
	}

	if req.GetTitle() != nil {
		if req.GetTitle().GetValue() == "" {
			return nil, status.Error(codes.InvalidArgument, "title must not be empty")
		}
		a.Title = req.GetTitle().GetValue()
	}
	if req.GetArtist() != nil {
		if req.GetArtist().GetValue() == "" {
			return nil, status.Error(codes.InvalidArgument, "artist must not be empty")
		}
		a.Artist = req.GetArtist().GetValue()
	}
	if req.GetPrice() != nil {
		if req.GetPrice().GetValue() < 0 {
			return nil, status.Error(codes.InvalidArgument, "price must not be negative")
		}
		a.Price = req.GetPrice().GetValue()
	}
	return &pb.UpdateResponse{Album: copyAlbum(a)}, nil
}

// Delete removes an album by ID and returns the deleted ID.
func (s *Server) Delete(_ context.Context, req *pb.DeleteRequest) (*pb.DeleteResponse, error) {
	if req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.albums[req.GetId()]; !ok {
		return nil, status.Errorf(codes.NotFound, "Album with ID %s not found", req.GetId())
	}
	delete(s.albums, req.GetId())
	return &pb.DeleteResponse{Id: req.GetId()}, nil
}
