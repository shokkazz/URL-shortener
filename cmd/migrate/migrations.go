package main

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"shortener-service/internal/config"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("load config:", err)
	}

	db, err := sql.Open("pgx", cfg.Dbc.DSN())
	if err != nil {
		log.Fatal("open database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("ping database:", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal("set goose dialect:", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatal("run migrations:", err)
	}

	log.Println("migrations completed")
}
