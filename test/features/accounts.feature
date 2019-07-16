Feature: accounts
  Payment System should be able to list all accounts

  Scenario: list accounts when on server there are a few accounts
    Given the following "account" list exist:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    When I send "GET" request to "/v1/accounts"
    And output json should have "accounts" field with following data:
      | id       | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |

  Scenario: list accounts when on server there are no accounts
    Given the following "account" list exist:
      | user_id  | balance | currency |
    When I send "GET" request to "/v1/accounts"
    And output json should have "accounts" field with following data:
      | id       | balance | currency |
