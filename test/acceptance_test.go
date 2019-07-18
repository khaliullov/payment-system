// +build !test at

package test

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/gherkin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	_ "github.com/lib/pq"

	"github.com/khaliullov/payment-system/pkg/repository"
	"github.com/khaliullov/payment-system/pkg/service"
	"github.com/khaliullov/payment-system/pkg/transport"
)

type apiFeature struct {
	httpHost string
	httpPort int

	dbHost     string
	dbPort     int
	dbName     string
	dbUser     string
	dbPassword string

	db *sql.DB

	logger log.Logger

	client service.Service

	accounts     []*repository.Account
	transactions []interface{}

	Args      map[string]string
	lastError error
}

func (af *apiFeature) init() {
	// Logging domain.
	{
		af.logger = log.NewLogfmtLogger(os.Stderr)
		af.logger = level.NewFilter(af.logger, level.AllowDebug())
		af.logger = log.With(af.logger, "ts", log.DefaultTimestampUTC)
		af.logger = log.With(af.logger, "caller", log.DefaultCaller)
	}
	af.dbHost = envString("DB_HOST", "localhost")
	af.dbPort = envInt("DB_PORT", 5432)
	af.dbName = envString("DB_NAME", "psdb")
	af.dbUser = envString("DB_USER", "postgres")
	af.dbPassword = envString("DB_PASSWORD", "postgres")
	DSN := &url.URL{
		Scheme:   "postgresql",
		RawQuery: "sslmode=disable",
		Host:     af.dbHost + ":" + strconv.Itoa(af.dbPort),
		Path:     af.dbName,
		User:     url.UserPassword(af.dbUser, af.dbPassword),
	}
	var err error
	af.db, err = sql.Open("postgres", DSN.String())
	if err != nil {
		_ = level.Info(af.logger).Log("DB err", err.Error())
		os.Exit(1)
	}
	af.httpHost = envString("HTTP_HOST", "localhost")
	af.httpPort = envInt("HTTP_PORT", 8080)
	af.client, _ = transport.NewHTTPClient(af.httpHost+":"+strconv.Itoa(af.httpPort), af.logger)
	af.Args = make(map[string]string)
}

func (af *apiFeature) TruncateDB() {
	_, err := af.db.Exec("TRUNCATE account RESTART IDENTITY CASCADE")
	if err != nil {
		_ = level.Info(af.logger).Log("DB error", err.Error())
		os.Exit(1)
	}
	_, err = af.db.Exec("TRUNCATE payment RESTART IDENTITY CASCADE")
	if err != nil {
		_ = level.Info(af.logger).Log("DB error", err.Error())
		os.Exit(1)
	}
}

func (af *apiFeature) deInit() {
	af.TruncateDB()
	af.db.Close()
}

func (af *apiFeature) theFollowingListExist(tableName string, recordList *gherkin.DataTable) error {
	var fields, marks []string
	fields = make([]string, 0)
	marks = make([]string, 0)
	head := recordList.Rows[0].Cells
	for n, cell := range head {
		fields = append(fields, cell.Value)
		marks = append(marks, "$"+strconv.Itoa(n+1))
	}
	query := "INSERT INTO " + tableName + "(" + strings.Join(fields, ", ") + ") VALUES(" + strings.Join(marks, ", ") + ")"
	for i := 1; i < len(recordList.Rows); i++ {
		vals := make([]interface{}, 0)
		for n, cell := range recordList.Rows[i].Cells {
			switch head[n].Value {
			case "balance", "amount":
				value, err := strconv.ParseFloat(cell.Value, 64)
				if err != nil {
					_ = level.Info(af.logger).Log("table", tableName, "column", head[n].Value, "value", cell.Value, "err", err.Error())
					os.Exit(1)
				}
				vals = append(vals, value)
			default:
				vals = append(vals, cell.Value)
			}
		}
		_, err := af.db.Exec(query, vals...)
		if err != nil {
			_ = level.Info(af.logger).Log("table", tableName, "values", vals, "err", err.Error())
			os.Exit(1)
		}
	}
	return nil
}

func (af *apiFeature) iSendRequestTo(_, requestPath string) error {
	switch requestPath {
	case transport.AccountPath:
		var err error
		af.accounts, err = af.client.Account(context.Background())
		if err != nil {
			return err
		}
	case transport.TransactionPath:
		var err error
		af.transactions, err = af.client.TransactionHistory(context.Background())
		if err != nil {
			return err
		}
	case transport.TransferPath:
		var err error
		var (
			from     = ""
			to       = ""
			currency = ""
			amount   = 0.00
		)
		for k, v := range af.Args {
			switch k {
			case "amount":
				value, err := strconv.ParseFloat(v, 64)
				if err != nil {
					_ = level.Info(af.logger).Log("key", k, "value", v, "err", err.Error())
					os.Exit(1)
				}
				amount = value
			case "from":
				from = v
			case "to":
				to = v
			case "currency":
				currency = v
			}
		}
		_, err = af.client.Transfer(context.Background(), from, to, amount, currency)
		if err == nil {
			af.lastError = nil
			return nil
		}
		switch err.Error() {
		case service.ErrRequiredArgumentMissing.Error(), repository.ErrPayerNotFound.Error():
			af.lastError = err
		case repository.ErrPayeeNotFound.Error(), service.ErrDifferentCurrency.Error():
			af.lastError = err
		case service.ErrSelfTransfer.Error(), service.ErrInsufficientFunds.Error():
			af.lastError = err
		case service.ErrWrongCurrency.Error(), service.ErrTransactionFailed.Error():
			af.lastError = err
		default:
			return err
		}
	default:
		return godog.ErrPending
	}
	return nil
}

func (af *apiFeature) outputJSONShouldHaveFieldWithFollowingData(fieldName string, recordList *gherkin.DataTable) error {
	head := recordList.Rows[0].Cells
	if fieldName == "accounts" && len(recordList.Rows)-1 != len(af.accounts) {
		return fmt.Errorf("different length of accounts: %d != %d", len(recordList.Rows)-1, len(af.accounts))
	}
	for i := 1; i < len(recordList.Rows); i++ {
		for n, cell := range recordList.Rows[i].Cells {
			switch head[n].Value {
			case "id":
				if cell.Value != af.accounts[i-1].UserID {
					return fmt.Errorf("User Ids are different: %s != %s", cell.Value, af.accounts[i-1].UserID)
				}
			case "balance", "amount":
				value, err := strconv.ParseFloat(cell.Value, 64)
				if err != nil {
					_ = level.Info(af.logger).Log("column", head[n].Value, "value", cell.Value, "err", err.Error())
					os.Exit(1)
				}
				if head[n].Value == "balance" && value != af.accounts[i-1].Balance {
					return fmt.Errorf("Balances are different: %f != %f", value, af.accounts[i-1].Balance)
				}
				if head[n].Value == "amount" && value != af.transactions[i-1].(map[string]interface{})["amount"] {
					return fmt.Errorf("Amounts are different: %f != %f", value, af.transactions[i-1].(map[string]interface{})["amount"])
				}
			case "direction":
				if cell.Value != af.transactions[i-1].(map[string]interface{})["direction"] {
					return fmt.Errorf("Directions are different: %s != %s", cell.Value, af.transactions[i-1].(map[string]interface{})["direction"])
				}
			case "error":
				if cell.Value != af.transactions[i-1].(map[string]interface{})["error"] {
					return fmt.Errorf("Errors are different: %s != %s", cell.Value, af.transactions[i-1].(map[string]interface{})["error"])
				}
			case "payee":
				var value string
				if af.transactions[i-1].(map[string]interface{})["direction"] == repository.DirectionIncoming {
					value = af.transactions[i-1].(map[string]interface{})["account"].(string)
				} else {
					value = af.transactions[i-1].(map[string]interface{})["to_account"].(string)
				}
				if cell.Value != value {
					return fmt.Errorf("Payees are different: %s != %s", cell.Value, value)
				}
			case "payer":
				var value string
				if af.transactions[i-1].(map[string]interface{})["direction"] == repository.DirectionIncoming {
					value = af.transactions[i-1].(map[string]interface{})["from_account"].(string)
				} else {
					value = af.transactions[i-1].(map[string]interface{})["account"].(string)
				}
				if cell.Value != value {
					return fmt.Errorf("Payers are different: %s != %s", cell.Value, value)
				}
			case "currency":
				if fieldName == "accounts" {
					if cell.Value != af.accounts[i-1].Currency {
						return fmt.Errorf("Currencies are different: %s != %s", cell.Value, af.accounts[i-1].Currency)
					}
				}
			}
		}
	}
	return nil
}

func (af *apiFeature) requestArgumentsAre(requstsArgs *gherkin.DataTable) error {
	head := requstsArgs.Rows[0].Cells
	for i := 1; i < len(requstsArgs.Rows); i++ {
		var key, value string
		for n, cell := range requstsArgs.Rows[i].Cells {
			switch head[n].Value {
			case "key":
				key = cell.Value
			case "value":
				value = cell.Value
			}
		}
		if key != "" {
			af.Args[key] = value
		}
	}
	return nil
}

func (af *apiFeature) iShouldGetError(errString string) error {
	if errString == "" && af.lastError == nil {
		return nil
	}
	if errString == af.lastError.Error() {
		return nil
	}
	return fmt.Errorf("Error should be %s, but got %v", errString, af.lastError)
}

func (af *apiFeature) andTableShouldContainFollowingData(tableName string, recordList *gherkin.DataTable) error {
	fields := make([]string, 0)
	head := recordList.Rows[0].Cells
	for _, cell := range head {
		fields = append(fields, cell.Value)
	}
	query := "SELECT " + strings.Join(fields, ", ") + " FROM " + tableName
	rows, err := af.db.Query(query)
	if err != nil {
		_ = level.Error(af.logger).Log("table", tableName, "err", err)
		os.Exit(1)
	}
	defer rows.Close()

	rowsCount := 0
	colsCount := len(fields)
	for rows.Next() {
		cells := make([]interface{}, colsCount)
		cellPtrs := make([]interface{}, colsCount)
		for i := range fields {
			cellPtrs[i] = &cells[i]
		}
		err := rows.Scan(cellPtrs...)
		if err != nil {
			return fmt.Errorf("failed to parse row in %s: %v", tableName, err)
		}
		found := false
		for i := 1; i < len(recordList.Rows); i++ {
			matched := 0
			for n, cell := range recordList.Rows[i].Cells {
				switch cells[n].(type) {
				case []uint8:
					cells[n] = string([]byte(cells[n].([]uint8)[:]))
				}
				if cells[n].(string) == cell.Value {
					matched++
				} else {
					break
				}
			}
			if matched == colsCount {
				found = true
				break
			}
		}
		if found != true {
			return fmt.Errorf("Record %v was not found in table: %s", cells, tableName)
		}
		rowsCount++
	}
	if len(recordList.Rows)-1 != rowsCount {
		return fmt.Errorf("different length of %s: %d != %d", tableName, len(recordList.Rows)-1, rowsCount)
	}

	return nil
}

// FeatureContext - init and route steps
func FeatureContext(s *godog.Suite) {
	api := &apiFeature{}
	api.init()
	s.Step(`^the following "([^"]*)" list exist:$`, api.theFollowingListExist)
	s.Step(`^I send "([^"]*)" request to "([^"]*)"$`, api.iSendRequestTo)
	s.Step(`^output json should have "([^"]*)" field with following data:$`, api.outputJSONShouldHaveFieldWithFollowingData)
	s.Step(`^request arguments are:$`, api.requestArgumentsAre)
	s.Step(`^I should get error "([^"]*)"$`, api.iShouldGetError)
	s.Step(`^and table "([^"]*)" should contain following data:$`, api.andTableShouldContainFollowingData)
	s.BeforeScenario(func(interface{}) {
		api.TruncateDB()
		for k := range api.Args {
			delete(api.Args, k)
		}
	})
	s.AfterSuite(api.deInit)
}

// TestMain is entry point
func TestMain(m *testing.M) {
	var opt = godog.Options{
		Paths: []string{"features"},
	}
	godog.BindFlags("godog.", flag.CommandLine, &opt)
	flag.Parse()
	opt.Paths = flag.Args()

	status := godog.RunWithOptions("godogs", func(s *godog.Suite) {
		FeatureContext(s)
	}, opt)

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func envString(env, fallback string) string {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	return e
}

func envInt(env string, fallback int) int {
	e := os.Getenv(env)
	if e == "" {
		return fallback
	}
	v, err := strconv.Atoi(e)
	if err != nil {
		return fallback
	}
	return v
}
