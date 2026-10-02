package domain

import "context"

type URLRepository interface {
	CreateShortLink(ctx context.Context, url ShortURL) error
	DeleteShortLink(ctx context.Context, url ShortURL) error
	GetShortLink(ctx context.Context, url ShortURL) (ShortURL, error)
	GetByShortURL(ctx context.Context, short string) (ShortURL, error)
	RegisterClick(ctx context.Context, shortenedURL string) (ShortURL, error)
	DeleteInactive(ctx context.Context) (int64, error)
}
