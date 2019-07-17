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
    And and table "account" should contain following data:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    And and table "payment" should contain following data:
      | direction | payer    | payee    | amount | error |

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
    And and table "account" should contain following data:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    And and table "payment" should contain following data:
      | direction | payer    | payee    | amount | error |

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
    And and table "account" should contain following data:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    And and table "payment" should contain following data:
      | direction | payer    | payee    | amount | error |

  Scenario: try to make transfer to the same account
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
    And and table "account" should contain following data:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    And and table "payment" should contain following data:
      | direction | payer    | payee    | amount | error |

  Scenario: try to make transfer to with insufficient funds
    Given the following "account" list exist:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    When request arguments are:
      | key    | value    |
      | from   | alice456 |
      | to     | bob123   |
      | amount | 0.02    |
    And I send "POST" request to "/v1/transfer"
    Then I should get error "Insufficient funds"
    And and table "account" should contain following data:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    And and table "payment" should contain following data:
      | direction | payer    | payee    | amount | error              |
      | outgoing  | alice456 | bob123   | 0.02   | Insufficient funds |

  Scenario: try to make transfer to with different currency
    Given the following "account" list exist:
      | user_id  | balance | currency |
      | alice456 | 0.01    | RUB      |
      | bob123   | 100.00  | USD      |
    When request arguments are:
      | key    | value    |
      | from   | bob123   |
      | to     | alice456 |
      | amount | 10       |
    And I send "POST" request to "/v1/transfer"
    Then I should get error "Different currency"
    And and table "account" should contain following data:
      | user_id  | balance | currency |
      | alice456 | 0.01    | RUB      |
      | bob123   | 100.00  | USD      |
    And and table "payment" should contain following data:
      | direction | payer    | payee    | amount | error              |
      | outgoing  | bob123   | alice456 | 10.00  | Different currency |

  Scenario: try to make transfer to with wrong currency
    Given the following "account" list exist:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    When request arguments are:
      | key      | value    |
      | from     | bob123   |
      | to       | alice456 |
      | amount   | 10       |
      | currency | RUB      |
    And I send "POST" request to "/v1/transfer"
    Then I should get error "Wrong currency"
    And and table "account" should contain following data:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    And and table "payment" should contain following data:
      | direction | payer    | payee    | amount | error          |
      | outgoing  | bob123   | alice456 | 10.00  | Wrong currency |

  Scenario: try to make successful transfer
    Given the following "account" list exist:
      | user_id  | balance | currency |
      | alice456 | 0.01    | USD      |
      | bob123   | 100.00  | USD      |
    When request arguments are:
      | key      | value    |
      | from     | bob123   |
      | to       | alice456 |
      | amount   | 10       |
      | currency | USD      |
    And I send "POST" request to "/v1/transfer"
    Then I should get error ""
    And and table "account" should contain following data:
      | user_id  | balance | currency |
      | alice456 | 10.01   | USD      |
      | bob123   | 90.00   | USD      |
    And and table "payment" should contain following data:
      | direction | payer    | payee    | amount | error |
      | outgoing  | bob123   | alice456 | 10.00  |       |
      | incoming  | bob123   | alice456 | 10.00  |       |
