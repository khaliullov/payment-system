package service

import (
	"context"

	"github.com/go-kit/kit/log"

	"github.com/khaliullov/payment-system/pkg/repository"
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
	accounts, err = ps.repository.GetAccounts(ctx)
	return
}

// TransactionHistory implements Service.
func (ps paymentService) TransactionHistory(ctx context.Context) (transactions []interface{}, err error) {
	transactions, err = ps.repository.GetTransactions(ctx)
	return
}

// Transfer implements Service.
func (ps paymentService) Transfer(ctx context.Context, from, to string, amount float64, currency string) (transaction *repository.Transaction, err error) {
	transaction, err = ps.repository.MakeTransfer(ctx, from, to, amount, currency)
	return
}
