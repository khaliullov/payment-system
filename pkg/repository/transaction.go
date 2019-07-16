package repository

import (
	"time"
)

// Transaction represents transaction history record of payment system.
type Transaction struct {
	TxnID     int       `json:"-"`
	Direction string    `json:"direction"`
	Date      time.Time `json:"-"`
	Payer     string    `json:"account"`
	Payee     string    `json:"to_account"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"-"`
	Error     string    `json:"error"`
}

type TransactionIncoming struct {
	TxnID     int       `json:"-"`
	Direction string    `json:"direction"`
	Date      time.Time `json:"-"`
	Payer     string    `json:"from_account"`
	Payee     string    `json:"account"`
	Amount    float64   `json:"amount"`
	Currency  string    `json:"-"`
	Error     string    `json:"error"`
}
