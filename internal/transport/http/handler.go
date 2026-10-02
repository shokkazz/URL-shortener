package http

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"shortener-service/internal/domain"
	"strings"
)

type URLService interface {
	GetByShortenedURL(ctx context.Context, shortenedURL string) (domain.ShortURL, error)
	DeleteShortLink(ctx context.Context, originalURL string) error
	GetShortLink(ctx context.Context, originalURL string) (domain.ShortURL, error)
	CreateShortLink(ctx context.Context, originalURL string, expiresIn *int, maxClicks *int) (domain.ShortURL, error)
	RegisterClick(ctx context.Context, shortenedURL string) (domain.ShortURL, error)
}
type Handler struct {
	us        URLService
	errorPage *template.Template
}

func NewHandler(us URLService) *Handler {
	tmpl := template.Must(
		template.ParseFiles("web/errors.html"),
	)
	return &Handler{
		errorPage: tmpl,
		us:        us,
	}
}

func (h *Handler) Redirect(
	w http.ResponseWriter,
	r *http.Request,
) {
	shortenedURL := r.PathValue("shortenedURL")
	if isPrefetch(r) {
		link, err := h.us.GetByShortenedURL(r.Context(), shortenedURL)
		if err != nil {
			h.renderErrorPage(w, http.StatusNotFound, "Link not found",
				"This short link does not exist.")
			return
		}
		http.Redirect(w, r, link.URL, http.StatusFound)
		return
	}
	link, err := h.us.RegisterClick(
		r.Context(),
		shortenedURL,
	)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrShortLinkExpired):
			h.renderErrorPage(
				w,
				http.StatusGone,
				"Link expired",
				"This short link has expired.",
			)

		case errors.Is(err, domain.ErrClickLimitReached):
			h.renderErrorPage(
				w,
				http.StatusGone,
				"Click limit reached",
				"This short link has reached its maximum number of clicks.",
			)

		case errors.Is(err, domain.ErrShortLinkNotFound):
			h.renderErrorPage(
				w,
				http.StatusNotFound,
				"Link not found",
				"This short link does not exist.",
			)

		default:
			h.renderErrorPage(
				w,
				http.StatusInternalServerError,
				"Internal server error",
				"Something went wrong while processing the short link.",
			)
		}

		return
	}

	http.Redirect(
		w,
		r,
		link.URL,
		http.StatusFound,
	)
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

func (h *Handler) CreateShortLink(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req createShortLinkRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	result, err := h.us.CreateShortLink(
		r.Context(),
		req.URL,
		req.ExpiresIn,
		req.MaxClicks,
	)

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidURL):
			h.respondError(
				w,
				http.StatusBadRequest,
				"invalid url",
			)

		case errors.Is(err, domain.ErrInvalidTTL):
			h.respondError(
				w,
				http.StatusBadRequest,
				"expires_in must be greater than 0",
			)

		case errors.Is(err, domain.ErrInvalidMaxClicks):
			h.respondError(
				w,
				http.StatusBadRequest,
				"max_clicks must be greater than 0",
			)

		default:
			h.respondError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
		}

		return
	}

	h.respondJSON(
		w,
		http.StatusCreated,
		result,
	)
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

func (h *Handler) renderErrorPage(
	w http.ResponseWriter,
	status int,
	title string,
	message string,
) {
	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	w.WriteHeader(status)

	err := h.errorPage.Execute(
		w,
		struct {
			Status  int
			Title   string
			Message string
		}{
			Status:  status,
			Title:   title,
			Message: message,
		},
	)

	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
	}
}

func isPrefetch(r *http.Request) bool {
	if r.Header.Get("Purpose") == "prefetch" {
		return true
	}
	return strings.HasPrefix(r.Header.Get("Sec-Purpose"), "prefetch")
}
