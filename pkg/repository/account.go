package repository

// Account represents user account of payment system.
type Account struct {
	UserID   string  `json:"id"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}
