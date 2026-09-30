package service

import (
	"context"
	"crypto/rand"
	"errors"

	"shortener-service/internal/domain"
)

type ShortenerService struct {
	ur domain.URLRepository

	shortURLLength int
	charset        string
}

func NewShortenerService(ur domain.URLRepository, shortURLLength int, charset string) *ShortenerService {
	return &ShortenerService{
		ur:             ur,
		shortURLLength: shortURLLength,
		charset:        charset,
	}
}

func (s *ShortenerService) CreateShortLink(ctx context.Context, originalURL string) (domain.ShortURL, error) {
	existing, err := s.ur.GetShortLink(
		ctx,
		domain.ShortURL{
			URL: originalURL,
		},
	)

	if err == nil {
		return existing, nil
	}

	if !errors.Is(err, domain.ErrShortLinkNotFound) {
		return domain.ShortURL{}, err
	}

	for {
		shortenedURL, err := s.generateShortURL()
		if err != nil {
			return domain.ShortURL{}, err
		}

		shortURL := domain.ShortURL{
			URL:          originalURL,
			ShortenedURL: shortenedURL,
		}

		err = s.ur.CreateShortLink(ctx, shortURL)
		if err == nil {
			return shortURL, nil
		}

		if errors.Is(err, domain.ErrShortLinkExists) {
			continue
		}

		return domain.ShortURL{}, err
	}
}

func (s *ShortenerService) GetShortLink(ctx context.Context, originalURL string) (domain.ShortURL, error) {
	return s.ur.GetShortLink(
		ctx,
		domain.ShortURL{
			URL: originalURL,
		},
	)
}

func (s *ShortenerService) DeleteShortLink(ctx context.Context, originalURL string) error {
	return s.ur.DeleteShortLink(
		ctx,
		domain.ShortURL{
			URL: originalURL,
		},
	)
}

func (s *ShortenerService) GetByShortenedURL(ctx context.Context, shortenedURL string) (domain.ShortURL, error) {
	return s.ur.GetByShortURL(ctx, shortenedURL)
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
