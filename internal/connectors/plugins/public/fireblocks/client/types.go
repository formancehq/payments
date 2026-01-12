package client

// VaultAccountsPagedResponse represents a paginated list of vault accounts
type VaultAccountsPagedResponse struct {
	Accounts []VaultAccount `json:"accounts"`
	Paging   Paging         `json:"paging"`
}

// Paging contains pagination information
type Paging struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

// VaultAccount represents a Fireblocks vault account
type VaultAccount struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	HiddenOnUI    bool         `json:"hiddenOnUI"`
	CustomerRefID string       `json:"customerRefId,omitempty"`
	AutoFuel      bool         `json:"autoFuel"`
	Assets        []VaultAsset `json:"assets,omitempty"`
}

// VaultAsset represents an asset within a vault account
type VaultAsset struct {
	ID                   string `json:"id"`
	Total                string `json:"total"`
	Balance              string `json:"balance,omitempty"`
	Available            string `json:"available"`
	Pending              string `json:"pending,omitempty"`
	Frozen               string `json:"frozen,omitempty"`
	LockedAmount         string `json:"lockedAmount,omitempty"`
	Staked               string `json:"staked,omitempty"`
	TotalStakedCPU       string `json:"totalStakedCPU,omitempty"`
	TotalStakedNetwork   string `json:"totalStakedNetwork,omitempty"`
	SelfStakedCPU        string `json:"selfStakedCPU,omitempty"`
	SelfStakedNetwork    string `json:"selfStakedNetwork,omitempty"`
	PendingRefundCPU     string `json:"pendingRefundCPU,omitempty"`
	PendingRefundNetwork string `json:"pendingRefundNetwork,omitempty"`
	BlockHeight          string `json:"blockHeight,omitempty"`
	BlockHash            string `json:"blockHash,omitempty"`
}

// ExternalWallet represents an external wallet (whitelisted address)
type ExternalWallet struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	CustomerRefID string               `json:"customerRefId,omitempty"`
	Assets        []ExternalWalletAsset `json:"assets,omitempty"`
}

// ExternalWalletAsset represents an asset within an external wallet
type ExternalWalletAsset struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	Address        string `json:"address"`
	Tag            string `json:"tag,omitempty"`
	ActivationTime string `json:"activationTime,omitempty"`
}

// InternalWallet represents an internal wallet
type InternalWallet struct {
	ID            string               `json:"id"`
	Name          string               `json:"name"`
	CustomerRefID string               `json:"customerRefId,omitempty"`
	Assets        []InternalWalletAsset `json:"assets,omitempty"`
}

// InternalWalletAsset represents an asset within an internal wallet
type InternalWalletAsset struct {
	ID             string `json:"id"`
	Status         string `json:"status"`
	Address        string `json:"address"`
	Tag            string `json:"tag,omitempty"`
	ActivationTime string `json:"activationTime,omitempty"`
}

// TransactionResponse represents a Fireblocks transaction
type TransactionResponse struct {
	ID                            string                `json:"id"`
	ExternalTxID                  string                `json:"externalTxId,omitempty"`
	Status                        string                `json:"status"`
	SubStatus                     string                `json:"subStatus,omitempty"`
	TxHash                        string                `json:"txHash,omitempty"`
	Operation                     string                `json:"operation"`
	Note                          string                `json:"note,omitempty"`
	AssetID                       string                `json:"assetId"`
	Source                        SourceDestination     `json:"source"`
	SourceAddress                 string                `json:"sourceAddress,omitempty"`
	Destination                   SourceDestination     `json:"destination"`
	Destinations                  []DestinationAmount   `json:"destinations,omitempty"`
	DestinationAddress            string                `json:"destinationAddress,omitempty"`
	DestinationAddressDescription string                `json:"destinationAddressDescription,omitempty"`
	DestinationTag                string                `json:"destinationTag,omitempty"`
	AmountInfo                    AmountInfo            `json:"amountInfo"`
	FeeInfo                       FeeInfo               `json:"feeInfo"`
	FeeCurrency                   string                `json:"feeCurrency,omitempty"`
	NetworkRecords                []NetworkRecord       `json:"networkRecords,omitempty"`
	CreatedAt                     int64                 `json:"createdAt"`
	LastUpdated                   int64                 `json:"lastUpdated"`
	CreatedBy                     string                `json:"createdBy,omitempty"`
	SignedBy                      []string              `json:"signedBy,omitempty"`
	RejectedBy                    string                `json:"rejectedBy,omitempty"`
	ExchangeTxID                  string                `json:"exchangeTxId,omitempty"`
	CustomerRefID                 string                `json:"customerRefId,omitempty"`
	NumOfConfirmations            int                   `json:"numOfConfirmations,omitempty"`
	BlockInfo                     *BlockInfo            `json:"blockInfo,omitempty"`
	Index                         int                   `json:"index,omitempty"`
	RewardInfo                    *RewardInfo           `json:"rewardInfo,omitempty"`
	SystemMessages                []SystemMessage       `json:"systemMessages,omitempty"`
	AddressType                   string                `json:"addressType,omitempty"`
	RequestedAmount               float64               `json:"requestedAmount,omitempty"`
	Amount                        float64               `json:"amount,omitempty"`
	NetAmount                     float64               `json:"netAmount,omitempty"`
	AmountUSD                     float64               `json:"amountUSD,omitempty"`
	ServiceFee                    float64               `json:"serviceFee,omitempty"`
	NetworkFee                    float64               `json:"networkFee,omitempty"`
}

// SourceDestination represents the source or destination of a transaction
type SourceDestination struct {
	Type    string `json:"type"`
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	SubType string `json:"subType,omitempty"`
}

// DestinationAmount represents a destination with amount
type DestinationAmount struct {
	Destination SourceDestination `json:"destination"`
	Amount      float64           `json:"amount"`
	AmountUSD   float64           `json:"amountUSD,omitempty"`
}

// AmountInfo contains amount details
type AmountInfo struct {
	Amount          string `json:"amount,omitempty"`
	RequestedAmount string `json:"requestedAmount,omitempty"`
	NetAmount       string `json:"netAmount,omitempty"`
	AmountUSD       string `json:"amountUSD,omitempty"`
}

// FeeInfo contains fee details
type FeeInfo struct {
	NetworkFee string `json:"networkFee,omitempty"`
	ServiceFee string `json:"serviceFee,omitempty"`
	GasPrice   string `json:"gasPrice,omitempty"`
}

// NetworkRecord represents network-specific transaction data
type NetworkRecord struct {
	Source             SourceDestination `json:"source"`
	Destination        SourceDestination `json:"destination"`
	TxHash             string            `json:"txHash,omitempty"`
	NetworkFee         string            `json:"networkFee,omitempty"`
	AssetID            string            `json:"assetId"`
	NetAmount          string            `json:"netAmount,omitempty"`
	Status             string            `json:"status"`
	Type               string            `json:"type,omitempty"`
	DestinationAddress string            `json:"destinationAddress,omitempty"`
	SourceAddress      string            `json:"sourceAddress,omitempty"`
}

// BlockInfo contains blockchain block information
type BlockInfo struct {
	BlockHeight string `json:"blockHeight,omitempty"`
	BlockHash   string `json:"blockHash,omitempty"`
}

// RewardInfo contains staking reward information
type RewardInfo struct {
	SrcRewards  string `json:"srcRewards,omitempty"`
	DestRewards string `json:"destRewards,omitempty"`
}

// SystemMessage represents a system message
type SystemMessage struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// AssetTypeResponse represents supported asset information
type AssetTypeResponse struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	ContractAddress string `json:"contractAddress,omitempty"`
	NativeAsset     string `json:"nativeAsset,omitempty"`
	Decimals        int    `json:"decimals,omitempty"`
}

// Transaction status constants
const (
	TxStatusSubmitted           = "SUBMITTED"
	TxStatusPendingScreening    = "PENDING_SCREENING"
	TxStatusPendingAuthorization = "PENDING_AUTHORIZATION"
	TxStatusQueued              = "QUEUED"
	TxStatusPendingSignature    = "PENDING_SIGNATURE"
	TxStatusPending3rdParty     = "PENDING_3RD_PARTY_MANUAL_APPROVAL"
	TxStatusPending3rdPartyOther = "PENDING_3RD_PARTY"
	TxStatusBroadcasting        = "BROADCASTING"
	TxStatusConfirming          = "CONFIRMING"
	TxStatusCompleted           = "COMPLETED"
	TxStatusCancelling          = "CANCELLING"
	TxStatusCancelled           = "CANCELLED"
	TxStatusBlocked             = "BLOCKED"
	TxStatusRejected            = "REJECTED"
	TxStatusFailed              = "FAILED"
)

// Peer type constants
const (
	PeerTypeVaultAccount      = "VAULT_ACCOUNT"
	PeerTypeExchangeAccount   = "EXCHANGE_ACCOUNT"
	PeerTypeInternalWallet    = "INTERNAL_WALLET"
	PeerTypeExternalWallet    = "EXTERNAL_WALLET"
	PeerTypeOneTimeAddress    = "ONE_TIME_ADDRESS"
	PeerTypeNetworkConnection = "NETWORK_CONNECTION"
	PeerTypeFiatAccount       = "FIAT_ACCOUNT"
	PeerTypeCompound          = "COMPOUND"
)

// Fee level constants
const (
	FeeLevelHigh   = "HIGH"
	FeeLevelMedium = "MEDIUM"
	FeeLevelLow    = "LOW"
)
