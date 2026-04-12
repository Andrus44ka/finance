package handlers

import (
	"database/sql"
	"encoding/json"
	"myfinance/internal/models"
	"net/http"
	"strconv"
	"time"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{db: db}
}

// ========== СЧЕТА ==========

// GET /api/accounts - получить все счета
func (h *Handler) GetAccounts(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query("SELECT id, name, balance FROM accounts ORDER BY id")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	accounts := []models.Account{}
	for rows.Next() {
		var a models.Account
		err := rows.Scan(&a.ID, &a.Name, &a.Balance)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		accounts = append(accounts, a)
	}
	json.NewEncoder(w).Encode(accounts)
}

// GET /api/accounts/{id} - получить счет по ID
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid id", 400)
		return
	}

	var a models.Account
	err = h.db.QueryRow("SELECT id, name, balance FROM accounts WHERE id = $1", id).
		Scan(&a.ID, &a.Name, &a.Balance)
	if err == sql.ErrNoRows {
		http.Error(w, "Account not found", 404)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(a)
}

// POST /api/accounts - создать счет
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req models.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	var id int
	err := h.db.QueryRow(
		"INSERT INTO accounts (name, balance) VALUES ($1, $2) RETURNING id",
		req.Name, req.Balance,
	).Scan(&id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

// DELETE /api/accounts/{id} - удалить счет (только если нет транзакций)
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid id", 400)
		return
	}

	result, err := h.db.Exec("DELETE FROM accounts WHERE id = $1", id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "Account not found", 404)
		return
	}
	w.WriteHeader(204)
}

// ========== ТРАНЗАКЦИИ ==========

// POST /api/transactions/income - пополнение
func (h *Handler) Income(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID   int     `json:"account_id"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
		Date        string  `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	date := time.Now()
	if req.Date != "" {
		var err error
		date, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			http.Error(w, "Invalid date format, use YYYY-MM-DD", 400)
			return
		}
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	// Вставляем транзакцию
	_, err = tx.Exec(
		"INSERT INTO transactions (date, amount, description, from_account_id, to_account_id) VALUES ($1, $2, $3, NULL, $4)",
		date, req.Amount, req.Description, req.AccountID,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Обновляем баланс
	_, err = tx.Exec("UPDATE accounts SET balance = balance + $1 WHERE id = $2", req.Amount, req.AccountID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// POST /api/transactions/expense - трата
func (h *Handler) Expense(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AccountID   int     `json:"account_id"`
		Amount      float64 `json:"amount"`
		Description string  `json:"description"`
		Date        string  `json:"date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	date := time.Now()
	if req.Date != "" {
		var err error
		date, err = time.Parse("2006-01-02", req.Date)
		if err != nil {
			http.Error(w, "Invalid date format, use YYYY-MM-DD", 400)
			return
		}
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	// Проверяем достаточно ли денег (опционально, если не хотим отрицательный баланс)
	var balance float64
	err = tx.QueryRow("SELECT balance FROM accounts WHERE id = $1", req.AccountID).Scan(&balance)
	if err != nil {
		http.Error(w, "Account not found", 404)
		return
	}
	if balance < req.Amount {
		http.Error(w, "Insufficient funds", 400)
		return
	}

	_, err = tx.Exec(
		"INSERT INTO transactions (date, amount, description, from_account_id, to_account_id) VALUES ($1, $2, $3, $4, NULL)",
		date, req.Amount, req.Description, req.AccountID,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_, err = tx.Exec("UPDATE accounts SET balance = balance - $1 WHERE id = $2", req.Amount, req.AccountID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// POST /api/transactions/transfer - перевод между счетами
func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req models.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer tx.Rollback()

	// Проверяем достаточно ли денег
	var balance float64
	err = tx.QueryRow("SELECT balance FROM accounts WHERE id = $1", req.FromAccountID).Scan(&balance)
	if err != nil {
		http.Error(w, "Source account not found", 404)
		return
	}
	if balance < req.Amount {
		http.Error(w, "Insufficient funds", 400)
		return
	}

	// Вставляем транзакцию
	_, err = tx.Exec(
		"INSERT INTO transactions (date, amount, description, from_account_id, to_account_id) VALUES ($1, $2, $3, $4, $5)",
		time.Now(), req.Amount, req.Description, req.FromAccountID, req.ToAccountID,
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Обновляем балансы
	_, err = tx.Exec("UPDATE accounts SET balance = balance - $1 WHERE id = $2", req.Amount, req.FromAccountID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	_, err = tx.Exec("UPDATE accounts SET balance = balance + $1 WHERE id = $2", req.Amount, req.ToAccountID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if err = tx.Commit(); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(201)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GET /api/transactions - получить все транзакции (с фильтрацией)
func (h *Handler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	accountID := r.URL.Query().Get("account_id") // опционально: фильтр по счету
	limit := r.URL.Query().Get("limit")
	if limit == "" {
		limit = "100"
	}

	query := `
        SELECT id, date, amount, description, COALESCE(from_account_id, 0), COALESCE(to_account_id, 0)
        FROM transactions
    `
	args := []interface{}{}

	if accountID != "" {
		query += " WHERE from_account_id = $1 OR to_account_id = $1"
		args = append(args, accountID)
	}

	query += " ORDER BY date DESC LIMIT $" + strconv.Itoa(len(args)+1)
	args = append(args, limit)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	transactions := []models.Transaction{}
	for rows.Next() {
		var t models.Transaction
		var fromID, toID int
		err := rows.Scan(&t.ID, &t.Date, &t.Amount, &t.Description, &fromID, &toID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if fromID != 0 {
			t.FromAccountID = &fromID
		}
		if toID != 0 {
			t.ToAccountID = &toID
		}
		transactions = append(transactions, t)
	}
	json.NewEncoder(w).Encode(transactions)
}
