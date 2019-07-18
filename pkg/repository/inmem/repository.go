package inmem

import (
	"database/sql"
	"sync"

	"github.com/khaliullov/payment-system/pkg/repository"
)

// New returns an inmem payment Repository.
func NewInmem() repository.Repository {
	// return  repository
	return &RepositoryInmem{
		Accounts:     make([]*repository.Account, 0),
		Transactions: make([]interface{}, 0),
		acMutex:      new(sync.RWMutex),
		txMutex:      new(sync.RWMutex),
	}
}

type RepositoryInmem struct {
	Accounts     []*repository.Account
	Transactions []interface{}
	acMutex      *sync.RWMutex
	txMutex      *sync.RWMutex
}

type fakeDBTransaction struct{}

func (fdbt fakeDBTransaction) Commit() error {
	return nil
}

func (fdbt fakeDBTransaction) Rollback() error {
	return nil
}

func (fdbt fakeDBTransaction) QueryRow(query string, args ...interface{}) *sql.Row {
	return nil
}

func (fdbt fakeDBTransaction) Exec(query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

// GetAccounts returns all Accounts
func (ir *RepositoryInmem) GetAccounts() ([]*repository.Account, error) {
	accounts := make([]*repository.Account, len(ir.Accounts))
	ir.acMutex.RLock()
	copy(accounts, ir.Accounts)
	defer ir.acMutex.RUnlock()
	return accounts, nil
}

// GetTransactions returns all Transaction history.
func (ir *RepositoryInmem) GetTransactions() ([]interface{}, error) {
	transactions := make([]interface{}, len(ir.Transactions))
	ir.txMutex.RLock()
	copy(transactions, ir.Transactions)
	defer ir.txMutex.RUnlock()
	return transactions, nil
}

// Begin - start transaction
func (ir *RepositoryInmem) Begin() (repository.DBTransaction, error) {
	return fakeDBTransaction{}, nil
}

// GetAndLockTwoAccounts - get 2 accounts from store
func (ir *RepositoryInmem) GetAndLockTwoAccounts(txn repository.DBTransaction, lockingOrder repository.LockOrder, accountName1, accountName2 string) (acc1, acc2 *repository.Account, err error) {
	acc1 = ir.getAccount(accountName1)
	if acc1 == nil {
		if lockingOrder == repository.LockFromTo {
			return nil, nil, repository.ErrPayerNotFound
		}
		return nil, nil, repository.ErrPayeeNotFound
	}
	acc2 = ir.getAccount(accountName2)
	if acc2 == nil {
		if lockingOrder == repository.LockToFrom {
			return nil, nil, repository.ErrPayerNotFound
		}
		return nil, nil, repository.ErrPayeeNotFound
	}

	return acc1, acc2, nil
}

func (ir *RepositoryInmem) getAccount(accountName string) *repository.Account {
	ir.acMutex.RLock()
	defer ir.acMutex.RUnlock()
	for _, acc := range ir.Accounts {
		if acc.UserID == accountName {
			return acc
		}
	}
	return nil
}

// InsertTransaction insert transaction into the transcation history
func (ir *RepositoryInmem) InsertTransaction(direction, from, to string, amount float64, currency, txnError string) (err error) {
	ir.txMutex.Lock()
	defer ir.txMutex.Unlock()
	if direction == repository.DirectionIncoming {
		transaction := &repository.TransactionIncoming{
			Direction: direction,
			Payer:     from,
			Payee:     to,
			Amount:    amount,
			Currency:  currency,
		}
		ir.Transactions = append(ir.Transactions, transaction)
	} else {
		transaction := &repository.Transaction{
			Direction: direction,
			Payer:     from,
			Payee:     to,
			Amount:    amount,
			Currency:  currency,
		}
		ir.Transactions = append(ir.Transactions, transaction)
	}
	return nil
}

// UpdateBalance - set new balance for account
func (ir *RepositoryInmem) UpdateBalance(txn repository.DBTransaction, accountName string, balance float64) (err error) {
	account := ir.getAccount(accountName)
	ir.acMutex.Lock()
	defer ir.acMutex.Unlock()
	if account != nil {
		account.Balance = balance
		return nil
	}
	return sql.ErrNoRows
}

// FlushStore - reset store (flush/purge all data)
func (ir *RepositoryInmem) FlushStore() {
	ir.acMutex.Lock()
	ir.txMutex.Lock()
	defer func() {
		ir.acMutex.Unlock()
		ir.txMutex.Unlock()
	}()
	ir.Accounts = ir.Accounts[:0]
	ir.Transactions = ir.Transactions[:0]
}

// InsertAccount - inserts account into store
func (ir *RepositoryInmem) InsertAccount(account *repository.Account) {
	ir.acMutex.Lock()
	defer ir.acMutex.Unlock()
	ir.Accounts = append(ir.Accounts, account)
}
