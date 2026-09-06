package main

import (
    "os"
    "context"
    "log"
    "net"
    "sync"
    "fmt"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
    "google.golang.org/grpc/reflection"
    pb "github.com/ravirajsubramanian/metering/common"
)

type album struct {
    ID     string  `json:"id"`
    Title  string  `json:"title"`
    Artist string  `json:"artist"`
    Price  float64 `json:"price"`
}

var albums = []album{
    {ID: "1", Title: "Blue Train", Artist: "John Coltrane", Price: 56.99},
    {ID: "2", Title: "Jeru", Artist: "Gerry Mulligan", Price: 17.99},
    {ID: "3", Title: "Sarah Vaughan and Clifford Brown", Artist: "Sarah Vaughan", Price: 39.99},
}

func getAlbum(id string) (album, bool) {
    var match album
    for i := 0; i < len(albums); i++ {
        fmt.Println(albums[i])
        if (albums[i].ID == id){
            match = albums[i]
        }
    }
    return match, true
}

type server struct {
	pb.UnimplementedAlbumServiceServer
	mu    sync.RWMutex
	albums map[string]*pb.Album
}

func (s *server) Create(ctx context.Context, req *pb.CreateRequest) (*pb.CreateResponse, error){
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
    return &pb.CreateResponse{Album album}, nil
}

func (s *server) Read(ctx context.Context, req *pb.ReadRequest) (*pb.ReadResponse, error) {
// 	s.mu.RLock()
// 	defer s.mu.RUnlock()

	pickedAlbum, exists := getAlbum(req.GetId())
	if !exists {
		return nil, status.Errorf(codes.NotFound, "Album with ID %s not found", req.GetId())
	}

	return &pb.ReadResponse{
        Album: &pb.Album{
            Id:     pickedAlbum.ID,
            Title:  pickedAlbum.Title,
            Artist: pickedAlbum.Artist,
            Price:  pickedAlbum.Price,
        },
    }, nil
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
