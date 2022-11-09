package payments

type Account struct {
	Reference string      `json:"reference" bson:"reference"`
	Provider  string      `json:"provider" bson:"provider"`
	Type      AccountType `json:"type" bson:"type"`
}

type AccountType string

const (
	AccountTypeSource  AccountType = "source"
	AccountTypeTarget  AccountType = "target"
	AccountTypeUnknown AccountType = "unknown"
)
