package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"educare/internal/api"
	"educare/internal/db"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	secret := os.Getenv("JWT_SECRET")
	if dbURL == "" || len(secret) < 32 {
		log.Fatal("set DATABASE_URL and JWT_SECRET (at least 32 characters)")
	}
	migrations := os.Getenv("MIGRATIONS_DIR")
	if migrations == "" {
		migrations = "migrations"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatal("cannot create database pool: ", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatal("cannot reach database: ", err)
	}
	if err := db.Migrate(ctx, pool, migrations); err != nil {
		log.Fatal("migration failed: ", err)
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           api.New(pool, secret).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stopped := make(chan os.Signal, 1)
	signal.Notify(stopped, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stopped
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}()

	fmt.Println("EduSpace API running on http://localhost:" + port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("API server stopped unexpectedly: ", err)
	}
}
