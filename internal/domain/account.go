package domain

// AccountType is the kind of brokerage account.
type AccountType string

// Brokerage is a standard cash/securities brokerage account.
const Brokerage AccountType = "BROKERAGE"

// Account is a brokerage account belonging to the authenticated user.
type Account struct {
	// No is the human-readable account number, shown in the UI.
	No string
	// Seq is the account's identifier key used in API calls (e.g. orders).
	Seq int64
	// Type is the account kind.
	Type AccountType
}
