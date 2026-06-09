package mappers

import "github.com/formancehq/payments/ee/plugins/routable/client"

const MetadataPrefix = "com.routable.spec/"

// Metadata keys read from PSPPaymentInitiation.Metadata when initiating
// a payable. See MAPPINGS.md §5 for the per-key semantics and defaults.
const (
	MetadataKeyType             = MetadataPrefix + "type"
	MetadataKeyDeliveryMethod   = MetadataPrefix + "delivery_method"
	MetadataKeyExternalID       = MetadataPrefix + "external_id"
	MetadataKeyMemo             = MetadataPrefix + "memo"
	MetadataKeyMessage          = MetadataPrefix + "message"
	MetadataKeyLineDescription  = MetadataPrefix + "line_item_description"
	MetadataKeyActingTeamMember = MetadataPrefix + "acting_team_member"
)

// Self-describing aliases written on synced PSPPayment metadata so
// dashboards and operators don't have to know Routable's wire
// vocabulary to correlate a Payment with the originating Formance
// Transfer (PaymentInitiation). MAPPINGS.md §5.5 documents the full
// correlation contract; the short version is:
//
//   - MetadataKeyRoutablePayableID is always present (mirrors the
//     Routable payable UUID, which also lives on PSPPayment.Reference).
//   - MetadataKeyPaymentInitiationReference is present only when
//     external_id is set, i.e. when Formance initiated the payable.
const (
	MetadataKeyPaymentInitiationReference = MetadataPrefix + "payment_initiation_reference"
	MetadataKeyRoutablePayableID          = MetadataPrefix + "payable_id"
)

const (
	DefaultPayableType    = "ach"
	DefaultDeliveryMethod = "ach_standard"
)

// FieldOr returns meta[key], or fallback when the key is absent or empty.
func FieldOr(meta map[string]string, key, fallback string) string {
	if v, ok := meta[key]; ok && v != "" {
		return v
	}
	return fallback
}

func CompanyMetadata(co client.Company) map[string]string {
	m := map[string]string{
		MetadataPrefix + "object":        co.Object,
		MetadataPrefix + "type":          co.Type,
		MetadataPrefix + "status":        co.Status,
		MetadataPrefix + "country_code":  co.CountryCode,
		MetadataPrefix + "is_vendor":     boolString(co.IsVendor),
		MetadataPrefix + "is_customer":   boolString(co.IsCustomer),
		MetadataPrefix + "is_archived":   boolString(co.IsArchived),
		MetadataPrefix + "external_id":   co.ExternalID,
		MetadataPrefix + "business_name": co.BusinessName,
		MetadataPrefix + "display_name":  co.DisplayName,
	}
	if co.RegisteredAddress != nil {
		addr := co.RegisteredAddress
		m[MetadataPrefix+"address.line_1"] = addr.Line1
		m[MetadataPrefix+"address.line_2"] = addr.Line2
		m[MetadataPrefix+"address.city"] = addr.City
		m[MetadataPrefix+"address.state"] = addr.State
		m[MetadataPrefix+"address.postal_code"] = addr.PostalCode
		m[MetadataPrefix+"address.country"] = addr.Country
	}
	return stripEmpty(m)
}

func SettingsAccountMetadata(a client.Account) map[string]string {
	return stripEmpty(map[string]string{
		MetadataPrefix + "object":         a.Object,
		MetadataPrefix + "type":           a.Type,
		MetadataPrefix + "is_valid":       boolString(a.IsValid),
		MetadataPrefix + "currency_code":  a.CurrencyCode,
		MetadataPrefix + "account_type":   a.TypeDetails.AccountType,
		MetadataPrefix + "bank_name":      a.TypeDetails.BankName,
		MetadataPrefix + "account_number": a.TypeDetails.AccountNumber,
		MetadataPrefix + "routing_number": a.TypeDetails.RoutingNumber,
	})
}

func PayableMetadata(p client.Payable) map[string]string {
	return stripEmpty(map[string]string{
		MetadataPrefix + "type":               p.Type,
		MetadataPrefix + "delivery_method":    p.DeliveryMethod,
		MetadataPrefix + "status":             p.Status,
		MetadataKeyExternalID:                 p.ExternalID,
		MetadataKeyPaymentInitiationReference: p.ExternalID,
		MetadataKeyRoutablePayableID:          p.ID,
		MetadataPrefix + "memo":               p.Memo,
		MetadataPrefix + "reference":          p.Reference,
	})
}

func ReceivableMetadata(r client.Receivable) map[string]string {
	return stripEmpty(map[string]string{
		MetadataPrefix + "type":               r.Type,
		MetadataPrefix + "delivery_method":    r.DeliveryMethod,
		MetadataPrefix + "status":             r.Status,
		MetadataKeyExternalID:                 r.ExternalID,
		MetadataKeyPaymentInitiationReference: r.ExternalID,
		MetadataKeyRoutablePayableID:          r.ID,
		MetadataPrefix + "memo":               r.Memo,
		MetadataPrefix + "reference":          r.Reference,
	})
}

func stripEmpty(m map[string]string) map[string]string {
	for k, v := range m {
		if v == "" {
			delete(m, k)
		}
	}
	return m
}

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
