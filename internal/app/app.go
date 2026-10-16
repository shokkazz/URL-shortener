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
	"shortener-service/internal/service"
	myhttp "shortener-service/internal/transport/http"
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
	httpLog := logs.With("service.layer", "http")
	serviceLog := logs.With("service.layer", "service")
	ur := postgres.NewURLRepository(db)
	us := service.NewShortenerService(ur, cfg.ShortURLLength, cfg.Charset, cfg.CleanupSeconds, serviceLog)
	uh := myhttp.NewHandler(us, httpLog)
	us.StartCleanup(ctx)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{shortenedURL}", uh.Redirect)
	mux.HandleFunc("POST /shorten", uh.CreateShortLink)
	mux.HandleFunc("GET /shorten", uh.GetShortLink)
	mux.HandleFunc("DELETE /shorten", uh.DeleteShortLink)
	mux.Handle("/", http.FileServer(http.Dir("./web")))

	loggingHandler := loggerMiddleware(httpLog, mux)

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

func loggerMiddleware(logs *logger.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		rw := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(rw, r)

		logs.Info(r.Context(), "HTTP request completed",
			"http.method", r.Method,
			"http.path", r.URL.Path,
			"http.status_code", rw.statusCode,
			"http.remote_addr", r.RemoteAddr,
			"http.duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseRecorder) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}
func (rw *responseRecorder) Write(body []byte) (int, error) {
	if rw.statusCode == 0 {
		rw.statusCode = http.StatusOK
	}

	return rw.ResponseWriter.Write(body)
}
