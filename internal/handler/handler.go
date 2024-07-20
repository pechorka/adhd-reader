package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pechorka/adhd-reader/internal/handler/internal/request"
	"github.com/pechorka/adhd-reader/internal/handler/internal/respond"
	"github.com/pechorka/adhd-reader/internal/handler/mw/auth"
	"github.com/pechorka/adhd-reader/internal/service"
	"github.com/pechorka/adhd-reader/internal/storage"
)

type Service interface {
	FullTexts(userID int64, after *time.Time) ([]storage.TextWithChunks, error)
	SyncTexts(userID int64, texts []service.SyncText) ([]service.SyncText, error)
	NextChunk(userID int64) (storage.Text, string, service.ChunkType, error)
	PrevChunk(userID int64) (storage.Text, string, service.ChunkType, error)
}

type Handlers struct {
	svc Service
}

func NewHandlers(svc Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) Register(mx chi.Router) {
	mx.Get("/text", h.GetTexts)
	mx.Post("/text/sync", h.SyncTexts)
	mx.Post("/text/chunk/next", h.NextChunk)
	mx.Post("/text/chunk/prev", h.PrevChunk)
}

type GetTextsResponse struct {
	Texts []GetTextsResponseItem `json:"texts"`
}

type GetTextsResponseItem struct {
	TextUUID     string   `json:"id"`
	Name         string   `json:"name"`
	CurrentChunk int64    `json:"currentChunk"`
	Chunks       []string `json:"chunks"`
}

func (h *Handlers) GetTexts(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	var after *time.Time
	if afterQ := r.URL.Query().Get("after"); afterQ != "" {
		afterT, err := time.Parse(time.RFC3339, afterQ)
		if err != nil {
			respond.ErrorWithCode(w, http.StatusBadRequest, respond.CODE_INVALID_DATE_FORMAT)
			return
		}
		after = &afterT
	}
	texts, err := h.svc.FullTexts(userID, after)
	if err != nil {
		respond.ErrorWithCode(w, http.StatusInternalServerError, respond.CODE_INTERNAL_ERROR)
		return
	}
	resp := GetTextsResponse{Texts: make([]GetTextsResponseItem, 0, len(texts))}
	for _, text := range texts {
		resp.Texts = append(resp.Texts, GetTextsResponseItem{
			TextUUID:     text.UUID,
			Name:         text.Name,
			CurrentChunk: text.CurrentChunk,
			Chunks:       text.Chunks,
		})
	}
	respond.JSON(w, resp)
}

type SyncTextsRequest struct {
	Items []SyncItem `json:"items"`
}

type SyncItem struct {
	TextUUID     string `json:"id"`
	ModifiedAt   string `json:"modifiedAt"`
	CurrentChunk int64  `json:"currentChunk"`
	Deleted      bool   `json:"deleted"`
}

type SyncTextsResponse struct {
	Items []SyncItem `json:"items"`
}

func (h *Handlers) SyncTexts(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	var req SyncTextsRequest
	err := request.DecodeJSON(r.Body, &req)
	if err != nil {
		respond.ErrorWithCode(w, http.StatusBadRequest, respond.CODE_INVALID_JSON)
		return
	}
	syncTexts := make([]service.SyncText, 0, len(req.Items))
	for _, item := range req.Items {
		modifiedAt, err := time.Parse(time.RFC3339, item.ModifiedAt)
		if err != nil {
			respond.RespondErrorWithText(w, http.StatusBadRequest, respond.CODE_INVALID_DATE_FORMAT, "invalid date format for item: "+item.TextUUID)
			return
		}
		syncTexts = append(syncTexts, service.SyncText{
			TextUUID:     item.TextUUID,
			ModifiedAt:   modifiedAt,
			CurrentChunk: item.CurrentChunk,
			Deleted:      item.Deleted,
		})
	}
	syncOnMobile, err := h.svc.SyncTexts(userID, syncTexts)
	if err != nil {
		respond.ErrorWithCode(w, http.StatusInternalServerError, respond.CODE_INTERNAL_ERROR)
		return
	}
	resp := SyncTextsResponse{Items: make([]SyncItem, 0, len(syncOnMobile))}
	for _, item := range syncOnMobile {
		resp.Items = append(resp.Items, SyncItem{
			TextUUID:     item.TextUUID,
			ModifiedAt:   item.ModifiedAt.Format(time.RFC3339),
			CurrentChunk: item.CurrentChunk,
			Deleted:      item.Deleted,
		})
	}
	respond.JSON(w, resp)
}

type NextChunkRequest struct {
	TextUUID string `json:"id"`
}

type NextChunkResponse struct {
	TextUUID string `json:"id"`
	Chunk    string `json:"chunk"`
	Type     string `json:"type"`
}

func (h *Handlers) NextChunk(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	var req NextChunkRequest
	err := request.DecodeJSON(r.Body, &req)
	if err != nil {
		respond.ErrorWithCode(w, http.StatusBadRequest, respond.CODE_INVALID_JSON)
		return
	}
	text, chunk, chunkType, err := h.svc.NextChunk(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTextFinished):
			respond.ErrorWithCode(w, http.StatusBadRequest, respond.CODE_ALREADY_AT_LAST_CHUNK)
		default:
			respond.ErrorWithCode(w, http.StatusInternalServerError, respond.CODE_INTERNAL_ERROR)
		}
		return
	}
	resp := NextChunkResponse{
		TextUUID: text.UUID,
		Chunk:    chunk,
		Type:     chunkType.String(),
	}
	respond.JSON(w, resp)
}

type PrevChunkRequest struct {
	TextUUID string `json:"id"`
}

type PrevChunkResponse struct {
	TextUUID string `json:"id"`
	Chunk    string `json:"chunk"`
	Type     string `json:"type"`
}

func (h *Handlers) PrevChunk(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r.Context())
	var req PrevChunkRequest
	err := request.DecodeJSON(r.Body, &req)
	if err != nil {
		respond.ErrorWithCode(w, http.StatusBadRequest, respond.CODE_INVALID_JSON)
		return
	}
	text, chunk, chunkType, err := h.svc.PrevChunk(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFirstChunk):
			respond.ErrorWithCode(w, http.StatusBadRequest, respond.CODE_ALREADY_AT_FIRST_CHUNK)
		default:
			respond.ErrorWithCode(w, http.StatusInternalServerError, respond.CODE_INTERNAL_ERROR)
		}
		return
	}
	resp := PrevChunkResponse{
		TextUUID: text.UUID,
		Chunk:    chunk,
		Type:     chunkType.String(),
	}
	respond.JSON(w, resp)
}
