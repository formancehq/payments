package mappers

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
)

func TestPaymentMetadata(t *testing.T) {
	t.Parallel()
	tx := client.UserTransaction{
		Type:    TxTypeWithdrawal,
		Fee:     "0.005",
		OrderID: json.Number("0"),
	}
	got := PaymentMetadata(tx)
	if got[MetadataKeyType] != TxTypeWithdrawal {
		t.Errorf("missing type: %v", got)
	}
	if got[MetadataKeySource] != PaymentSourceUserTransactions {
		t.Errorf("user_transactions metadata must carry source=user_transactions, got %q", got[MetadataKeySource])
	}
	if got[MetadataKeyFee] != "0.005" {
		t.Errorf("missing fee: %v", got)
	}
	if _, ok := got[MetadataKeyOrderID]; ok {
		t.Errorf("order_id 0 should not surface: %v", got)
	}

	tx2 := client.UserTransaction{Type: "0", Fee: "0", OrderID: json.Number("12345")}
	got2 := PaymentMetadata(tx2)
	if _, ok := got2[MetadataKeyFee]; ok {
		t.Errorf("zero fee should not surface: %v", got2)
	}
	if got2[MetadataKeyOrderID] != "12345" {
		t.Errorf("missing order_id: %v", got2)
	}
}

func TestConversionMetadata(t *testing.T) {
	t.Parallel()
	tx := client.UserTransaction{Type: TxTypeBuySell, Fee: "0.000000"}
	got := ConversionMetadata(tx, "eur_usdc", "0.86047")
	if got[MetadataKeyType] != TxTypeBuySell {
		t.Errorf("missing type: %v", got)
	}
	if got[MetadataKeyCurrencyPair] != "eur_usdc" {
		t.Errorf("missing currency_pair: %v", got)
	}
	if got[MetadataKeyRate] != "0.86047" {
		t.Errorf("missing rate: %v", got)
	}
	if _, ok := got[MetadataKeyFee]; ok {
		t.Errorf("zero fee should be omitted: %v", got)
	}
}

func TestMetadataKeysAreNamespaced(t *testing.T) {
	t.Parallel()
	// Guard against accidental drift from the com.bitstamp.spec/* prefix.
	for _, k := range []string{
		MetadataKeyType, MetadataKeyFee, MetadataKeyOrderID,
		MetadataKeyCurrencyPair, MetadataKeyClientOrderID, MetadataKeyRate,
		MetadataKeySource, MetadataKeyTransferPairID, MetadataKeyTransferDirection,
		MetadataKeyCounterpartySubAccountID, MetadataKeyCounterpartySubAccountName,
		MetadataKeyNetwork, MetadataKeyTxID, MetadataKeyDestinationAddress,
		MetadataKeyPendingReason, MetadataKeyBankTransactionID,
		MetadataKeyNetworks, MetadataKeyWithdrawalFees, MetadataKeyTradableMarkets,
		MetadataKeyFeeTierMaker, MetadataKeyFeeTierTaker, MetadataKeyMinOrderValue,
		MetadataKeyMarketSymbol, MetadataKeyOrderDatetimeSecs,
	} {
		if !startsWith(k, MetadataPrefix) {
			t.Errorf("%q not under %s", k, MetadataPrefix)
		}
	}
}

func startsWith(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}

func TestCryptoTransactionMetadata(t *testing.T) {
	t.Parallel()
	got := CryptoTransactionMetadata(CryptoKindDeposit, "bitcoin", "tx-1", "addr-1", "ADDRESS_VERIFICATION_NEEDED")
	if got[MetadataKeySource] != PaymentSourceCryptoTransactions {
		t.Errorf("source = %q", got[MetadataKeySource])
	}
	if got[MetadataKeyType] != CryptoKindDeposit {
		t.Errorf("kind = %q", got[MetadataKeyType])
	}
	if got[MetadataKeyNetwork] != "bitcoin" || got[MetadataKeyTxID] != "tx-1" ||
		got[MetadataKeyDestinationAddress] != "addr-1" || got[MetadataKeyPendingReason] != "ADDRESS_VERIFICATION_NEEDED" {
		t.Errorf("missing fields: %+v", got)
	}

	// Empty optionals must be omitted, not empty-stringed.
	got = CryptoTransactionMetadata(CryptoKindWithdrawal, "", "", "", "")
	for _, k := range []string{MetadataKeyNetwork, MetadataKeyTxID, MetadataKeyDestinationAddress, MetadataKeyPendingReason} {
		if _, present := got[k]; present {
			t.Errorf("empty %s must be omitted, got %v", k, got[k])
		}
	}
}

func TestWithdrawalRequestMetadata(t *testing.T) {
	t.Parallel()
	got := WithdrawalRequestMetadata("4", "bitcoin", "addr-1", "tx-1", "bank-1")
	if got[MetadataKeySource] != PaymentSourceWithdrawalRequests {
		t.Errorf("source = %q", got[MetadataKeySource])
	}
	if got[MetadataKeyType] != "4" {
		t.Errorf("withdrawal type = %q", got[MetadataKeyType])
	}
	if got[MetadataKeyBankTransactionID] != "bank-1" {
		t.Errorf("bank tx id = %q", got[MetadataKeyBankTransactionID])
	}

	got = WithdrawalRequestMetadata("0", "", "", "", "")
	for _, k := range []string{MetadataKeyNetwork, MetadataKeyDestinationAddress, MetadataKeyTxID, MetadataKeyBankTransactionID} {
		if _, present := got[k]; present {
			t.Errorf("empty %s must be omitted", k)
		}
	}
}

func TestTransferPairMetadata(t *testing.T) {
	t.Parallel()
	got := TransferPairMetadata(458254264, TransferDirectionOutgoing, "sub-acct-A", "Trading Sub")
	if got[MetadataKeyTransferPairID] != "458254264" {
		t.Errorf("pair id = %q", got[MetadataKeyTransferPairID])
	}
	if got[MetadataKeyTransferDirection] != TransferDirectionOutgoing {
		t.Errorf("direction = %q", got[MetadataKeyTransferDirection])
	}
	if got[MetadataKeyCounterpartySubAccountID] != "sub-acct-A" || got[MetadataKeyCounterpartySubAccountName] != "Trading Sub" {
		t.Errorf("counterparty fields missing: %+v", got)
	}

	// Empty counterparty must be omitted (sub_accounts endpoint is 404).
	got = TransferPairMetadata(42, TransferDirectionIncoming, "", "")
	if _, present := got[MetadataKeyCounterpartySubAccountID]; present {
		t.Errorf("empty counterparty id must be omitted")
	}
	if _, present := got[MetadataKeyCounterpartySubAccountName]; present {
		t.Errorf("empty counterparty name must be omitted")
	}
}

func TestMergeMetadata(t *testing.T) {
	t.Parallel()
	a := map[string]string{"k1": "a1", "k2": "a2"}
	b := map[string]string{"k2": "b2", "k3": "b3"}

	got := MergeMetadata(a, b)
	if got["k1"] != "a1" || got["k2"] != "b2" || got["k3"] != "b3" {
		t.Errorf("merge order broken: %+v", got)
	}
	// Inputs must not mutate.
	if a["k2"] != "a2" {
		t.Error("MergeMetadata must not mutate input maps")
	}
}
