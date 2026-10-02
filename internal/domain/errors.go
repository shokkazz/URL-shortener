package domain

import "errors"

var (
	ErrShortLinkNotFound = errors.New("short link not found")
	ErrShortLinkExists   = errors.New("short link already exists")
	ErrShortURLCollision = errors.New("short url collision")
	ErrShortLinkExpired  = errors.New("short link expired")
	ErrClickLimitReached = errors.New("click limit reached")
	ErrInvalidURL        = errors.New("invalid url")
	ErrInvalidTTL        = errors.New("invalid ttl")
	ErrInvalidMaxClicks  = errors.New("invalid max clicks")
)
