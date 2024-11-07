package atlar

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/formancehq/go-libs/v2/metadata"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/models"
	"github.com/get-momo/atlar-v1-go-client/client/transactions"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
)

type transactionsState struct {
	NextToken string `json:"nextToken"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState transactionsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	var payments []models.PSPPayment
	nextToken := oldState.NextToken
	for {
		resp, err := p.client.GetV1Transactions(ctx, nextToken, int64(req.PageSize))
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		payments, err = p.fillPayments(ctx, resp, payments)
		if err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}

		nextToken = resp.Payload.NextToken
		if resp.Payload.NextToken == "" || len(payments) >= req.PageSize {
			break
		}
	}

	// If token is empty, this is perfect as the next polling task will refetch
	// everything ! And that's what we want since Atlar doesn't provide any
	// filters or sorting options.
	newState := transactionsState{
		NextToken: nextToken,
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  nextToken != "",
	}, nil
}

func (p *Plugin) fillPayments(
	ctx context.Context,
	resp *transactions.GetV1TransactionsOK,
	payments []models.PSPPayment,
) ([]models.PSPPayment, error) {
	for _, item := range resp.Payload.Items {
		payment, err := p.transactionToPayment(ctx, item)
		if err != nil {
			return nil, err
		}

		if payment != nil {
			payments = append(payments, *payment)
		}
	}

	return payments, nil
}

func (p *Plugin) transactionToPayment(
	ctx context.Context,
	from *atlar_models.Transaction,
) (*models.PSPPayment, error) {
	if _, ok := supportedCurrenciesWithDecimal[*from.Amount.Currency]; !ok {
		// Discard transactions with unsupported currencies
		return nil, nil
	}

	raw, err := json.Marshal(from)
	if err != nil {
		return nil, err
	}

	paymentType := determinePaymentType(from)

	itemAmount := from.Amount
	amount, err := atlarTransactionAmountToPaymentAbsoluteAmount(*itemAmount.Value)
	if err != nil {
		return nil, err
	}

	createdAt, err := ParseAtlarTimestamp(from.Created)
	if err != nil {
		return nil, err
	}

	accountResponse, err := p.client.GetV1AccountsID(ctx, *from.Account.ID)
	if err != nil {
		return nil, err
	}

	thirdPartyResponse, err := p.client.GetV1BetaThirdPartiesID(ctx, accountResponse.Payload.ThirdPartyID)
	if err != nil {
		return nil, err
	}

	payment := models.PSPPayment{
		Reference: from.ID,
		CreatedAt: createdAt,
		Type:      paymentType,
		Amount:    amount,
		Asset:     currency.FormatAsset(supportedCurrenciesWithDecimal, *from.Amount.Currency),
		Scheme:    determinePaymentScheme(from),
		Status:    determinePaymentStatus(from),
		Metadata:  extractPaymentMetadata(from, accountResponse.Payload, thirdPartyResponse.Payload),
		Raw:       raw,
	}

	if *itemAmount.Value >= 0 {
		// DEBIT
		payment.DestinationAccountReference = from.Account.ID
	} else {
		// CREDIT
		payment.SourceAccountReference = from.Account.ID
	}

	return &payment, nil
}

func determinePaymentType(item *atlar_models.Transaction) models.PaymentType {
	if *item.Amount.Value >= 0 {
		return models.PAYMENT_TYPE_PAYIN
	} else {
		return models.PAYMENT_TYPE_PAYOUT
	}
}

func determinePaymentStatus(item *atlar_models.Transaction) models.PaymentStatus {
	if item.Reconciliation.Status == atlar_models.ReconciliationDetailsStatusEXPECTED {
		// A payment initiated by the owner of the accunt through the Atlar API,
		// which was not yet reconciled with a payment from the statement
		return models.PAYMENT_STATUS_PENDING
	}
	if item.Reconciliation.Status == atlar_models.ReconciliationDetailsStatusBOOKED {
		// A payment comissioned with the bank, which was not yet reconciled with a
		// payment from the statement
		return models.PAYMENT_STATUS_SUCCEEDED
	}
	if item.Reconciliation.Status == atlar_models.ReconciliationDetailsStatusRECONCILED {
		return models.PAYMENT_STATUS_SUCCEEDED
	}
	return models.PAYMENT_STATUS_OTHER
}

func determinePaymentScheme(item *atlar_models.Transaction) models.PaymentScheme {
	// item.Characteristics.BankTransactionCode.Domain
	// item.Characteristics.BankTransactionCode.Family
	// TODO: fees and interest -> models.PaymentSchemeOther with additional info on metadata. Will need example transactions for that

	if *item.Amount.Value > 0 {
		return models.PAYMENT_SCHEME_SEPA_DEBIT
	} else if *item.Amount.Value < 0 {
		return models.PAYMENT_SCHEME_SEPA_CREDIT
	}
	return models.PAYMENT_SCHEME_SEPA
}

func extractPaymentMetadata(transaction *atlar_models.Transaction, account *atlar_models.Account, bank *atlar_models.ThirdParty) metadata.Metadata {
	result := metadata.Metadata{}
	if transaction.Date != "" {
		result = result.Merge(computeMetadata("date", transaction.Date))
	}
	if transaction.ValueDate != "" {
		result = result.Merge(computeMetadata("valueDate", transaction.ValueDate))
	}
	result = result.Merge(computeMetadata("remittanceInformation/type", *transaction.RemittanceInformation.Type))
	result = result.Merge(computeMetadata("remittanceInformation/value", *transaction.RemittanceInformation.Value))
	result = result.Merge(computeMetadata("bank/id", bank.ID))
	result = result.Merge(computeMetadata("bank/name", bank.Name))
	result = result.Merge(computeMetadata("bank/bic", account.Bank.Bic))
	result = result.Merge(computeMetadata("btc/domain", transaction.Characteristics.BankTransactionCode.Domain))
	result = result.Merge(computeMetadata("btc/family", transaction.Characteristics.BankTransactionCode.Family))
	result = result.Merge(computeMetadata("btc/subfamily", transaction.Characteristics.BankTransactionCode.Subfamily))
	result = result.Merge(computeMetadata("btc/description", transaction.Characteristics.BankTransactionCode.Description))
	result = result.Merge(computeMetadataBool("returned", transaction.Characteristics.Returned))
	if transaction.CounterpartyDetails != nil && transaction.CounterpartyDetails.Name != "" {
		result = result.Merge(computeMetadata("counterparty/name", transaction.CounterpartyDetails.Name))
		if transaction.CounterpartyDetails.ExternalAccount != nil && transaction.CounterpartyDetails.ExternalAccount.Identifier != nil {
			result = result.Merge(computeMetadata("counterparty/bank/bic", transaction.CounterpartyDetails.ExternalAccount.Bank.Bic))
			result = result.Merge(computeMetadata("counterparty/bank/name", transaction.CounterpartyDetails.ExternalAccount.Bank.Name))
			result = result.Merge(computeMetadata(
				fmt.Sprintf("counterparty/identifier/%s", transaction.CounterpartyDetails.ExternalAccount.Identifier.Type),
				transaction.CounterpartyDetails.ExternalAccount.Identifier.Number))
		}
	}
	if transaction.Characteristics.Returned {
		result = result.Merge(computeMetadata("returnReason/code", transaction.Characteristics.ReturnReason.Code))
		result = result.Merge(computeMetadata("returnReason/description", transaction.Characteristics.ReturnReason.Description))
		result = result.Merge(computeMetadata("returnReason/btc/domain", transaction.Characteristics.ReturnReason.OriginalBankTransactionCode.Domain))
		result = result.Merge(computeMetadata("returnReason/btc/family", transaction.Characteristics.ReturnReason.OriginalBankTransactionCode.Family))
		result = result.Merge(computeMetadata("returnReason/btc/subfamily", transaction.Characteristics.ReturnReason.OriginalBankTransactionCode.Subfamily))
		result = result.Merge(computeMetadata("returnReason/btc/description", transaction.Characteristics.ReturnReason.OriginalBankTransactionCode.Description))
	}
	if transaction.Characteristics.VirtualAccount != nil {
		result = result.Merge(computeMetadata("virtualAccount/market", transaction.Characteristics.VirtualAccount.Market))
		result = result.Merge(computeMetadata("virtualAccount/rawIdentifier", transaction.Characteristics.VirtualAccount.RawIdentifier))
		result = result.Merge(computeMetadata("virtualAccount/bank/id", transaction.Characteristics.VirtualAccount.Bank.ID))
		result = result.Merge(computeMetadata("virtualAccount/bank/name", transaction.Characteristics.VirtualAccount.Bank.Name))
		result = result.Merge(computeMetadata("virtualAccount/bank/bic", transaction.Characteristics.VirtualAccount.Bank.Bic))
		result = result.Merge(computeMetadata("virtualAccount/identifier/holderName", *transaction.Characteristics.VirtualAccount.Identifier.HolderName))
		result = result.Merge(computeMetadata("virtualAccount/identifier/market", transaction.Characteristics.VirtualAccount.Identifier.Market))
		result = result.Merge(computeMetadata("virtualAccount/identifier/type", transaction.Characteristics.VirtualAccount.Identifier.Type))
		result = result.Merge(computeMetadata("virtualAccount/identifier/number", transaction.Characteristics.VirtualAccount.Identifier.Number))
	}
	result = result.Merge(computeMetadata("reconciliation/status", transaction.Reconciliation.Status))
	result = result.Merge(computeMetadata("reconciliation/transactableId", transaction.Reconciliation.TransactableID))
	result = result.Merge(computeMetadata("reconciliation/transactableType", transaction.Reconciliation.TransactableType))
	if transaction.Characteristics.CurrencyExchange != nil {
		result = result.Merge(computeMetadata("currencyExchange/sourceCurrency", transaction.Characteristics.CurrencyExchange.SourceCurrency))
		result = result.Merge(computeMetadata("currencyExchange/targetCurrency", transaction.Characteristics.CurrencyExchange.TargetCurrency))
		result = result.Merge(computeMetadata("currencyExchange/exchangeRate", transaction.Characteristics.CurrencyExchange.ExchangeRate))
		result = result.Merge(computeMetadata("currencyExchange/unitCurrency", transaction.Characteristics.CurrencyExchange.UnitCurrency))
	}
	if transaction.CounterpartyDetails != nil && transaction.CounterpartyDetails.MandateReference != "" {
		result = result.Merge(computeMetadata("mandateReference", transaction.CounterpartyDetails.MandateReference))
	}

	return result
}

func atlarTransactionAmountToPaymentAbsoluteAmount(atlarAmount int64) (*big.Int, error) {
	var amount big.Int
	amountInt := amount.SetInt64(atlarAmount)
	amountInt = amountInt.Abs(amountInt)
	return amountInt, nil
}
