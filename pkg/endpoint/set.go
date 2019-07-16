package endpoint

import (
	"context"

	ep "github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"

	"github.com/khaliullov/payment-system/pkg/repository"
	"github.com/khaliullov/payment-system/pkg/service"
)

// Set collects all of the endpoints that compose an add service. It's meant to
// be used as a helper struct, to collect all of the endpoints into a single
// parameter.
type Set struct {
	HealthCheckEndpoint        ep.Endpoint
	AccountEndpoint            ep.Endpoint
	TransactionHistoryEndpoint ep.Endpoint
	TransferEndpoint           ep.Endpoint
}

// New returns a Set that wraps the provided server, and wires in all of the
// expected endpoint middlewares via the various parameters.
func New(svc service.Service, logger log.Logger) Set {
	var healthCheckEndpoint ep.Endpoint
	{
		healthCheckEndpoint = MakeHealthCheckEndpoint(svc)
		healthCheckEndpoint = LoggingMiddleware(log.With(logger, "method", "HealthCheck"))(healthCheckEndpoint)
	}
	var accountEndpoint ep.Endpoint
	{
		accountEndpoint = MakeAccountEndpoint(svc)
		accountEndpoint = LoggingMiddleware(log.With(logger, "method", "Account"))(accountEndpoint)
	}
	var transactionHistoryEndpoint ep.Endpoint
	{
		transactionHistoryEndpoint = MakeTransactionHistoryEndpoint(svc)
		transactionHistoryEndpoint = LoggingMiddleware(log.With(logger, "method", "TransactionHistory"))(transactionHistoryEndpoint)
	}
	var transferEndpoint ep.Endpoint
	{
		transferEndpoint = MakeTransferEndpoint(svc)
		transferEndpoint = LoggingMiddleware(log.With(logger, "method", "Transfer"))(transferEndpoint)
	}
	return Set{
		HealthCheckEndpoint:        healthCheckEndpoint,
		AccountEndpoint:            accountEndpoint,
		TransactionHistoryEndpoint: transactionHistoryEndpoint,
		TransferEndpoint:           transferEndpoint,
	}
}

// HealthCheck implements the service interface, so Set may be used as a service.
// This is primarily useful in the context of a client library.
func (s Set) HealthCheck(ctx context.Context) (bool, error) {
	resp, err := s.HealthCheckEndpoint(ctx, HealthCheckRequest{})
	if err != nil {
		return false, err
	}
	response := resp.(HealthCheckResponse)
	return response.Success, response.Error
}

// Account implements the service interface, so Set may be used as a service.
// This is primarily useful in the context of a client library.
func (s Set) Account(ctx context.Context) ([]*repository.Account, error) {
	resp, err := s.AccountEndpoint(ctx, AccountRequest{})
	if err != nil {
		return nil, err
	}
	response := resp.(AccountResponse)
	return response.Accounts, response.Error
}

// TransactionHistory implements the service interface, so Set may be used as a service.
// This is primarily useful in the context of a client library.
func (s Set) TransactionHistory(ctx context.Context) ([]interface{}, error) {
	resp, err := s.TransactionHistoryEndpoint(ctx, TransactionHistoryRequest{})
	if err != nil {
		return nil, err
	}
	response := resp.(TransactionHistoryResponse)
	return response.Transactions, response.Error
}

// Transfer implements the service interface, so Set may be used as a service.
// This is primarily useful in the context of a client library.
func (s Set) Transfer(ctx context.Context, from, to string, amount float64, currency string) (*repository.Transaction, error) {
	resp, err := s.TransferEndpoint(ctx, TransferRequest{From: from, To: to, Amount: amount, Currency: currency})
	if err != nil {
		return nil, err
	}
	response := resp.(TransferResponse)
	return nil, response.Error
}

// MakeHealthCheckEndpoint constructs a HealthCheck endpoint wrapping the service.
func MakeHealthCheckEndpoint(s service.Service) ep.Endpoint {
	return func(ctx context.Context, _ interface{}) (response interface{}, err error) {
		v, err := s.HealthCheck(ctx)
		return HealthCheckResponse{Success: v, Error: err}, nil
	}
}

// MakeAccountEndpoint constructs a Account endpoint wrapping the service.
func MakeAccountEndpoint(s service.Service) ep.Endpoint {
	return func(ctx context.Context, _ interface{}) (response interface{}, err error) {
		v, err := s.Account(ctx)
		success := false
		if err == nil {
			success = true
		}
		return AccountResponse{Success: success, Accounts: v, Error: err}, nil
	}
}

// MakeTransactionHistoryEndpoint constructs a TransactionHistory endpoint wrapping the service.
func MakeTransactionHistoryEndpoint(s service.Service) ep.Endpoint {
	return func(ctx context.Context, _ interface{}) (response interface{}, err error) {
		v, err := s.TransactionHistory(ctx)
		success := false
		if err == nil {
			success = true
		}
		return TransactionHistoryResponse{Success: success, Transactions: v, Error: err}, nil
	}
}

// MakeTransferEndpoint constructs a Transfer endpoint wrapping the service.
func MakeTransferEndpoint(s service.Service) ep.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(TransferRequest)
		_, err = s.Transfer(ctx, req.From, req.To, req.Amount, req.Currency)
		success := false
		if err == nil {
			success = true
		}
		return TransferResponse{Success: success, Error: err}, nil
	}
}

// compile time assertions for our response types implementing endpoint.Failer.
var (
	_ ep.Failer = HealthCheckResponse{}
	_ ep.Failer = AccountResponse{}
	_ ep.Failer = TransactionHistoryResponse{}
	_ ep.Failer = TransferResponse{}
)

// HealthCheckRequest collects the request parameters for the HealthCheck method.
type HealthCheckRequest struct{}

// AccountRequest collects the request parameters for the Account method.
type AccountRequest struct{}

// TransactionHistoryRequest collects the request parameters for the TransactionHistory method.
type TransactionHistoryRequest struct{}

// TransferRequest collects the request parameters for the Transfer method.
type TransferRequest struct {
	From, To, Currency string
	Amount             float64
}

// HealthCheckResponse collects the response values for the HealthCheck method.
type HealthCheckResponse struct {
	Success bool  `json:"success"`
	Error   error `json:"error,omitempty"`
}

// AccountResponse collects the response values for the Account method.
type AccountResponse struct {
	Success  bool                  `json:"success"`
	Accounts []*repository.Account `json:"accounts"`
	Error    error                 `json:"error,omitempty"`
}

// TransactionHistoryResponse collects the response values for the TransactionHistory method.
type TransactionHistoryResponse struct {
	Success      bool          `json:"success"`
	Transactions []interface{} `json:"payments"`
	Error        error         `json:"error,omitempty"`
}

// TransferResponse collects the response values for the Transfer method.
type TransferResponse struct {
	Success bool  `json:"success"`
	Error   error `json:"error,omitempty"`
}

// Failed implements endpoint.Failer.
func (r HealthCheckResponse) Failed() error {
	return r.Error
}

// Failed implements endpoint.Failer.
func (r AccountResponse) Failed() error {
	return r.Error
}

// Failed implements endpoint.Failer.
func (r TransactionHistoryResponse) Failed() error {
	return r.Error
}

// Failed implements endpoint.Failer.
func (r TransferResponse) Failed() error {
	return r.Error
}
