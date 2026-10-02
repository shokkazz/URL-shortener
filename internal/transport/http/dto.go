package http

type createShortLinkRequest struct {
	URL       string `json:"url"`
	ExpiresIn *int   `json:"expires_in,omitempty"`
	MaxClicks *int   `json:"max_clicks,omitempty"`
}
