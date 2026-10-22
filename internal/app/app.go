package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"shortener-service/internal/config"
	"shortener-service/internal/database/postgres"
	"shortener-service/internal/logger"
	"shortener-service/internal/observability"
	"shortener-service/internal/service"
	myhttp "shortener-service/internal/transport/http"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func StartApp() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Println("Error loading config:", err)
		return
	}
	logs, err := logger.New(*cfg)
	if err != nil {
		log.Println("Error creating logger:", err)
		return
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, cfg.Dbc.DSN())

	if err != nil {
		log.Println("Error during connection to database: ", err)
		return
	}

	defer db.Close()
	err = db.Ping(ctx)
	if err != nil {
		log.Println("Error during database initial ping: ", err)
		return
	}
	metrics := observability.NewMetrics()

	metrics.StartDBMetrics(
		ctx,
		db,
		30*time.Second,
	)
	httpLog := logs.With("service.layer", "http")
	serviceLog := logs.With("service.layer", "service")
	ur := postgres.NewURLRepository(db, metrics)
	us := service.NewShortenerService(ur, cfg.ShortURLLength, cfg.Charset, cfg.CleanupSeconds, serviceLog, metrics)
	uh := myhttp.NewHandler(us, httpLog)
	us.StartCleanup(ctx)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{shortenedURL}", uh.Redirect)
	mux.HandleFunc("POST /shorten", uh.CreateShortLink)
	mux.HandleFunc("GET /shorten", uh.GetShortLink)
	mux.HandleFunc("DELETE /shorten", uh.DeleteShortLink)
	mux.Handle("GET /metrics", metrics.Handler())
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	loggingHandler := loggerMiddleware(httpLog, metrics, mux)

	server := http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      loggingHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		IdleTimeout:  30 * time.Second,
	}
	serverErr := make(chan error, 1)
	quit := make(chan os.Signal, 1)

	go func() {
		logs.Info(ctx, "server started on port", cfg.Port)
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(quit)
	select {
	case err := <-serverErr:
		logs.Error(ctx, "Server failed:", err)

	case sig := <-quit:
		logs.Info(ctx, "Received signal:", sig)
	}

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = server.Shutdown(ctxShutdown); err != nil {
		logs.Info(ctx, "Server forced to shutdown: ", err)
	}
	logs.Info(ctx, "Server stopped")
}

func loggerMiddleware(
	logs *logger.Logger,
	metrics *observability.Metrics,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()

		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}

		method := r.Method

		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		rw := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		status := strconv.Itoa(rw.statusCode)

		metrics.HTTPRequestsTotal.WithLabelValues(
			method,
			route,
			status,
		).Inc()

		metrics.HTTPRequestDuration.WithLabelValues(
			method,
			route,
		).Observe(duration.Seconds())

		metrics.HTTPResponsesTotal.WithLabelValues(status).Inc()

		logs.Info(r.Context(), "HTTP request completed",
			"http.method", method,
			"http.route", route,
			"http.path", r.URL.Path,
			"http.status_code", rw.statusCode,
			"http.remote_addr", r.RemoteAddr,
			"http.duration_ms", duration.Milliseconds(),
		)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rw *responseRecorder) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		return
	}

	rw.statusCode = statusCode
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseRecorder) Write(body []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	return rw.ResponseWriter.Write(body)
}
