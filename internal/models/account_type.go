package models

type AccountType string

const (
	ACCOUNT_TYPE_UNKNOWN AccountType = "UNKNOWN"
	// Internal accounts refers to user's digital e-wallets. It serves as a
	// secure storage for funds within the payments provider environment.
	ACCOUNT_TYPE_INTERNAL AccountType = "INTERNAL"
	// External accounts represents actual bank accounts of the user.
	ACCOUNT_TYPE_EXTERNAL AccountType = "EXTERNAL"
)

func AccountTypeFromString(value string) AccountType {
	switch value {
	case string(ACCOUNT_TYPE_INTERNAL):
		return ACCOUNT_TYPE_INTERNAL
	case string(ACCOUNT_TYPE_EXTERNAL):
		return ACCOUNT_TYPE_EXTERNAL
	}
	return ACCOUNT_TYPE_UNKNOWN
}
