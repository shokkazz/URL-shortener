package observability

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	Registry *prometheus.Registry

	HTTPRequestsTotal    *prometheus.CounterVec
	HTTPRequestDuration  *prometheus.HistogramVec
	HTTPRequestsInFlight prometheus.Gauge
	HTTPResponsesTotal   *prometheus.CounterVec

	ShortLinksCreated         prometheus.Counter
	ShortLinkCreateErrors     prometheus.Counter
	ShortLinkNotFound         prometheus.Counter
	ShortLinkRedirectDuration prometheus.Histogram

	DBPoolTotalConns         prometheus.Gauge
	DBPoolAcquiredConns      prometheus.Gauge
	DBPoolIdleConns          prometheus.Gauge
	DBPoolMaxConns           prometheus.Gauge
	DBPoolAcquiredRatio      prometheus.Gauge
	DBPoolAcquireTotal       prometheus.Counter
	DBPoolAcquireWaitSeconds prometheus.Counter

	DBQueryDuration    *prometheus.HistogramVec
	DBSlowQueriesTotal prometheus.Counter
	DBQueryErrorsTotal prometheus.Counter

	DBSizeBytes      prometheus.Gauge
	DBTableSizeBytes *prometheus.GaugeVec

	DBMetricsRefreshErrors prometheus.Counter
	mu                     sync.Mutex
	lastAcquireCount       int64
	lastAcquireWaitSeconds float64
}

func NewMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(
			collectors.ProcessCollectorOpts{},
		),
	)

	m := &Metrics{
		Registry: reg,

		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{"method", "route", "status"},
		),

		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "route"},
		),

		HTTPRequestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently in flight.",
			},
		),

		HTTPResponsesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_responses_total",
				Help: "Total HTTP responses by status code.",
			},
			[]string{"status"},
		),

		ShortLinksCreated: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "short_links_created_total",
				Help: "Total successfully created short links.",
			},
		),

		ShortLinkCreateErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "short_link_create_errors_total",
				Help: "Total errors while creating short links.",
			},
		),

		ShortLinkNotFound: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "short_link_not_found_total",
				Help: "Total redirects for non-existent short links.",
			},
		),

		ShortLinkRedirectDuration: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "short_link_redirect_duration_seconds",
				Help:    "Duration of short link redirect operations.",
				Buckets: prometheus.DefBuckets,
			},
		),

		DBPoolTotalConns: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_pool_total_connections",
				Help: "Total PostgreSQL pool connections.",
			},
		),

		DBPoolAcquiredConns: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_pool_acquired_connections",
				Help: "Currently acquired PostgreSQL pool connections.",
			},
		),

		DBPoolIdleConns: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_pool_idle_connections",
				Help: "Idle PostgreSQL pool connections.",
			},
		),

		DBPoolMaxConns: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_pool_max_connections",
				Help: "Maximum PostgreSQL pool connections.",
			},
		),

		DBPoolAcquiredRatio: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_pool_acquired_connections_ratio",
				Help: "Fraction of PostgreSQL pool connections acquired.",
			},
		),

		DBPoolAcquireTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "db_pool_acquire_total",
				Help: "Total successful acquisitions from PostgreSQL pool.",
			},
		),

		DBPoolAcquireWaitSeconds: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "db_pool_acquire_wait_seconds_total",
				Help: "Cumulative time spent acquiring pool connections.",
			},
		),

		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Duration of instrumented PostgreSQL queries.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),

		DBSlowQueriesTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "db_slow_queries_total",
				Help: "Total instrumented queries exceeding slow threshold.",
			},
		),

		DBQueryErrorsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "db_query_errors_total",
				Help: "Total errors from instrumented database operations.",
			},
		),

		DBSizeBytes: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_size_bytes",
				Help: "Current size of the connected PostgreSQL database.",
			},
		),

		DBTableSizeBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "db_table_size_bytes",
				Help: "Current total on-disk size of PostgreSQL tables and indexes.",
			},
			[]string{"table"},
		),

		DBMetricsRefreshErrors: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "db_metrics_refresh_errors_total",
				Help: "Errors refreshing PostgreSQL statistics.",
			},
		),
	}

	reg.MustRegister(
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPRequestsInFlight,
		m.HTTPResponsesTotal,
		m.ShortLinksCreated,
		m.ShortLinkCreateErrors,
		m.ShortLinkNotFound,
		m.ShortLinkRedirectDuration,
		m.DBPoolTotalConns,
		m.DBPoolAcquiredConns,
		m.DBPoolIdleConns,
		m.DBPoolMaxConns,
		m.DBPoolAcquiredRatio,
		m.DBPoolAcquireTotal,
		m.DBPoolAcquireWaitSeconds,
		m.DBQueryDuration,
		m.DBSlowQueriesTotal,
		m.DBQueryErrorsTotal,
		m.DBSizeBytes,
		m.DBTableSizeBytes,
		m.DBMetricsRefreshErrors,
	)

	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(
		m.Registry,
		promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		},
	)
}

func (m *Metrics) RecordDBQuery(
	operation string,
	duration time.Duration,
	err error,
	slowThreshold time.Duration,
) {
	m.DBQueryDuration.
		WithLabelValues(operation).
		Observe(duration.Seconds())

	if err != nil {
		m.DBQueryErrorsTotal.Inc()
	}

	if slowThreshold > 0 && duration >= slowThreshold {
		m.DBSlowQueriesTotal.Inc()
	}
}

func (m *Metrics) StartDBMetrics(
	ctx context.Context,
	db *pgxpool.Pool,
	interval time.Duration,
) {
	if interval <= 0 {
		interval = 30 * time.Second
	}

	m.refreshDBMetrics(ctx, db)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.refreshDBMetrics(ctx, db)
			}
		}
	}()
}

func (m *Metrics) refreshDBMetrics(
	ctx context.Context,
	db *pgxpool.Pool,
) {
	stats := db.Stat()

	total := float64(stats.TotalConns())
	acquired := float64(stats.AcquiredConns())

	m.DBPoolTotalConns.Set(total)
	m.DBPoolAcquiredConns.Set(acquired)
	m.DBPoolIdleConns.Set(float64(stats.IdleConns()))
	m.DBPoolMaxConns.Set(float64(stats.MaxConns()))

	if stats.MaxConns() > 0 {
		m.DBPoolAcquiredRatio.Set(
			acquired / float64(stats.MaxConns()),
		)
	} else {
		m.DBPoolAcquiredRatio.Set(0)
	}

	currentAcquireCount := stats.AcquireCount()
	currentWaitSeconds := stats.AcquireDuration().Seconds()

	m.mu.Lock()
	if currentAcquireCount >= m.lastAcquireCount {
		m.DBPoolAcquireTotal.Add(
			float64(currentAcquireCount - m.lastAcquireCount),
		)
	}
	if currentWaitSeconds >= m.lastAcquireWaitSeconds {
		m.DBPoolAcquireWaitSeconds.Add(
			currentWaitSeconds - m.lastAcquireWaitSeconds,
		)
	}
	m.lastAcquireCount = currentAcquireCount
	m.lastAcquireWaitSeconds = currentWaitSeconds

	m.mu.Unlock()

	var dbSize int64
	err := db.QueryRow(
		ctx,
		`SELECT pg_database_size(current_database())`,
	).Scan(&dbSize)
	if err != nil {
		m.DBMetricsRefreshErrors.Inc()
		return
	}

	m.DBSizeBytes.Set(float64(dbSize))

	rows, err := db.Query(ctx, `
		SELECT schemaname || '.' || relname,
		       pg_total_relation_size(relid)
		FROM pg_catalog.pg_statio_user_tables
	`)
	if err != nil {
		m.DBMetricsRefreshErrors.Inc()
		return
	}
	defer rows.Close()

	for rows.Next() {
		var table string
		var size int64

		if err := rows.Scan(&table, &size); err != nil {
			m.DBMetricsRefreshErrors.Inc()
			continue
		}

		m.DBTableSizeBytes.
			WithLabelValues(table).
			Set(float64(size))
	}

	if rows.Err() != nil {
		m.DBMetricsRefreshErrors.Inc()
	}
}
