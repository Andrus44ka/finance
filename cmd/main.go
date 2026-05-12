package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"myfinance/internal/handlers"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	connStr := os.Getenv("DATABASE_URL")

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Создаём таблицы если их нет
	if err := initDB(db); err != nil {
		log.Fatal("Failed to init DB:", err)
	}

	h := handlers.NewHandler(db)
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("GET /api/accounts", h.GetAccounts)
	mux.HandleFunc("GET /api/accounts/{id}", h.GetAccount)
	mux.HandleFunc("POST /api/accounts", h.CreateAccount)
	mux.HandleFunc("DELETE /api/accounts/{id}", h.DeleteAccount)
	mux.HandleFunc("POST /api/transactions/income", h.Income)
	mux.HandleFunc("POST /api/transactions/expense", h.Expense)
	mux.HandleFunc("POST /api/transactions/transfer", h.Transfer)
	mux.HandleFunc("GET /api/transactions", h.GetTransactions)

	// Web endpoints
	mux.HandleFunc("GET /", handlers.WebHandler)
	mux.HandleFunc("GET /static/", handlers.StaticHandler)

	go func() {
		log.Println("HTTP server starting on :80")
		if err := http.ListenAndServe(":80", mux); err != nil {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
}

func initDB(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS accounts (
		id      SERIAL PRIMARY KEY,
		name    VARCHAR(100)   NOT NULL,
		balance DECIMAL(15, 2) NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id              SERIAL PRIMARY KEY,
		date            DATE           NOT NULL,
		amount          DECIMAL(15, 2) NOT NULL,
		description     TEXT,
		from_account_id INT REFERENCES accounts(id),
		to_account_id   INT REFERENCES accounts(id)
	);`

	_, err := db.Exec(query)
	return err
}
