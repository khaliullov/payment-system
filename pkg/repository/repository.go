package repository

import (
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

	// QueryAccount is a query for fetching all accounts
	QueryAccount = "SELECT user_id, balance, currency FROM account"

	// QueryTransaction is a query for fetching all transactions
	QueryTransaction = "SELECT txn_id, direction, date, payer, payee, amount, currency, error FROM payment"

	// QueryUpdate is a query for updating accounts balance
	QueryUpdate = "UPDATE account SET balance = $1 WHERE user_id = $2"

	// QueryInsert is a query for inserting trasaction into history
	QueryInsert = "INSERT INTO payment(direction, payer, payee, amount, currency, error) VALUES ($1, $2, $3, $4, $5, $6)"

	// ErrPayerNotFound error fired when payer (sender) not found
	ErrPayerNotFound = errors.New("Payer not found")

	// ErrPayeeNotFound error fired when payee (receiver) not found
	ErrPayeeNotFound = errors.New("Payee not found")
)

type Repository interface {
	GetAccounts() ([]*Account, error)
	GetTransactions() ([]interface{}, error)
	Begin() (DBTransaction, error)
	GetAndLockTwoAccounts(txn DBTransaction, lockingOrder LockOrder, accountName1, accountName2 string) (acc1, acc2 *Account, err error)
	InsertTransaction(direction, from, to string, amount float64, currency, txnError string) (err error)
	UpdateBalance(txn DBTransaction, accountName string, balance float64) (err error)
}

// New returns a payment Repository.
func New(db *sql.DB, logger log.Logger) Repository {
	// return  repository
	return &repository{
		db:     db,
		logger: log.With(logger, "repository", "paymentsdb"),
	}
}

type repository struct {
	db     *sql.DB
	logger log.Logger
}

// DBTransaction is a wrapper for sql.Tx
type DBTransaction interface {
	Rollback() error
	Commit() error
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// NewDBTranscation creates new instance with wrapped sql.Tx
func NewDBTranscation(txn *sql.Tx) DBTransaction {
	return dbTransaction{
		txn: txn,
	}
}

type dbTransaction struct {
	txn *sql.Tx
}

// Rollback is a wrapper for Rollback
func (dbt dbTransaction) Rollback() error {
	return dbt.txn.Rollback()
}

// Commit is a wrapper
func (dbt dbTransaction) Commit() error {
	return dbt.txn.Commit()
}

// QueryRow wrapper
func (dbt dbTransaction) QueryRow(query string, args ...interface{}) *sql.Row {
	return dbt.txn.QueryRow(query, args...)
}

// Exec wrapper
func (dbt dbTransaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	return dbt.txn.Exec(query, args...)
}

// GetAccounts returns all Accounts
func (r *repository) GetAccounts() ([]*Account, error) {
	rows, err := r.db.Query(QueryAccount)
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
func (r *repository) GetTransactions() ([]interface{}, error) {
	rows, err := r.db.Query(QueryTransaction)
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

func (r *repository) Begin() (DBTransaction, error) {
	txn, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	return NewDBTranscation(txn), nil
}

func (r *repository) GetAndLockTwoAccounts(txn DBTransaction, lockingOrder LockOrder, accountName1, accountName2 string) (acc1, acc2 *Account, err error) {
	acc1, err = r.getAndLockAccount(txn, accountName1)
	if err != nil {
		if err == sql.ErrNoRows {
			if lockingOrder == LockFromTo {
				return nil, nil, ErrPayerNotFound
			}
			return nil, nil, ErrPayeeNotFound
		}
		return nil, nil, err
	}
	acc2, err = r.getAndLockAccount(txn, accountName2)
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

func (r *repository) getAndLockAccount(txn DBTransaction, accountName string) (account *Account, err error) {
	account = &Account{}
	row := txn.QueryRow(QueryLock, accountName)
	err = row.Scan(&account.UserID, &account.Balance, &account.Currency)
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (r *repository) UpdateBalance(txn DBTransaction, accountName string, balance float64) (err error) {
	_, err = txn.Exec(QueryUpdate, balance, accountName)
	return
}

func (r *repository) InsertTransaction(direction, from, to string, amount float64, currency, txnError string) (err error) {
	_, err = r.db.Exec(QueryInsert, direction, from, to, amount, currency, txnError)
	return
}
