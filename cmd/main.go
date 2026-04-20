package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"myfinance/internal/handlers"
	"myfinance/internal/telegram"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
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
		log.Println("HTTP server starting on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatal(err)
		}
	}()

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	apiURL := os.Getenv("API_URL")

	bot, err := telegram.NewFinanceBot(botToken, apiURL)
	if err != nil {
		log.Println("!!")
		log.Fatal(err)
	}

	go func() {
		log.Println("Starting Telegram bot...")
		bot.Start()
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
}
