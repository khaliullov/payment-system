package service

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"

	"github.com/khaliullov/payment-system/pkg/repository"
)

// Middleware describes a service (as opposed to endpoint) middleware.
type Middleware func(Service) Service

// LoggingMiddleware takes a logger as a dependency
// and returns a ServiceMiddleware.
func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return loggingMiddleware{logger, next}
	}
}

type loggingMiddleware struct {
	logger log.Logger
	next   Service
}

func (mw loggingMiddleware) HealthCheck(ctx context.Context) (success bool, err error) {
	defer func() {
		_ = level.Info(mw.logger).Log("method", "HealthCheck", "success", success, "err", err)
	}()
	return mw.next.HealthCheck(ctx)
}

func (mw loggingMiddleware) Account(ctx context.Context) (_ []*repository.Account, err error) {
	defer func() {
		_ = level.Info(mw.logger).Log("method", "Account", "err", err)
	}()
	return mw.next.Account(ctx)
}

func (mw loggingMiddleware) TransactionHistory(ctx context.Context) (_ []interface{}, err error) {
	defer func() {
		_ = level.Info(mw.logger).Log("method", "TransactionHistory", "err", err)
	}()
	return mw.next.TransactionHistory(ctx)
}

func (mw loggingMiddleware) Transfer(ctx context.Context, from, to string, amount float64, currency string) (_ *repository.Transaction, err error) {
	defer func() {
		_ = level.Info(mw.logger).Log("method", "Transfer", "from", from, "to", to, "amount", amount, "currency", currency, "err", err)
	}()
	return mw.next.Transfer(ctx, from, to, amount, currency)
}
