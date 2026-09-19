//go:build integration

// Package integration exercises the AlbumService CRUD API end to end over
// a real gRPC connection (in-memory bufconn transport, real server +
// real generated client). Run with:
//
//	go test -tags integration ./tests/integration/... -v -count=1
package integration

import (
	"context"
	"net"
	"testing"
	"time"

	pb "github.com/ravirajsubramanian/metering/common"
	"github.com/ravirajsubramanian/metering/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const bufSize = 1024 * 1024

// newTestClient spins up a real gRPC server (fresh in-memory store) on a
// bufconn listener and returns a client connected to it.
func newTestClient(t *testing.T) pb.AlbumServiceClient {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	s := grpc.NewServer()
	pb.RegisterAlbumServiceServer(s, service.NewServer())

	go func() {
		// Serve returns only on error/close; ignore ErrServerStopped on cleanup.
		_ = s.Serve(lis)
	}()
	t.Cleanup(func() {
		s.Stop()
		_ = lis.Close()
	})

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return pb.NewAlbumServiceClient(conn)
}

func testCtx(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func mustCreate(t *testing.T, c pb.AlbumServiceClient, title, artist string, price float64) *pb.Album {
	t.Helper()

	ctx, cancel := testCtx(t)
	defer cancel()

	res, err := c.Create(ctx, &pb.CreateRequest{Title: title, Artist: artist, Price: price})
	if err != nil {
		t.Fatalf("Create(%q): %v", title, err)
	}
	if res.GetAlbum().GetId() == "" {
		t.Fatal("Create: expected non-empty album id")
	}
	return res.GetAlbum()
}

func assertCode(t *testing.T, err error, want codes.Code, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error with code %v, got nil", what, want)
	}
	if got := status.Code(err); got != want {
		t.Fatalf("%s: expected code %v, got %v (%v)", what, want, got, err)
	}
}

func TestCreateAlbum(t *testing.T) {
	c := newTestClient(t)

	a := mustCreate(t, c, "Blue Train", "John Coltrane", 56.99)
	if a.GetTitle() != "Blue Train" || a.GetArtist() != "John Coltrane" || a.GetPrice() != 56.99 {
		t.Fatalf("Create: unexpected album fields: %+v", a)
	}

	// Each create must mint a distinct ID.
	b := mustCreate(t, c, "Jeru", "Gerry Mulligan", 17.99)
	if b.GetId() == a.GetId() {
		t.Fatalf("Create: duplicate id %q for two albums", a.GetId())
	}

	// Both albums must be readable back.
	for _, want := range []*pb.Album{a, b} {
		ctx, cancel := testCtx(t)
		res, err := c.Read(ctx, &pb.ReadRequest{Id: want.GetId()})
		cancel()
		if err != nil {
			t.Fatalf("Read(%q): %v", want.GetId(), err)
		}
		if res.GetAlbum().GetTitle() != want.GetTitle() {
			t.Fatalf("Read(%q): got title %q, want %q", want.GetId(), res.GetAlbum().GetTitle(), want.GetTitle())
		}
	}
}

func TestCreateAlbum_Validation(t *testing.T) {
	c := newTestClient(t)

	cases := []struct {
		name   string
		title  string
		artist string
		price  float64
	}{
		{"empty title", "", "Artist", 9.99},
		{"empty artist", "Title", "", 9.99},
		{"negative price", "Title", "Artist", -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := testCtx(t)
			defer cancel()
			_, err := c.Create(ctx, &pb.CreateRequest{Title: tc.title, Artist: tc.artist, Price: tc.price})
			assertCode(t, err, codes.InvalidArgument, "Create validation")
		})
	}
}

func TestReadAlbum(t *testing.T) {
	c := newTestClient(t)

	created := mustCreate(t, c, "Sarah Vaughan and Clifford Brown", "Sarah Vaughan", 39.99)

	ctx, cancel := testCtx(t)
	res, err := c.Read(ctx, &pb.ReadRequest{Id: created.GetId()})
	cancel()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	got := res.GetAlbum()
	if got.GetId() != created.GetId() || got.GetTitle() != created.GetTitle() ||
		got.GetArtist() != created.GetArtist() || got.GetPrice() != created.GetPrice() {
		t.Fatalf("Read: got %+v, want %+v", got, created)
	}

	t.Run("unknown id", func(t *testing.T) {
		ctx, cancel := testCtx(t)
		defer cancel()
		_, err := c.Read(ctx, &pb.ReadRequest{Id: "does-not-exist"})
		assertCode(t, err, codes.NotFound, "Read unknown id")
	})

	t.Run("empty id", func(t *testing.T) {
		ctx, cancel := testCtx(t)
		defer cancel()
		_, err := c.Read(ctx, &pb.ReadRequest{Id: ""})
		assertCode(t, err, codes.InvalidArgument, "Read empty id")
	})
}

func TestUpdateAlbum(t *testing.T) {
	c := newTestClient(t)

	created := mustCreate(t, c, "Old Title", "Old Artist", 10.0)

	t.Run("full update", func(t *testing.T) {
		ctx, cancel := testCtx(t)
		defer cancel()
		res, err := c.Update(ctx, &pb.UpdateRequest{
			Id:     created.GetId(),
			Title:  wrapperspb.String("New Title"),
			Artist: wrapperspb.String("New Artist"),
			Price:  wrapperspb.Double(20.5),
		})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		a := res.GetAlbum()
		if a.GetTitle() != "New Title" || a.GetArtist() != "New Artist" || a.GetPrice() != 20.5 {
			t.Fatalf("Update: unexpected album: %+v", a)
		}
		// Persisted: read back the same values.
		rctx, rcancel := testCtx(t)
		defer rcancel()
		rres, err := c.Read(rctx, &pb.ReadRequest{Id: created.GetId()})
		if err != nil {
			t.Fatalf("Read after update: %v", err)
		}
		if rres.GetAlbum().GetTitle() != "New Title" {
			t.Fatalf("Read after update: got %+v", rres.GetAlbum())
		}
	})

	t.Run("partial update keeps other fields", func(t *testing.T) {
		ctx, cancel := testCtx(t)
		defer cancel()
		res, err := c.Update(ctx, &pb.UpdateRequest{
			Id:    created.GetId(),
			Title: wrapperspb.String("Partial Title"),
		})
		if err != nil {
			t.Fatalf("Update partial: %v", err)
		}
		a := res.GetAlbum()
		if a.GetTitle() != "Partial Title" {
			t.Fatalf("Update partial: title not applied: %+v", a)
		}
		if a.GetArtist() != "New Artist" || a.GetPrice() != 20.5 {
			t.Fatalf("Update partial: untouched fields changed: %+v", a)
		}
	})

	t.Run("unknown id", func(t *testing.T) {
		ctx, cancel := testCtx(t)
		defer cancel()
		_, err := c.Update(ctx, &pb.UpdateRequest{Id: "does-not-exist", Title: wrapperspb.String("x")})
		assertCode(t, err, codes.NotFound, "Update unknown id")
	})

	t.Run("empty id", func(t *testing.T) {
		ctx, cancel := testCtx(t)
		defer cancel()
		_, err := c.Update(ctx, &pb.UpdateRequest{Id: "", Title: wrapperspb.String("x")})
		assertCode(t, err, codes.InvalidArgument, "Update empty id")
	})

	t.Run("empty title value rejected", func(t *testing.T) {
		ctx, cancel := testCtx(t)
		defer cancel()
		_, err := c.Update(ctx, &pb.UpdateRequest{Id: created.GetId(), Title: wrapperspb.String("")})
		assertCode(t, err, codes.InvalidArgument, "Update empty title")
	})

	t.Run("negative price rejected", func(t *testing.T) {
		ctx, cancel := testCtx(t)
		defer cancel()
		_, err := c.Update(ctx, &pb.UpdateRequest{Id: created.GetId(), Price: wrapperspb.Double(-5)})
		assertCode(t, err, codes.InvalidArgument, "Update negative price")
	})
}

func TestDeleteAlbum(t *testing.T) {
	c := newTestClient(t)

	created := mustCreate(t, c, "To Delete", "Some Artist", 1.99)

	ctx, cancel := testCtx(t)
	del, err := c.Delete(ctx, &pb.DeleteRequest{Id: created.GetId()})
	cancel()
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if del.GetId() != created.GetId() {
		t.Fatalf("Delete: got id %q, want %q", del.GetId(), created.GetId())
	}

	// Reading a deleted album must yield NotFound.
	rctx, rcancel := testCtx(t)
	_, err = c.Read(rctx, &pb.ReadRequest{Id: created.GetId()})
	rcancel()
	assertCode(t, err, codes.NotFound, "Read after delete")

	t.Run("unknown id", func(t *testing.T) {
		ctx, cancel := testCtx(t)
		defer cancel()
		_, err := c.Delete(ctx, &pb.DeleteRequest{Id: "does-not-exist"})
		assertCode(t, err, codes.NotFound, "Delete unknown id")
	})

	t.Run("empty id", func(t *testing.T) {
		ctx, cancel := testCtx(t)
		defer cancel()
		_, err := c.Delete(ctx, &pb.DeleteRequest{Id: ""})
		assertCode(t, err, codes.InvalidArgument, "Delete empty id")
	})

	t.Run("double delete", func(t *testing.T) {
		a := mustCreate(t, c, "Temp", "Temp Artist", 2.99)
		ctx, cancel := testCtx(t)
		_, err := c.Delete(ctx, &pb.DeleteRequest{Id: a.GetId()})
		cancel()
		if err != nil {
			t.Fatalf("first Delete: %v", err)
		}
		ctx2, cancel2 := testCtx(t)
		defer cancel2()
		_, err = c.Delete(ctx2, &pb.DeleteRequest{Id: a.GetId()})
		assertCode(t, err, codes.NotFound, "second Delete")
	})
}

// TestCRUDLifecycle runs the full Create -> Read -> Update -> Read ->
// Delete -> Read(NotFound) flow against one album, proving the operations
// compose end to end.
func TestCRUDLifecycle(t *testing.T) {
	c := newTestClient(t)

	// Create
	created := mustCreate(t, c, "Lifecycle", "Lifecycle Artist", 42.0)

	// Read
	ctx, cancel := testCtx(t)
	rres, err := c.Read(ctx, &pb.ReadRequest{Id: created.GetId()})
	cancel()
	if err != nil {
		t.Fatalf("lifecycle Read: %v", err)
	}
	if rres.GetAlbum().GetPrice() != 42.0 {
		t.Fatalf("lifecycle Read: unexpected album: %+v", rres.GetAlbum())
	}

	// Update
	ctx, cancel = testCtx(t)
	ures, err := c.Update(ctx, &pb.UpdateRequest{
		Id:    created.GetId(),
		Price: wrapperspb.Double(43.5),
	})
	cancel()
	if err != nil {
		t.Fatalf("lifecycle Update: %v", err)
	}
	if ures.GetAlbum().GetPrice() != 43.5 || ures.GetAlbum().GetTitle() != "Lifecycle" {
		t.Fatalf("lifecycle Update: unexpected album: %+v", ures.GetAlbum())
	}

	// Read after update
	ctx, cancel = testCtx(t)
	rres, err = c.Read(ctx, &pb.ReadRequest{Id: created.GetId()})
	cancel()
	if err != nil {
		t.Fatalf("lifecycle Read after update: %v", err)
	}
	if rres.GetAlbum().GetPrice() != 43.5 {
		t.Fatalf("lifecycle: price not persisted: %+v", rres.GetAlbum())
	}

	// Delete
	ctx, cancel = testCtx(t)
	dres, err := c.Delete(ctx, &pb.DeleteRequest{Id: created.GetId()})
	cancel()
	if err != nil {
		t.Fatalf("lifecycle Delete: %v", err)
	}
	if dres.GetId() != created.GetId() {
		t.Fatalf("lifecycle Delete: got id %q, want %q", dres.GetId(), created.GetId())
	}

	// Read after delete must be NotFound.
	ctx, cancel = testCtx(t)
	defer cancel()
	_, err = c.Read(ctx, &pb.ReadRequest{Id: created.GetId()})
	assertCode(t, err, codes.NotFound, "lifecycle Read after delete")
}
