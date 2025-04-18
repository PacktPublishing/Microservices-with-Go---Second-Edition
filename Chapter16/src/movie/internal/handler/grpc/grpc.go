package grpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"movieexample.com/gen"
	"movieexample.com/metadata/pkg/model"
	"movieexample.com/movie/internal/controller/movie"
)

// Handler defines a movie gRPC handler.
type Handler struct {
	gen.UnimplementedMovieServiceServer
	ctrl *movie.Controller
}

// New creates a new movie gRPC handler.
func New(ctrl *movie.Controller) *Handler {
	return &Handler{ctrl: ctrl}
}

// GetMovieDetails returns moviie details by id.
func (h *Handler) GetMovieDetails(ctx context.Context, req *gen.GetMovieDetailsRequest) (*gen.GetMovieDetailsResponse, error) {
	if req == nil || req.MovieId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "nil req or empty id")
	}
	m, err := h.ctrl.Get(ctx, req.MovieId)
	if err != nil && errors.Is(err, movie.ErrNotFound) {
		return nil, status.Errorf(codes.NotFound, err.Error())
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	var rating float64
	if m.Rating != nil {
		rating = *m.Rating
	}
	return &gen.GetMovieDetailsResponse{
		MovieDetails: &gen.MovieDetails{
			Metadata: model.MetadataToProto(&m.Metadata),
			Rating:   rating,
		},
	}, nil
}

// UploadFile handles streaming file uploads.
func (h *Handler) UploadFile(stream gen.MovieService_UploadFileServer) error {
	var filename string
	var file *os.File
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&gen.UploadResponse{Message: fmt.Sprintf("File %s uploaded successfully", filename)})
		}
		if err != nil {
			return err
		}

		if file == nil {
			file, err = os.Create(filename)
			if err != nil {
				return err
			}
			defer file.Close()
		}
		if _, err := file.Write(req.Chunk); err != nil {
			return err
		}
	}
}
