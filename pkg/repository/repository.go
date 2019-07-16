package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// Account lock order
type LockOrder int

// Account lock order
const (
	LockFromTo LockOrder = iota
	LockToFrom
)

var (
	// DirectionIncoming incoming transaction direction
	DirectionIncoming = "incoming"

	// DirectionOutgoing outgoing transaction direction
	DirectionOutgoing = "outgoing"

	// QueryLock is a query for locking account for update
	QueryLock = "SELECT user_id, balance, currency FROM account WHERE user_id = $1 FOR NO KEY UPDATE"

	// QueryUpdate is a query for updating accounts balance
	QueryUpdate = "UPDATE account SET balance = $1 WHERE user_id = $2"

	// QueryInsert is a query for inserting trasaction into history
	QueryInsert = "INSERT INTO payment(direction, payer, payee, amount, currency, error) VALUES ($1, $2, $3, $4, $5, $6)"

	// ErrRequiredArgumentMissing - not enough parameters or they empty
	ErrRequiredArgumentMissing = errors.New("Required argument missing or it is incorrect")

	// ErrPayerNotFound error fired when payer (sender) not found
	ErrPayerNotFound = errors.New("Payer not found")

	// ErrPayeeNotFound error fired when payee (receiver) not found
	ErrPayeeNotFound = errors.New("Payee not found")

	// ErrSelfTransfer error fired when payee equals payer
	ErrSelfTransfer = errors.New("Transfer to self")

	// ErrInsufficientFunds error fired when not enough money for transfer
	ErrInsufficientFunds = errors.New("Insufficient funds")

	// ErrDifferentCurrency error fired when account have different currencies
	ErrDifferentCurrency = errors.New("Different currency")

	// ErrWrongCurrency error fired when trying to make transfer with different currency from account's currency
	ErrWrongCurrency = errors.New("Wrong currency")

	// ErrTransactionFailed error fired when DB failes to make transaction
	ErrTransactionFailed = errors.New("Transaction failed")
)

type Repository interface {
	GetAccounts(context.Context) ([]*Account, error)
	GetTransactions(context.Context) ([]interface{}, error)
	MakeTransfer(context.Context, string, string, float64, string) (*Transaction, error)
}

// New returns a payment Repository.
func New(db *sql.DB, logger log.Logger) Repository {
	// return  repository
	return repository{
		db:     db,
		logger: log.With(logger, "repository", "paymentsdb"),
	}
}

type repository struct {
	db     *sql.DB
	logger log.Logger
}

// GetAccounts returns all Accounts
func (r repository) GetAccounts(ctx context.Context) ([]*Account, error) {
	rows, err := r.db.Query("SELECT user_id, balance, currency FROM account")
	if err != nil {
		_ = level.Error(r.logger).Log("method", "GetAccounts", "err", err)
		return nil, err
	}
	defer rows.Close()

	var accounts = make([]*Account, 0)
	for rows.Next() {
		account := &Account{}
		err := rows.Scan(&account.UserID, &account.Balance, &account.Currency)
		if err != nil {
			_ = level.Error(r.logger).Log("method", "GetAccounts", "err", err)
			return nil, err
		}
		accounts = append(accounts, account)
	}
	if err = rows.Err(); err != nil {
		_ = level.Error(r.logger).Log("method", "GetAccounts", "err", err)
		return nil, err
	}
	return accounts, nil
}

// GetTransactions returns all Transaction history.
func (r repository) GetTransactions(ctx context.Context) ([]interface{}, error) {
	rows, err := r.db.Query("SELECT txn_id, direction, date, payer, payee, amount, currency, error FROM payment")
	if err != nil {
		_ = level.Error(r.logger).Log("method", "GetTransactions", "err", err)
		return nil, err
	}
	defer rows.Close()

	var transactions = make([]interface{}, 0)
	for rows.Next() {
		txn := &Transaction{}
		err := rows.Scan(&txn.TxnID, &txn.Direction, &txn.Date, &txn.Payer, &txn.Payee, &txn.Amount, &txn.Currency,
			&txn.Error)
		if err != nil {
			_ = level.Error(r.logger).Log("method", "GetTransactions", "err", err)
			return nil, err
		}
		if txn.Direction == DirectionIncoming {
			transactions = append(transactions, &TransactionIncoming{
				TxnID:     txn.TxnID,
				Direction: txn.Direction,
				Date:      txn.Date,
				Payer:     txn.Payer,
				Payee:     txn.Payee,
				Amount:    txn.Amount,
				Currency:  txn.Currency,
				Error:     txn.Error,
			})
		} else {
			transactions = append(transactions, txn)
		}
	}
	if err = rows.Err(); err != nil {
		_ = level.Error(r.logger).Log("method", "GetTransactions", "err", err)
		return nil, err
	}
	return transactions, nil
}

// MakeTransfer transfer money from one Account to another.
func (r repository) MakeTransfer(ctx context.Context, from, to string, amount float64, currency string) (txnOut *Transaction, err error) {
	if from == "" || to == "" || amount <= 0 {
		return nil, ErrRequiredArgumentMissing
	}

	// to avoid deadlock: lock rows in alphabet order
	// https://www.citusdata.com/blog/2018/02/22/seven-tips-for-dealing-with-postgres-locks/
	// with NO KEY UPDATE flag
	// https://habr.com/ru/company/wargaming/blog/323354/

	txn, err := r.db.Begin()
	if err != nil { // failed to start txn
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = txn.Rollback()
		}
		if txnOut != nil {
			if err == nil {
				_ = r.insertTransaction(DirectionIncoming, from, to, amount, txnOut.Currency, "")
				_ = r.insertTransaction(DirectionOutgoing, from, to, amount, txnOut.Currency, "")
			} else {
				_ = r.insertTransaction(DirectionOutgoing, from, to, amount, txnOut.Currency, err.Error())
			}
		}
	}()

	if from == to {
		return nil, ErrSelfTransfer
	}

	var toAccount, fromAccount *Account
	if from > to {
		toAccount, fromAccount, err = r.lockTwoAccounts(txn, LockToFrom, to, from)
	} else {
		fromAccount, toAccount, err = r.lockTwoAccounts(txn, LockFromTo, from, to)
	}

	if err != nil { // record not found or failed to lock
		return nil, err
	}

	txnOut = &Transaction{
		Payee:    to,
		Payer:    from,
		Amount:   amount,
		Currency: fromAccount.Currency,
	}

	if currency != "" && fromAccount.Currency != currency { // check if requested currency fits to users currency (if was specifed)
		txnOut.Currency = currency
		return txnOut, ErrWrongCurrency
	}

	if fromAccount.Currency != toAccount.Currency { // compare dest and source currencies
		return txnOut, ErrDifferentCurrency
	}

	if fromAccount.Balance < amount { // check enough money for transfer
		return txnOut, ErrInsufficientFunds
	}

	err = r.updateBalance(txn, from, fromAccount.Balance-amount)
	if err != nil {
		return txnOut, ErrTransactionFailed
	}

	err = r.updateBalance(txn, to, toAccount.Balance+amount)
	if err != nil {
		return txnOut, ErrTransactionFailed
	}

	err = txn.Commit()
	if err != nil {
		return txnOut, ErrTransactionFailed
	}

	return txnOut, nil
}

func (r repository) lockTwoAccounts(txn *sql.Tx, lockingOrder LockOrder, accountName1, accountName2 string) (acc1, acc2 *Account, err error) {
	acc1, err = r.lockAccount(txn, accountName1)
	if err != nil {
		if err == sql.ErrNoRows {
			if lockingOrder == LockFromTo {
				return nil, nil, ErrPayerNotFound
			}
			return nil, nil, ErrPayeeNotFound
		}
		return nil, nil, err
	}
	acc2, err = r.lockAccount(txn, accountName2)
	if err != nil {
		if err == sql.ErrNoRows {
			if lockingOrder == LockToFrom {
				return nil, nil, ErrPayerNotFound
			}
			return nil, nil, ErrPayeeNotFound
		}
		return nil, nil, err
	}

	return acc1, acc2, nil
}

func (r repository) lockAccount(txn *sql.Tx, accountName string) (account *Account, err error) {
	account = &Account{}
	row := txn.QueryRow(QueryLock, accountName)
	err = row.Scan(&account.UserID, &account.Balance, &account.Currency)
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (r repository) updateBalance(txn *sql.Tx, accountName string, balance float64) (err error) {
	_, err = txn.Exec(QueryUpdate, balance, accountName)
	return
}

func (r repository) insertTransaction(direction, from, to string, amount float64, currency, txnError string) (err error) {
	_, err = r.db.Exec(QueryInsert, direction, from, to, amount, currency, txnError)
	return
}
