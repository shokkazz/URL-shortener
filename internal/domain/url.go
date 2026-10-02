package domain

import "time"

type ShortURL struct {
	URL          string     `json:"url"`
	ShortenedURL string     `json:"shortened_url"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	MaxClicks    *int       `json:"max_clicks,omitempty"`
	Clicks       int        `json:"clicks"`
}
