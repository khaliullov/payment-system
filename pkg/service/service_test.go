package service

import (
	"context"
	"testing"

	"github.com/khaliullov/payment-system/pkg/repository"
	"github.com/khaliullov/payment-system/pkg/repository/inmem"
)

func TestService(t *testing.T) {
	repo := inmem.NewInmem()
	inmemRepo := repo.(*inmem.RepositoryInmem)
	svc := NewPaymentService(repo)

	inmemRepo.FlushStore()
	acc1 := &repository.Account{
		UserID:   "alice456",
		Balance:  0.01,
		Currency: "USD",
	}
	inmemRepo.InsertAccount(acc1)
	acc2 := &repository.Account{
		UserID:   "bob123",
		Balance:  100,
		Currency: "USD",
	}
	inmemRepo.InsertAccount(acc2)

	// test wrong args
	_, err := svc.Transfer(context.Background(), "", "", 0, "")
	if err != ErrRequiredArgumentMissing {
		t.Errorf("Error should be: %v, got %v", ErrRequiredArgumentMissing, err)
	}

	// test self transfer
	_, err = svc.Transfer(context.Background(), "bob123", "bob123", 0.01, "")
	if err != ErrSelfTransfer {
		t.Errorf("Error should be: %v, got %v", ErrSelfTransfer, err)
	}

	// test payer not found
	_, err = svc.Transfer(context.Background(), "vasya", "bob123", 0.01, "")
	if err != repository.ErrPayerNotFound {
		t.Errorf("Error should be: %v, got %v", repository.ErrPayerNotFound, err)
	}

	// test payee not found
	_, err = svc.Transfer(context.Background(), "alice456", "petya", 0.01, "")
	if err != repository.ErrPayeeNotFound {
		t.Errorf("Error should be: %v, got %v", repository.ErrPayeeNotFound, err)
	}

	// test Insufficient funds
	_, err = svc.Transfer(context.Background(), "alice456", "bob123", 0.02, "")
	if err != ErrInsufficientFunds {
		t.Errorf("Error should be: %v, got %v", ErrInsufficientFunds, err)
	}

	// test wrong currency
	_, err = svc.Transfer(context.Background(), "bob123", "alice456", 0.01, "RUB")
	if err != ErrWrongCurrency {
		t.Errorf("Error should be: %v, got %v", ErrWrongCurrency, err)
	}

	accounts, _ := svc.Account(context.Background())
	accounts[0].Currency = "RUB"

	// test wrong currency
	_, err = svc.Transfer(context.Background(), "alice456", "bob123", 0.01, "")
	if err != ErrDifferentCurrency {
		t.Errorf("Error should be: %v, got %v", ErrDifferentCurrency, err)
	}

	accounts[0].Currency = "USD"

	// test successful transfer
	_, err = svc.Transfer(context.Background(), "bob123", "alice456", 0.01, "")
	if err != nil {
		t.Errorf("Error should be: %v, got %v", nil, err)
	}

	// get transaction history
	transactions, _ := svc.TransactionHistory(context.Background())
	if len(transactions) == 0 {
		t.Errorf("Transaction history should be filled")
	}
}
