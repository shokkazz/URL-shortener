package app

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"shortener-service/internal/config"
	"shortener-service/internal/database/postgres"
	"shortener-service/internal/service"
	myhttp "shortener-service/internal/transport/http"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func StartApp() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error during environment variables load: ", err)
		return
	}
	cfg := config.LoadConfig()
	if cfg == nil {
		log.Println("Error nil config")
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
	if err := runMigrations(cfg.Dbc.DSN()); err != nil {
		log.Println("Error during migrations:", err)
		return
	}
	ur := postgres.NewURLRepository(db)
	us := service.NewShortenerService(ur, cfg.ShortURLLength, cfg.Charset)
	uh := myhttp.NewHandler(us)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /{shortenedURL}", uh.Redirect)
	mux.HandleFunc("POST /shorten", uh.CreateShortLink)
	mux.HandleFunc("GET /shorten", uh.GetShortLink)
	mux.HandleFunc("DELETE /shorten", uh.DeleteShortLink)

	loggingHandler := loggerMiddleware(mux)

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
		log.Println("server started on " + cfg.Port)
		if err := server.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	defer signal.Stop(quit)
	select {
	case err := <-serverErr:
		log.Println("Server failed:", err)

	case sig := <-quit:
		log.Println("Received signal:", sig)
	}

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = server.Shutdown(ctxShutdown); err != nil {
		log.Println("Server forced to shutdown: ", err)
	}
	log.Println("Server stopped")
}

func loggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Println("request " + r.RemoteAddr + " " + r.Method + " " + r.URL.Path)
		rw := responseRecorder{
			w,
			http.StatusOK,
		}
		next.ServeHTTP(&rw, r)
		log.Println("request finished "+" ", rw.statusCode, " "+r.RemoteAddr+" "+r.Method+" "+r.URL.Path+" in ", time.Since(start))

	},
	)
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

func runMigrations(dsn string) error {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.Up(db, "migrations")
}
