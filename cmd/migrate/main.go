package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	"carwashinmaribackend/internal/database"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL es obligatoria")
	}
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		log.Fatalf("no se pudo conectar a PostgreSQL: %v", err)
	}
	if err = database.ApplyMigrations(ctx, db); err != nil {
		log.Fatal(err)
	}
	log.Println("migraciones aplicadas")
}
