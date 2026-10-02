package service

import (
	"context"
	"crypto/rand"
	"errors"
	"log"
	"net/url"
	"strings"
	"time"

	"shortener-service/internal/domain"
)

type ShortenerService struct {
	ur             domain.URLRepository
	shortURLLength int
	charset        string
	Cleanup        int
}

func NewShortenerService(
	ur domain.URLRepository,
	shortURLLength int,
	charset string,
	cleanup int,
) *ShortenerService {
	return &ShortenerService{
		Cleanup:        cleanup,
		ur:             ur,
		shortURLLength: shortURLLength,
		charset:        charset,
	}
}

func (s *ShortenerService) CreateShortLink(
	ctx context.Context,
	originalURL string,
	expiresIn *int,
	maxClicks *int,
) (domain.ShortURL, error) {

	normalizedURL, err := normalizeURL(originalURL)
	if err != nil {
		return domain.ShortURL{}, err
	}

	if expiresIn != nil && *expiresIn <= 0 {
		return domain.ShortURL{}, domain.ErrInvalidTTL
	}

	if maxClicks != nil && *maxClicks <= 0 {
		return domain.ShortURL{}, domain.ErrInvalidMaxClicks
	}
	var expiresAt *time.Time
	if expiresIn != nil {
		expiration := time.Now().UTC().Add(
			time.Duration(*expiresIn) * time.Second,
		)
		expiresAt = &expiration
	}
	for {
		shortenedURL, err := s.generateShortURL()
		if err != nil {
			return domain.ShortURL{}, err
		}

		shortURL := domain.ShortURL{
			URL:          normalizedURL,
			ShortenedURL: shortenedURL,
			ExpiresAt:    expiresAt,
			MaxClicks:    maxClicks,
			Clicks:       0,
		}

		err = s.ur.CreateShortLink(ctx, shortURL)

		if err == nil {
			return shortURL, nil
		}
		if errors.Is(err, domain.ErrShortLinkExists) {
			existing, getErr := s.ur.GetShortLink(
				ctx,
				domain.ShortURL{URL: normalizedURL},
			)
			if getErr != nil {
				return domain.ShortURL{}, getErr
			}

			if !s.isInactive(existing) {
				return existing, nil
			}

			if delErr := s.ur.DeleteShortLink(ctx, existing); delErr != nil {
				if !errors.Is(delErr, domain.ErrShortLinkNotFound) {
					return domain.ShortURL{}, delErr
				}
			}

			continue
		}
		if errors.Is(err, domain.ErrShortURLCollision) {
			continue
		}

		return domain.ShortURL{}, err
	}
}
func (s *ShortenerService) GetShortLink(
	ctx context.Context,
	originalURL string,
) (domain.ShortURL, error) {

	normalizedURL, err := normalizeURL(originalURL)
	if err != nil {
		return domain.ShortURL{}, err
	}

	return s.ur.GetShortLink(
		ctx,
		domain.ShortURL{URL: normalizedURL},
	)
}

func (s *ShortenerService) DeleteShortLink(
	ctx context.Context,
	originalURL string,
) error {

	normalizedURL, err := normalizeURL(originalURL)
	if err != nil {
		return err
	}

	return s.ur.DeleteShortLink(
		ctx,
		domain.ShortURL{
			URL: normalizedURL,
		},
	)
}

func (s *ShortenerService) GetByShortenedURL(
	ctx context.Context,
	shortenedURL string,
) (domain.ShortURL, error) {
	return s.ur.GetByShortURL(ctx, shortenedURL)
}

func (s *ShortenerService) RegisterClick(
	ctx context.Context,
	shortenedURL string,
) (domain.ShortURL, error) {
	return s.ur.RegisterClick(ctx, shortenedURL)
}

func (s *ShortenerService) generateShortURL() (string, error) {
	result := make([]byte, s.shortURLLength)
	randomBytes := make([]byte, s.shortURLLength)

	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}

	for i := range result {
		result[i] = s.charset[int(randomBytes[i])%len(s.charset)]
	}

	return string(result), nil
}
func normalizeURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)

	if rawURL == "" {
		return "", domain.ErrInvalidURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", domain.ErrInvalidURL
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", domain.ErrInvalidURL
	}

	if parsed.Host == "" {
		return "", domain.ErrInvalidURL
	}

	if parsed.User != nil {
		return "", domain.ErrInvalidURL
	}

	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)

	if parsed.Scheme == "http" && strings.HasSuffix(parsed.Host, ":80") {
		parsed.Host = strings.TrimSuffix(parsed.Host, ":80")
	}

	if parsed.Scheme == "https" && strings.HasSuffix(parsed.Host, ":443") {
		parsed.Host = strings.TrimSuffix(parsed.Host, ":443")
	}

	if parsed.Path == "/" {
		parsed.Path = ""
	}

	return parsed.String(), nil
}
func (s *ShortenerService) isInactive(link domain.ShortURL) bool {
	now := time.Now()

	if link.ExpiresAt != nil && !link.ExpiresAt.After(now) {
		return true
	}
	if link.MaxClicks != nil && link.Clicks >= *link.MaxClicks {
		return true
	}
	return false
}

func (s *ShortenerService) StartCleanup(ctx context.Context) {

	go func() {
		ticker := time.NewTicker(time.Duration(s.Cleanup) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				n, err := s.ur.DeleteInactive(ctx)
				if err != nil {
					log.Println("cleanup error:", err)
					continue
				}
				if n > 0 {
					log.Printf("cleanup: removed %d inactive links", n)
				}
			}
		}
	}()
}
