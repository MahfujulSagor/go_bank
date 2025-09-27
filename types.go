package main

import (
	"math/rand"
	"time"
)

type CreateAccountRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type UpdateAccountBalanceRequest struct {
	Balance float64 `json:"balance"`
	Number  int64   `json:"number"`
}

type TransferRequest struct {
	FromAccountNo int64   `json:"from_account_no"`
	ToAccountNo   int64   `json:"to_account_no"`
	Amount        float64 `json:"amount"`
}

type Account struct {
	ID        int64     `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Number    int64     `json:"number"`
	Balance   float64   `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewAccount(firstName, lastName string) *Account {
	return &Account{
		FirstName: firstName,
		LastName:  lastName,
		Number:    rand.Int63n(1000000000),
		Balance:   0.0,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}
