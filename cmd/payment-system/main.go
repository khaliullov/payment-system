package main

import (
	"database/sql"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"text/tabwriter"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	_ "github.com/lib/pq"
	"github.com/oklog/oklog/pkg/group"

	"github.com/khaliullov/payment-system/pkg/endpoint"
	"github.com/khaliullov/payment-system/pkg/repository"
	"github.com/khaliullov/payment-system/pkg/service"
	"github.com/khaliullov/payment-system/pkg/transport"
)

func main() {
	fs := flag.NewFlagSet("payment-system", flag.ExitOnError)
	var (
		httpAddr   = fs.String("http-addr", ":"+strconv.Itoa(envInt("HTTP_PORT", 8000)), "HTTP listen address")
		dbHost     = fs.String("db-host", envString("DB_HOST", "localhost"), "postgresql host")
		dbPort     = fs.Int("db-port", envInt("DB_PORT", 5432), "postgresql port")
		dbName     = fs.String("db-name", envString("DB_NAME", "psdb"), "postgresql database name")
		dbUser     = fs.String("db-user", envString("DB_USER", "postgres"), "postgresql user")
		dbPassword = fs.String("db-password", envString("DB_PASSWORD", "postgres"), "postgresql password")
	)
	fs.Usage = usageFor(fs, os.Args[0]+" [flags]")
	_ = fs.Parse(os.Args[1:])

	// Logging domain.
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = level.NewFilter(logger, level.AllowDebug())
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	_ = level.Info(logger).Log("msg", "payment system started")
	defer func() {
		_ = level.Info(logger).Log("msg", "payment system ended")
	}()

	var db *sql.DB
	{
		var err error
		// Connect to the "payment-system" database
		DSN := &url.URL{
			Scheme:   "postgresql",
			RawQuery: "sslmode=disable",
			Host:     *dbHost + ":" + strconv.Itoa(*dbPort),
			Path:     *dbName,
			User:     url.UserPassword(*dbUser, *dbPassword),
		}
		db, err = sql.Open("postgres", DSN.String())
		if err != nil {
			_ = level.Error(logger).Log("db", err)
			os.Exit(1)
		}
	}
	// Build the layers of the service "onion" from the inside out. First, the
	// business logic service; then, the set of endpoints that wrap the service;
	// and finally, a series of concrete transport adapters. The adapters, like
	// the HTTP handler or the gRPC server, are the bridge between Go kit and
	// the interfaces that the transports expect. Note that we're not binding
	// them to ports or anything yet; we'll do that next.
	var (
		repository  = repository.New(db, logger)
		service     = service.New(repository, logger)
		endpoints   = endpoint.New(service, logger)
		httpHandler = transport.NewHTTPHandler(endpoints, logger)
	)

	// Now we're to the part of the func main where we want to start actually
	// running things, like servers bound to listeners to receive connections.
	//
	// The method is the same for each component: add a new actor to the group
	// struct, which is a combination of 2 anonymous functions: the first
	// function actually runs the component, and the second function should
	// interrupt the first function and cause it to return. It's in these
	// functions that we actually bind the Go kit server/handler structs to the
	// concrete transports and run them.
	//
	// Putting each component into its own block is mostly for aesthetics: it
	// clearly demarcates the scope in which each listener/socket may be used.
	var g group.Group
	{
		// The HTTP listener mounts the Go kit HTTP handler we created.
		httpListener, err := net.Listen("tcp", *httpAddr)
		if err != nil {
			_ = level.Error(logger).Log("transport", "HTTP", "during", "Listen", "err", err)
			os.Exit(1)
		}
		g.Add(func() error {
			_ = level.Info(logger).Log("transport", "HTTP", "addr", *httpAddr)
			return http.Serve(httpListener, httpHandler)
		}, func(error) {
			httpListener.Close()
		})
	}
	{
		// This function just sits and waits for ctrl-C.
		cancelInterrupt := make(chan struct{})
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				return fmt.Errorf("received signal %s", sig)
			case <-cancelInterrupt:
				return nil
			}
		}, func(error) {
			close(cancelInterrupt)
		})
	}
	// Run!
	_ = level.Error(logger).Log("exit", g.Run())
}

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "USAGE\n")
		fmt.Fprintf(os.Stderr, "  %s\n", short)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(w, "\t-%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		w.Flush()
		fmt.Fprintf(os.Stderr, "\n")
	}
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
