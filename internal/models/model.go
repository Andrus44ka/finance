package models

import "time"

type Account struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Balance float64 `json:"balance"`
}

type Transaction struct {
	ID            int       `json:"id"`
	Date          time.Time `json:"date"`
	Amount        float64   `json:"amount"`
	Description   string    `json:"description"`
	FromAccountID *int      `json:"from_account_id,omitempty"`
	ToAccountID   *int      `json:"to_account_id,omitempty"`
}

type CreateAccountRequest struct {
	Name    string  `json:"name" binding:"required"`
	Balance float64 `json:"balance" binding:"required"`
}

type CreateTransactionRequest struct {
	Date          string  `json:"date"` // формат "2006-01-02"
	Amount        float64 `json:"amount" binding:"required"`
	Description   string  `json:"description"`
	FromAccountID *int    `json:"from_account_id"`
	ToAccountID   *int    `json:"to_account_id"`
}

type TransferRequest struct {
	FromAccountID int     `json:"from_account_id" binding:"required"`
	ToAccountID   int     `json:"to_account_id" binding:"required"`
	Amount        float64 `json:"amount" binding:"required"`
	Description   string  `json:"description"`
}
