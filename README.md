# Payment system

Demo payment system with HTTP API. Its features:
- see list of available accounts
- see all payments (transactions)
- make payment (transfer) from account to account in the same currency

API documentation could be found in docs/api.md and docs/swagger.yml

## Running locally

To run project locally with docker-compose use:

    make up

This command will create `.env` file from `.env.dist`
and start Docker cluster with following components:
   - Postgresql Database
   - Nginx (load balancer, and swagger docs) http://127.0.0.1:8080 by default
   - Two backends http://127.0.0.1:8081 and http://127.0.0.1:8082

## Running unit tests

To run unit tests

    make test

This command will install `dep` in `$GOPATH/bin/`
    
## Code linting

To check code

    make lint

This command will install `golangci-lint` in `$GOPATH/bin/`
    
## Running acceptance tests

To run acceptance tests (on running instance):

    make API_HOST=127.0.0.1 HTTP_PORT=8080 at
    
**WARNING**: this command will reset DB (purge all data)
