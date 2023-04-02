package server

import (
	"context"

	"github.com/pechorka/adhd-reader/cmd/api/internal/middleware"
	"github.com/pechorka/adhd-reader/generated/api"
	"github.com/pechorka/adhd-reader/internal/service"
	"github.com/pkg/errors"
)

type Server struct {
	api.UnimplementedAdhdReaderServiceServer
	service *service.Service
}

var _ api.AdhdReaderServiceServer = (*Server)(nil)

func NewServer(s *service.Service) *Server {
	return &Server{service: s}
}

func (s *Server) SetChunkSize(ctx context.Context, req *api.SetChunkSizeRequest) (*api.SetChunkSizeResponse, error) {
	userID := middleware.UserID(ctx)
	err := s.service.SetChunkSize(userID, req.ChunkSize)
	if err != nil {
		return nil, errors.Wrap(err, "could not set chunk size")
	}
	return &api.SetChunkSizeResponse{}, nil
}

func (s *Server) AddText(ctx context.Context, req *api.AddTextRequest) (*api.AddTextResponse, error) {
	userID := middleware.UserID(ctx)
	textUUID, err := s.service.AddText(userID, req.TextName, req.Text)
	if err != nil {
		return nil, errors.Wrap(err, "could not save text")
	}
	return &api.AddTextResponse{
		TextUuid: textUUID,
	}, nil
}

func (s *Server) ListTexts(ctx context.Context, req *api.ListTextsRequest) (*api.ListTextsResponse, error) {
	userID := middleware.UserID(ctx)
	texts, err := s.service.ListTexts(userID)
	if err != nil {
		return nil, errors.Wrap(err, "could not retrive texts")
	}
	return &api.ListTextsResponse{
		Texts: mapTexts(texts),
	}, nil
}

func mapTexts(list []service.TextWithCompletion) []*api.TextWithCompletion {
	result := make([]*api.TextWithCompletion, 0, len(list))
	for _, item := range list {
		result = append(result, &api.TextWithCompletion{
			Uuid:              item.UUID,
			Name:              item.Name,
			CompletionPercent: int32(item.CompletionPercent),
		})
	}
	return result
}

func (s *Server) SelectText(ctx context.Context, req *api.SelectTextRequest) (*api.SelectTextResponse, error) {
	userID := middleware.UserID(ctx)
	text, err := s.service.SelectText(userID, req.TextUuid)
	if err != nil {
		return nil, errors.Wrap(err, "could not select text")
	}
	return &api.SelectTextResponse{
		Text: mapText(text),
	}, nil
}

func mapText(text service.Text) *api.Text {
	return &api.Text{
		Uuid: text.UUID,
		Name: text.Name,
	}
}

func (s *Server) SetPage(ctx context.Context, req *api.SetPageRequest) (*api.SetPageResponse, error) {
	userID := middleware.UserID(ctx)
	err := s.service.SetPage(userID, req.Page)
	if err != nil {
		return nil, errors.Wrap(err, "could not set page")
	}
	return &api.SetPageResponse{}, nil
}

func (s *Server) NextChunk(ctx context.Context, req *api.NextChunkRequest) (*api.NextChunkResponse, error) {
	userID := middleware.UserID(ctx)
	err := s.service.NextChunk(userID)
	if err != nil {
		return nil, errors.Wrap(err, "could not set page")
	}
	return &api.SetPageResponse{}, nil
}

func (s *Server) PrevChunk(ctx context.Context, req *api.PrevChunkRequest) (*api.PrevChunkResponse, error) {
	userID := middleware.UserID(ctx)
	panic("implement me")
}
