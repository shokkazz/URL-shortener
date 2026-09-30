package domain

import "errors"

var (
	ErrShortLinkNotFound = errors.New("short link not found")
	ErrShortLinkExists   = errors.New("short link already exists")
)
