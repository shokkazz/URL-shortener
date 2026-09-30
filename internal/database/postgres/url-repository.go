package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"shortener-service/internal/domain"
)

const uniqueViolation = "23505"

type URLRepository struct {
	db *pgxpool.Pool
}

func NewURLRepository(db *pgxpool.Pool) *URLRepository {
	return &URLRepository{
		db: db,
	}
}

func (r *URLRepository) CreateShortLink(ctx context.Context, url domain.ShortURL) error {
	const query = `
		INSERT INTO short_urls (url, shortened_url)
		VALUES ($1, $2)
	`

	_, err := r.db.Exec(
		ctx,
		query,
		url.URL,
		url.ShortenedURL,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.ErrShortLinkExists
		}

		return err
	}

	return nil
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
		SELECT url, shortened_url
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
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ShortURL{}, domain.ErrShortLinkNotFound
	}

	if err != nil {
		return domain.ShortURL{}, err
	}

	return result, nil
}
func (r *URLRepository) GetByShortURL(ctx context.Context, short string) (domain.ShortURL, error) {
	const query = `
	SELECT url, shortened_url
	FROM short_urls
	WHERE shortened_url = $1
`
	var result domain.ShortURL
	err := r.db.QueryRow(ctx, query, short).Scan(&result.URL, &result.ShortenedURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ShortURL{}, domain.ErrShortLinkNotFound
		}
		return domain.ShortURL{}, mapDBError(err)
	}
	return result, nil
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
