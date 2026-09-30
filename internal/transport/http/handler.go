package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"shortener-service/internal/domain"
	"strings"
)

type URLService interface {
	GetByShortenedURL(ctx context.Context, shortenedURL string) (domain.ShortURL, error)
	DeleteShortLink(ctx context.Context, originalURL string) error
	GetShortLink(ctx context.Context, originalURL string) (domain.ShortURL, error)
	CreateShortLink(ctx context.Context, originalURL string) (domain.ShortURL, error)
}
type Handler struct {
	us URLService
}

func NewHandler(us URLService) *Handler {
	return &Handler{
		us: us,
	}
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortenedURL := r.PathValue("shortenedURL")

	if shortenedURL == "" {
		h.respondError(w, http.StatusBadRequest, "shortened url is required")
		return
	}

	result, err := h.us.GetByShortenedURL(r.Context(), shortenedURL)
	if err != nil {
		if errors.Is(err, domain.ErrShortLinkNotFound) {
			h.respondError(w, http.StatusNotFound, "short link not found")
			return
		}

		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	http.Redirect(w, r, result.URL, http.StatusFound)
}

func (h *Handler) DeleteShortLink(w http.ResponseWriter, r *http.Request) {
	originalURL := strings.TrimSpace(r.URL.Query().Get("url"))

	if originalURL == "" {
		h.respondError(w, http.StatusBadRequest, "url is required")
		return
	}

	err := h.us.DeleteShortLink(r.Context(), originalURL)
	if err != nil {
		if errors.Is(err, domain.ErrShortLinkNotFound) {
			h.respondError(w, http.StatusNotFound, "short link not found")
			return
		}

		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetShortLink(w http.ResponseWriter, r *http.Request) {
	originalURL := strings.TrimSpace(r.URL.Query().Get("url"))

	if originalURL == "" {
		h.respondError(w, http.StatusBadRequest, "url is required")
		return
	}

	result, err := h.us.GetShortLink(r.Context(), originalURL)
	if err != nil {
		if errors.Is(err, domain.ErrShortLinkNotFound) {
			h.respondError(w, http.StatusNotFound, "short link not found")
			return
		}

		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.respondJSON(w, http.StatusOK, result)
}

func (h *Handler) CreateShortLink(w http.ResponseWriter, r *http.Request) {
	var req createShortLinkRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.URL = strings.TrimSpace(req.URL)

	if req.URL == "" {
		h.respondError(w, http.StatusBadRequest, "url is required")
		return
	}

	result, err := h.us.CreateShortLink(r.Context(), req.URL)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	h.respondJSON(w, http.StatusCreated, result)
}

func (h *Handler) respondError(
	w http.ResponseWriter,
	status int,
	message string,
) {
	h.respondJSON(w, status, map[string]string{
		"error": message,
	})
}

func (h *Handler) respondJSON(
	w http.ResponseWriter,
	status int,
	data any,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
