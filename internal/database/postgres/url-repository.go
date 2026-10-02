package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"shortener-service/internal/domain"
)

type URLRepository struct {
	db *pgxpool.Pool
}

func NewURLRepository(db *pgxpool.Pool) *URLRepository {
	return &URLRepository{
		db: db,
	}
}

func (r *URLRepository) DeleteInactive(ctx context.Context) (int64, error) {
	const query = `
		DELETE FROM short_urls
		WHERE (expires_at IS NOT NULL AND expires_at <= NOW())
		   OR (max_clicks IS NOT NULL AND clicks >= max_clicks)
	`

	cmd, err := r.db.Exec(ctx, query)
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

	_, err := r.db.Exec(
		ctx,
		query,
		url.URL,
		url.ShortenedURL,
		url.ExpiresAt,
		url.MaxClicks,
		url.Clicks,
	)

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

	cmd, err := r.db.Exec(
		ctx,
		query,
		url.URL,
	)

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

	err := r.db.QueryRow(
		ctx,
		query,
		url.URL,
	).Scan(
		&result.URL,
		&result.ShortenedURL,
		&result.ExpiresAt,
		&result.MaxClicks,
		&result.Clicks,
	)

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

	err := r.db.QueryRow(
		ctx,
		query,
		short,
	).Scan(
		&result.URL,
		&result.ShortenedURL,
		&result.ExpiresAt,
		&result.MaxClicks,
		&result.Clicks,
	)

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

	err := r.db.QueryRow(ctx, updateQuery, shortenedURL).Scan(
		&result.URL,
		&result.ShortenedURL,
		&result.ExpiresAt,
		&result.MaxClicks,
		&result.Clicks,
	)

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

	err = r.db.QueryRow(ctx, checkQuery, shortenedURL).
		Scan(&isExpired, &isLimitReached)

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
func mapDBError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	switch pgErr.Code {
	case "23505":
		return domain.ErrShortLinkExists
	case "23503":
		return domain.ErrShortLinkNotFound
	default:
		return err
	}
}
