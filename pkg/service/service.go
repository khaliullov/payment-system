package service

import (
	"context"
	"errors"

	"github.com/go-kit/kit/log"

	"github.com/khaliullov/payment-system/pkg/repository"
)

var (
	// ErrRequiredArgumentMissing - not enough parameters or they empty
	ErrRequiredArgumentMissing = errors.New("Required argument missing or it is incorrect")

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

// Service describes a service that adds things together.
type Service interface {
	HealthCheck(context.Context) (bool, error)
	Account(context.Context) ([]*repository.Account, error)
	TransactionHistory(context.Context) ([]interface{}, error)
	Transfer(context.Context, string, string, float64, string) (*repository.Transaction, error)
}

// New returns a payment Service with all of the expected middlewares wired in.
func New(repository repository.Repository, logger log.Logger) Service {
	var svc Service
	{
		svc = NewPaymentService(repository)
		svc = LoggingMiddleware(logger)(svc)
	}
	return svc
}

// NewPaymentService returns a naïve, stateless implementation of Service.
func NewPaymentService(repository repository.Repository) Service {
	return paymentService{
		repository: repository,
	}
}

type paymentService struct {
	repository repository.Repository
}

// HealthCheck implements Service.
func (ps paymentService) HealthCheck(_ context.Context) (bool, error) {
	return true, nil
}

// Account implements Service.
func (ps paymentService) Account(ctx context.Context) (accounts []*repository.Account, err error) {
	accounts, err = ps.repository.GetAccounts()
	return
}

// TransactionHistory implements Service.
func (ps paymentService) TransactionHistory(ctx context.Context) (transactions []interface{}, err error) {
	transactions, err = ps.repository.GetTransactions()
	return
}

// Transfer implements Service.
func (ps paymentService) Transfer(ctx context.Context, from, to string, amount float64, currency string) (txnOut *repository.Transaction, err error) {
	if from == "" || to == "" || amount <= 0 {
		return nil, ErrRequiredArgumentMissing
	}

	if from == to {
		return nil, ErrSelfTransfer
	}

	txn, err := ps.repository.Begin()
	if err != nil { // failed to start txn
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = txn.Rollback()
		}
		if txnOut != nil {
			if err == nil {
				_ = ps.repository.InsertTransaction(repository.DirectionIncoming, from, to, amount, txnOut.Currency, "")
				_ = ps.repository.InsertTransaction(repository.DirectionOutgoing, from, to, amount, txnOut.Currency, "")
			} else {
				_ = ps.repository.InsertTransaction(repository.DirectionOutgoing, from, to, amount, txnOut.Currency, err.Error())
			}
		}
	}()

	var toAccount, fromAccount *repository.Account
	if from > to {
		toAccount, fromAccount, err = ps.repository.GetAndLockTwoAccounts(txn, repository.LockToFrom, to, from)
	} else {
		fromAccount, toAccount, err = ps.repository.GetAndLockTwoAccounts(txn, repository.LockFromTo, from, to)
	}

	if err != nil { // record not found or failed to lock
		return nil, err
	}

	txnOut = &repository.Transaction{
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

	err = ps.repository.UpdateBalance(txn, from, fromAccount.Balance-amount)
	if err != nil {
		return txnOut, ErrTransactionFailed
	}

	err = ps.repository.UpdateBalance(txn, to, toAccount.Balance+amount)
	if err != nil {
		return txnOut, ErrTransactionFailed
	}

	err = txn.Commit()
	if err != nil {
		return txnOut, ErrTransactionFailed
	}

	return
}
