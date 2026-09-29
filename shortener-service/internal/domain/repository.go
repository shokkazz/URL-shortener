package domain

import "context"

type URLRepository interface {
	CreateShortLink(ctx context.Context, url ShortURL) error
	DeleteShortLink(ctx context.Context, url ShortURL) error
	UpdateShortLink(ctx context.Context, url ShortURL) error
	GetShortLink(ctx context.Context, url ShortURL) error
}
