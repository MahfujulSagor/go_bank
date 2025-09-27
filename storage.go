package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type Storage interface {
	CreateAccount(*Account) (int64, error)
	DeleteAccount(int64) (int64, error)
	UpdateAccountBalance(int64, *Account) (int64, error)
	TransferMoney(int64, int64, float64) error
	GetAccounts() ([]*Account, error)
	GetAccountByID(int64) (*Account, error)
}

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage() (*PostgresStorage, error) {
	connStr := "user=postgres dbname=postgres password=mypassword host=localhost port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	// Ping the database to ensure the connection is valid
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) Init() error {
	if err := s.createAccountTable(); err != nil {
		return err
	}
	return nil
}

func (s *PostgresStorage) createAccountTable() error {
	query := `CREATE TABLE IF NOT EXISTS account(
		id SERIAL PRIMARY KEY,
		first_name VARCHAR(100),
		last_name VARCHAR(100),
		number BIGINT UNIQUE,
		balance FLOAT DEFAULT 0.0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (s *PostgresStorage) CreateAccount(account *Account) (int64, error) {
	query := `INSERT INTO account(
			first_name, 
			last_name, 
			number,
			balance, 
			created_at, 
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	var id int64
	err := s.db.QueryRow(query, account.FirstName, account.LastName, account.Number, account.Balance, account.CreatedAt, account.UpdatedAt).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no rows returned")
		}
		return 0, err
	}
	return id, nil
}
func (s *PostgresStorage) DeleteAccount(id int64) (int64, error) {
	res, err := s.db.Query(`SELECT id FROM account WHERE id = $1`, id)
	if err != nil {
		return 0, err
	}
	if !res.Next() {
		return 0, nil
	}

	_, err = s.db.Exec(`DELETE FROM account WHERE id = $1`, id)
	if err != nil {
		return 0, err
	}

	return id, nil
}
func (s *PostgresStorage) UpdateAccountBalance(id int64, account *Account) (int64, error) {
	// Check if account EXISTS
	_, err := s.db.Query(`SELECT id FROM account WHERE id = $1`, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("account not found")
		}
		return 0, err
	}

	// Update account details
	res, err := s.db.Exec(`UPDATE account SET 
		balance = $1,
		updated_at = $2
		WHERE id = $3
		AND number = $4`,
		account.Balance,
		time.Now().UTC(),
		id,
		account.Number,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no rows updated")
		}
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected == 0 {
		return 0, fmt.Errorf("no rows updated")
	}
	return id, nil
}
func (s *PostgresStorage) TransferMoney(fromAccountNo, toAccountNo int64, amount float64) error {
	// Start a transaction
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	// Ensure to rollback the transaction in case of an error
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	var fromBalance float64
	err = tx.QueryRow(`SELECT balance FROM account WHERE number = $1 FOR UPDATE`, fromAccountNo).Scan(&fromBalance)
	if err != nil {
		return err
	}
	if fromBalance < amount {
		return fmt.Errorf("insufficient funds")
	}

	var toBalance float64
	err = tx.QueryRow(`SELECT balance FROM account WHERE number = $1 FOR UPDATE`, toAccountNo).Scan(&toBalance)
	if err != nil {
		return err
	}

	newFromBalance := fromBalance - amount
	newToBalance := toBalance + amount

	_, err = tx.Exec(`UPDATE account SET balance = $1, updated_at = CURRENT_TIMESTAMP WHERE number = $2`, newFromBalance, fromAccountNo)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE account SET balance = $1, updated_at = CURRENT_TIMESTAMP WHERE number = $2`, newToBalance, toAccountNo)
	if err != nil {
		return err
	}

	return nil
}
func (s *PostgresStorage) GetAccountByID(id int64) (*Account, error) {
	query := `SELECT id, first_name, last_name, number, balance, created_at, updated_at
		FROM account
		WHERE id = $1
		LIMIT 1`
	row := s.db.QueryRow(query, id)
	account := &Account{}
	if err := row.Scan(
		&account.ID,
		&account.FirstName,
		&account.LastName,
		&account.Number,
		&account.Balance,
		&account.CreatedAt,
		&account.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return account, nil
}
func (s *PostgresStorage) GetAccounts() ([]*Account, error) {
	res, err := s.db.Query(`
		SELECT id, first_name, last_name, number, balance, created_at, updated_at
		FROM account
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var accounts []*Account
	for res.Next() {
		account := &Account{}
		if err := res.Scan(
			&account.ID,
			&account.FirstName,
			&account.LastName,
			&account.Number,
			&account.Balance,
			&account.CreatedAt,
			&account.UpdatedAt,
		); err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	if err := res.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}
