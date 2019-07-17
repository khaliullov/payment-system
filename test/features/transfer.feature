Feature: transfer
  Payment System should be able to transfer money from account to account

  Scenario: try to make payment without arguments
    Given the following "account" list exist:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    When request arguments are:
      | key | value |
    And I send "POST" request to "/v1/transfer"
    Then I should get error "Required argument missing or it is incorrect"

  Scenario: try to make transfer from non-existing account
    Given the following "account" list exist:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    When request arguments are:
      | key    | value  |
      | from   | vasya  |
      | to     | bob123 |
      | amount | 10.00  |
    And I send "POST" request to "/v1/transfer"
    Then I should get error "Payer not found"

  Scenario: try to make transfer to non-existing account
    Given the following "account" list exist:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    When request arguments are:
      | key    | value    |
      | from   | alice456 |
      | to     | petya    |
      | amount | 0.01     |
    And I send "POST" request to "/v1/transfer"
    Then I should get error "Payee not found"

  Scenario: try to make to transfer to the same account
    Given the following "account" list exist:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    When request arguments are:
      | key    | value  |
      | from   | bob123 |
      | to     | bob123 |
      | amount | 0.01   |
    And I send "POST" request to "/v1/transfer"
    Then I should get error "Transfer to self"
