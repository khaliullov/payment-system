package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	ep "github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	trans "github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"

	"github.com/khaliullov/payment-system/pkg/endpoint"
	"github.com/khaliullov/payment-system/pkg/repository"
	"github.com/khaliullov/payment-system/pkg/service"
)

var (
	HealthCheckPath = "/v1/healthcheck" // for smoke test
	AccountPath     = "/v1/accounts"
	TransactionPath = "/v1/payments"
	TransferPath    = "/v1/transfer"
)

// NewHTTPHandler returns an HTTP handler that makes a set of endpoints
// available on predefined paths.
func NewHTTPHandler(endpoints endpoint.Set, logger log.Logger) http.Handler {
	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerErrorHandler(trans.NewLogErrorHandler(logger)),
	}

	m := mux.NewRouter()
	m.Methods("GET").Path(HealthCheckPath).Handler(httptransport.NewServer(
		endpoints.HealthCheckEndpoint,
		decodeHTTPHealthCheckRequest,
		encodeHTTPGenericResponse,
		options...,
	))
	m.Methods("GET").Path(AccountPath).Handler(httptransport.NewServer(
		endpoints.AccountEndpoint,
		decodeHTTPAccountRequest,
		encodeHTTPGenericResponse,
		options...,
	))
	m.Methods("GET").Path(TransactionPath).Handler(httptransport.NewServer(
		endpoints.TransactionHistoryEndpoint,
		decodeHTTPTransactionRequest,
		encodeHTTPGenericResponse,
		options...,
	))
	m.Methods("POST").Path(TransferPath).Handler(httptransport.NewServer(
		endpoints.TransferEndpoint,
		decodeHTTPTransferRequest,
		encodeHTTPGenericResponse,
		options...,
	))
	return m
}

// NewHTTPClient returns an Service backed by an HTTP server living at the
// remote instance. We expect instance to come from a service discovery system,
// so likely of the form "host:port". We bake-in certain middlewares,
// implementing the client library pattern.
func NewHTTPClient(instance string, logger log.Logger) (service.Service, error) {
	// Quickly sanitize the instance string.
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	// global client middlewares
	options := []httptransport.ClientOption{}

	// Each individual endpoint is an http/transport.Client (which implements
	// endpoint.Endpoint) that gets wrapped with various middlewares. If you
	// made your own client library, you'd do this work there, so your server
	// could rely on a consistent set of client behavior.
	var healthCheckEndpoint ep.Endpoint
	{
		healthCheckEndpoint = httptransport.NewClient(
			"GET",
			copyURL(u, AccountPath),
			encodeHTTPGenericRequest,
			decodeHTTPHealthCheckResponse,
			options...,
		).Endpoint()
	}
	var accountEndpoint ep.Endpoint
	{
		accountEndpoint = httptransport.NewClient(
			"GET",
			copyURL(u, AccountPath),
			encodeHTTPGenericRequest,
			decodeHTTPAccountResponse,
			options...,
		).Endpoint()
	}
	var transactionHistoryEndpoint ep.Endpoint
	{
		transactionHistoryEndpoint = httptransport.NewClient(
			"GET",
			copyURL(u, TransactionPath),
			encodeHTTPGenericRequest,
			decodeHTTPTransactionResponse,
			options...,
		).Endpoint()
	}
	var transferEndpoint ep.Endpoint
	{
		transferEndpoint = httptransport.NewClient(
			"POST",
			copyURL(u, TransferPath),
			encodeHTTPGenericRequest,
			decodeHTTPTransferResponse,
			options...,
		).Endpoint()
	}

	// Returning the endpoint.Set as a service.Service relies on the
	// endpoint.Set implementing the Service methods. That's just a simple bit
	// of glue code.
	return endpoint.Set{
		HealthCheckEndpoint:        healthCheckEndpoint,
		AccountEndpoint:            accountEndpoint,
		TransactionHistoryEndpoint: transactionHistoryEndpoint,
		TransferEndpoint:           transferEndpoint,
	}, nil
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
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

// decodeHTTPHealthCheckResponse is a transport/http.DecodeResponseFunc that decodes a
// JSON-encoded HealthCheck response from the HTTP response body. Primarily useful in a
// client.
func decodeHTTPHealthCheckResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var resp endpoint.HealthCheckResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// decodeHTTPAccountResponse is a transport/http.DecodeResponseFunc that decodes a
// JSON-encoded Account response from the HTTP response body. Primarily useful in a
// client.
func decodeHTTPAccountResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var resp endpoint.AccountResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// decodeHTTPTransactionResponse is a transport/http.DecodeResponseFunc that decodes a
// JSON-encoded Transaction response from the HTTP response body. Primarily useful in a
// client.
func decodeHTTPTransactionResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var resp endpoint.TransactionHistoryResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// decodeHTTPTransferResponse is a transport/http.DecodeResponseFunc that decodes a
// JSON-encoded Transfer response from the HTTP response body. Primarily useful in a
// client.
func decodeHTTPTransferResponse(_ context.Context, r *http.Response) (interface{}, error) {
	var resp endpoint.TransferResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// encodeHTTPGenericRequest is a transport/http.EncodeRequestFunc that
// JSON-encodes any request to the request body. Primarily useful in a client.
func encodeHTTPGenericRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
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
