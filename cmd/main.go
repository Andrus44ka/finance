package main

import (
	"database/sql"
	"log"
	"net/http"

	"myfinance/internal/handlers"

	_ "github.com/lib/pq"
)

func main() {
	// Подключение к PostgreSQL
	connStr := "user=andrus44ka dbname=financedb sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Проверяем подключение
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	h := handlers.NewHandler(db)

	// Роутинг (для Go 1.22+)
	mux := http.NewServeMux()

	// Счета
	mux.HandleFunc("GET /api/accounts", h.GetAccounts)
	mux.HandleFunc("GET /api/accounts/{id}", h.GetAccount)
	mux.HandleFunc("POST /api/accounts", h.CreateAccount)
	mux.HandleFunc("DELETE /api/accounts/{id}", h.DeleteAccount)

	// Транзакции
	mux.HandleFunc("POST /api/transactions/income", h.Income)
	mux.HandleFunc("POST /api/transactions/expense", h.Expense)
	mux.HandleFunc("POST /api/transactions/transfer", h.Transfer)
	mux.HandleFunc("GET /api/transactions", h.GetTransactions)

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
