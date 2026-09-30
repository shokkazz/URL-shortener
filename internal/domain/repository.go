package domain

import "context"

type URLRepository interface {
	CreateShortLink(ctx context.Context, url ShortURL) error
	DeleteShortLink(ctx context.Context, url ShortURL) error
	GetShortLink(ctx context.Context, url ShortURL) (ShortURL, error)
	GetByShortURL(ctx context.Context, short string) (ShortURL, error)
}
