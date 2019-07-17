Feature: payments
  Payment System should be able to list all transactions

  Scenario: list payments when on server there are a few accounts and transactions
    Given the following "account" list exist:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    And the following "payment" list exist:
      | direction | payer    | payee    | amount | currency | error              |
      | outgoing  | alice456 | bob123   | 100.00 | USD      |                    |
      | incoming  | alice456 | bob123   | 100.00 | USD      |                    |
      | outgoing  | bob123   | alice456 | 100.00 | USD      |                    |
      | incoming  | bob123   | alice456 | 100.00 | USD      |                    |
      | outgoing  | alice456 | bob123   | 100.00 | USD      | Insufficient funds |
    When I send "GET" request to "/v1/payments"
    Then output json should have "payments" field with following data:
      | direction | payer    | payee    | amount | error              |
      | outgoing  | alice456 | bob123   | 100.00 |                    |
      | incoming  | alice456 | bob123   | 100.00 |                    |
      | outgoing  | bob123   | alice456 | 100.00 |                    |
      | incoming  | bob123   | alice456 | 100.00 |                    |
      | outgoing  | alice456 | bob123   | 100.00 | Insufficient funds |
