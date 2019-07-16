CREATE TABLE public.account
(
  user_id  VARCHAR(40) PRIMARY KEY,
  currency VARCHAR(3)     NOT NULL DEFAULT 'USD',
  balance  NUMERIC(15, 2) NOT NULL DEFAULT 0
    CONSTRAINT positive_balance CHECK (balance >= 0)
);

CREATE TABLE public.payment
(
  txn_id    BIGSERIAL PRIMARY KEY,
  direction VARCHAR(9)     NOT NULL,
  date      TIMESTAMP               DEFAULT current_timestamp,
  payer     VARCHAR(40) REFERENCES account (user_id) ON DELETE CASCADE,
  payee     VARCHAR(40) REFERENCES account (user_id) ON DELETE CASCADE,
  amount    NUMERIC(15, 2) NOT NULL,
  currency  VARCHAR(3)     NOT NULL,
  error     VARCHAR(200)   NOT NULL DEFAULT ''
);
