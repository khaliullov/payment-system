# Payment System API

## Description

API of Payment System uses HTTP protocol.
Requests or responses are in JSON format.
API versioning is done via HTTP path prefix.
Current prefix is /v1.

## Methods

### List accounts

To list all accounts:

    GET /v1/accounts

This will return JSON with following fields:
- "success": (boolean) - True on success
- "accounts": (list) with accounts:
  * "id": (string) account name
  * "balance": (float) balance of account
  * "currency": (string) currency of account
- "error": (string) absent if no error occurred, otherwise 
this field contains error description.

Example response:

    {
      "success": true,
      "accounts": [
        {
          "id": "alice456",
          "balance": 90,
          "currency": "USD"
        },
        {
          "id": "bob123",
          "balance": 10.01,
          "currency": "USD"
        }
      ]
    }

### List payments

List all payments:

    GET /v1/payments

This will return all transaction history, including
failed requests (insufficient funds, etc.) with following fields:
- "success": boolean - True on success
- "payments": (list) with transaction history:
  * "direction": (string) outgoing/incoming
  * "account"/"from_account": (string) payer account (depending on direction)
  * "account"/"to_account": (string) payee account (depending on direction)
  * "amount": (float) transfer amount
  * "error": (string) empty if no error occurred during transferring.
- "error": (string) absent if no error occurred, otherwise 
this field contains error description.

Example response:

    {
      "success": true,
      "payments": [
        {
          "direction": "outgoing",
          "account": "alice456",
          "to_account": "bob123",
          "amount": 10,
          "error": "Insufficient funds"
        },
        {
          "direction": "incoming",
          "from_account": "bob123",
          "account": "alice456",
          "amount": 10,
          "error": ""
        },
        {
          "direction": "outgoing",
          "account": "bob123",
          "to_account": "alice456",
          "amount": 10,
          "error": ""
        }
      ]
    }

### Make payment

To transfer money from account to account:

    POST /v1/transfer
    Content-Type: application/json
    
    {
      "from": "string",
      "to": "string",
      "amount": 0
    }

This will transfer money between two accounts and return
JSON with following fields:
- "success": boolean - True on success
- "error": (string) absent if no error occurred, otherwise 
this field contains error description.

Request consist of the following fields:
- "from": (string) payer account
- "to": (string) payee account
- "amount": (float) transfer amount

Example request:

    POST /v1/transfer
    Content-Type: application/json
    
    {
      "to": "alice456",
      "from": "bob123",
      "amount": 10,
      "currency": "USD"
    }

Example response:

    {
      "success": true
    }

