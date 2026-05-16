-- +goose Up
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
);

-- +goose Down
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;