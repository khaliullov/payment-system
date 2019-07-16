package transport

import (
	"context"
	"encoding/json"
	"net/http"

	ep "github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	trans "github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/khaliullov/payment-system/pkg/endpoint"
	"github.com/khaliullov/payment-system/pkg/repository"
)

// NewHTTPHandler returns an HTTP handler that makes a set of endpoints
// available on predefined paths.
func NewHTTPHandler(endpoints endpoint.Set, logger log.Logger) http.Handler {
	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerErrorHandler(trans.NewLogErrorHandler(logger)),
	}

	m := mux.NewRouter()
	m.Methods("GET").Path("/v1/healthcheck").Handler(httptransport.NewServer(
		endpoints.HealthCheckEndpoint,
		decodeHTTPHealthCheckRequest,
		encodeHTTPGenericResponse,
		options...,
	))
	m.Methods("GET").Path("/v1/accounts").Handler(httptransport.NewServer(
		endpoints.AccountEndpoint,
		decodeHTTPAccountRequest,
		encodeHTTPGenericResponse,
		options...,
	))
	m.Methods("GET").Path("/v1/payments").Handler(httptransport.NewServer(
		endpoints.TransactionHistoryEndpoint,
		decodeHTTPTransactionRequest,
		encodeHTTPGenericResponse,
		options...,
	))
	m.Methods("POST").Path("/v1/transfer").Handler(httptransport.NewServer(
		endpoints.TransferEndpoint,
		decodeHTTPTransferRequest,
		encodeHTTPGenericResponse,
		options...,
	))
	return m
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.WriteHeader(err2code(err))
	_ = json.NewEncoder(w).Encode(errorWrapper{Success: false, Error: err.Error()})
}

func err2code(err error) int {
	switch err {
	case repository.ErrInsufficientFunds, repository.ErrPayerNotFound, repository.ErrPayeeNotFound:
		return http.StatusBadRequest
	case repository.ErrSelfTransfer, repository.ErrRequiredArgumentMissing, repository.ErrWrongCurrency:
		return http.StatusBadRequest
	case repository.ErrDifferentCurrency:
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

type errorWrapper struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// decodeHTTPHealthCheckRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded HealthCheck request from the HTTP request body. Primarily useful in a
// server.
func decodeHTTPHealthCheckRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return endpoint.HealthCheckRequest{}, nil
}

// decodeHTTPAccountRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded Account request from the HTTP request body. Primarily useful in a
// server.
func decodeHTTPAccountRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

// decodeHTTPTransactionRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded TransactionHistory request from the HTTP request body. Primarily useful in a
// server.
func decodeHTTPTransactionRequest(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

// decodeHTTPTransferRequest is a transport/http.DecodeRequestFunc that decodes a
// JSON-encoded Transfer request from the HTTP request body. Primarily useful in a
// server.
func decodeHTTPTransferRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req endpoint.TransferRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

// encodeHTTPGenericResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer. Primarily useful in a server.
func encodeHTTPGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if f, ok := response.(ep.Failer); ok && f.Failed() != nil {
		errorEncoder(ctx, f.Failed(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
