package postgres

import (
	"context"
	"errors"
	"shortener-service/internal/observability"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"shortener-service/internal/domain"
)

type URLRepository struct {
	db      *pgxpool.Pool
	metrics *observability.Metrics
}

func NewURLRepository(
	db *pgxpool.Pool,
	metrics *observability.Metrics,
) *URLRepository {
	return &URLRepository{
		db:      db,
		metrics: metrics,
	}
}

func (r *URLRepository) DeleteInactive(ctx context.Context) (int64, error) {
	const query = `
		DELETE FROM short_urls
		WHERE (expires_at IS NOT NULL AND expires_at <= NOW())
		   OR (max_clicks IS NOT NULL AND clicks >= max_clicks)
	`

	start := time.Now()
	cmd, err := r.db.Exec(ctx, query)
	r.observeQuery("delete_inactive", start, err)

	if err != nil {
		return 0, err
	}
	return cmd.RowsAffected(), nil
}

func (r *URLRepository) CreateShortLink(
	ctx context.Context,
	url domain.ShortURL,
) error {
	const query = `
    INSERT INTO short_urls (
        url,
        shortened_url,
        expires_at,
        max_clicks,
        clicks
    )
    VALUES ($1, $2, $3, $4, $5)
`

	start := time.Now()
	_, err := r.db.Exec(
		ctx,
		query,
		url.URL,
		url.ShortenedURL,
		url.ExpiresAt,
		url.MaxClicks,
		url.Clicks,
	)
	r.observeQuery("create_short_link", start, err)

	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return err
	}

	if pgErr.Code != "23505" {
		return err
	}

	switch pgErr.ConstraintName {
	case "short_urls_pkey":
		return domain.ErrShortLinkExists

	case "short_urls_shortened_url_key":
		return domain.ErrShortURLCollision

	default:
		return err
	}
}

func (r *URLRepository) DeleteShortLink(ctx context.Context, url domain.ShortURL) error {
	const query = `
		DELETE FROM short_urls
		WHERE url = $1
	`

	start := time.Now()
	cmd, err := r.db.Exec(ctx, query, url.URL)
	r.observeQuery("delete_short_link", start, err)

	if err != nil {
		return err
	}

	if cmd.RowsAffected() == 0 {
		return domain.ErrShortLinkNotFound
	}

	return nil
}

func (r *URLRepository) GetShortLink(
	ctx context.Context,
	url domain.ShortURL,
) (domain.ShortURL, error) {
	const query = `
		SELECT
			url,
			shortened_url,
			expires_at,
			max_clicks,
			clicks
		FROM short_urls
		WHERE url = $1
	`

	var result domain.ShortURL

	start := time.Now()
	err := r.db.QueryRow(ctx, query, url.URL).Scan(
		&result.URL,
		&result.ShortenedURL,
		&result.ExpiresAt,
		&result.MaxClicks,
		&result.Clicks,
	)
	r.observeQuery("get_short_link", start, err)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ShortURL{}, domain.ErrShortLinkNotFound
	}
	if err != nil {
		return domain.ShortURL{}, err
	}

	return result, nil
}
func (r *URLRepository) GetByShortURL(
	ctx context.Context,
	short string,
) (domain.ShortURL, error) {
	const query = `
		SELECT
			url,
			shortened_url,
			expires_at,
			max_clicks,
			clicks
		FROM short_urls
		WHERE shortened_url = $1
	`

	var result domain.ShortURL

	start := time.Now()
	err := r.db.QueryRow(ctx, query, short).Scan(
		&result.URL,
		&result.ShortenedURL,
		&result.ExpiresAt,
		&result.MaxClicks,
		&result.Clicks,
	)
	r.observeQuery("get_by_short_url", start, err)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ShortURL{}, domain.ErrShortLinkNotFound
	}
	if err != nil {
		return domain.ShortURL{}, err
	}

	return result, nil
}
func (r *URLRepository) RegisterClick(
	ctx context.Context,
	shortenedURL string,
) (domain.ShortURL, error) {
	const updateQuery = `
		UPDATE short_urls
		SET clicks = clicks + 1
		WHERE shortened_url = $1
		  AND (expires_at IS NULL OR expires_at > NOW())
		  AND (max_clicks IS NULL OR clicks < max_clicks)
		RETURNING
			url, shortened_url, expires_at, max_clicks, clicks
	`

	var result domain.ShortURL

	start := time.Now()
	err := r.db.QueryRow(ctx, updateQuery, shortenedURL).Scan(
		&result.URL,
		&result.ShortenedURL,
		&result.ExpiresAt,
		&result.MaxClicks,
		&result.Clicks,
	)
	r.observeQuery("register_click", start, err)

	if err == nil {
		return result, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.ShortURL{}, err
	}

	const checkQuery = `
		SELECT
			(expires_at IS NOT NULL AND expires_at <= NOW()) AS is_expired,
			(max_clicks IS NOT NULL AND clicks >= max_clicks) AS is_limit_reached
		FROM short_urls
		WHERE shortened_url = $1
	`

	var (
		isExpired      bool
		isLimitReached bool
	)

	start = time.Now()
	err = r.db.QueryRow(ctx, checkQuery, shortenedURL).
		Scan(&isExpired, &isLimitReached)
	r.observeQuery("check_click_status", start, err)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ShortURL{}, domain.ErrShortLinkNotFound
	}
	if err != nil {
		return domain.ShortURL{}, err
	}

	switch {
	case isExpired:
		return domain.ShortURL{}, domain.ErrShortLinkExpired
	case isLimitReached:
		return domain.ShortURL{}, domain.ErrClickLimitReached
	default:
		return domain.ShortURL{}, domain.ErrShortLinkNotFound
	}
}
func (r *URLRepository) observeQuery(
	operation string,
	start time.Time,
	err error,
) {
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}

	r.metrics.RecordDBQuery(
		operation,
		time.Since(start),
		err,
		250*time.Millisecond,
	)
}
