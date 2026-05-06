package routable

import "github.com/formancehq/payments/ee/plugins/routable/client"

// MetadataPrefix namespaces every Routable-specific metadata key written on
// PSP entities. Mirrors the convention used by other Formance EE plugins
// (e.g. com.coinbaseprime.spec/) so dashboards stay consistent.
const MetadataPrefix = "com.routable.spec/"

// Metadata keys read from PSPPaymentInitiation.Metadata when initiating a
// payable. Operators can override the per-payable Routable type/delivery
// method or supply a richer line-item description without code changes.
const (
	MetadataKeyType             = MetadataPrefix + "type"
	MetadataKeyDeliveryMethod   = MetadataPrefix + "delivery_method"
	MetadataKeyExternalID       = MetadataPrefix + "external_id"
	MetadataKeyMemo             = MetadataPrefix + "memo"
	MetadataKeyLineDescription  = MetadataPrefix + "line_item_description"
	MetadataKeyActingTeamMember = MetadataPrefix + "acting_team_member"
)

// Default Routable payable type/delivery_method when the caller does not
// provide overrides. ach + ach_standard is the most common money-out path
// and the safest default for an out-of-the-box experience.
const (
	defaultPayableType    = "ach"
	defaultDeliveryMethod = "ach_standard"
)

// fieldOr returns the metadata value for key, falling back to fallback when
// the key is absent or empty.
func fieldOr(meta map[string]string, key, fallback string) string {
	if v, ok := meta[key]; ok && v != "" {
		return v
	}
	return fallback
}

// companyMetadata flattens a Routable Company into a stable metadata map.
// Address fields are kept namespaced so downstream tenants can lift them
// into Formance accounts without re-parsing JSON blobs.
func companyMetadata(co client.Company) map[string]string {
	m := map[string]string{
		MetadataPrefix + "object":         co.Object,
		MetadataPrefix + "type":           co.Type,
		MetadataPrefix + "status":         co.Status,
		MetadataPrefix + "country_code":   co.CountryCode,
		MetadataPrefix + "is_vendor":      boolString(co.IsVendor),
		MetadataPrefix + "is_customer":    boolString(co.IsCustomer),
		MetadataPrefix + "is_archived":    boolString(co.IsArchived),
		MetadataPrefix + "external_id":    co.ExternalID,
		MetadataPrefix + "business_name":  co.BusinessName,
		MetadataPrefix + "display_name":   co.DisplayName,
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

// settingsAccountMetadata captures the few bookkeeping fields Routable
// returns on a settings account (funding source) that are useful to keep on
// the Formance PSPAccount.
func settingsAccountMetadata(a client.Account) map[string]string {
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

// payableMetadata captures the few bookkeeping fields Routable returns on a
// payable that we want to keep on the Formance PSPPayment.
func payableMetadata(p client.Payable) map[string]string {
	return stripEmpty(map[string]string{
		MetadataPrefix + "type":            p.Type,
		MetadataPrefix + "delivery_method": p.DeliveryMethod,
		MetadataPrefix + "status":          p.Status,
		MetadataPrefix + "external_id":     p.ExternalID,
		MetadataPrefix + "memo":            p.Memo,
		MetadataPrefix + "reference":       p.Reference,
	})
}

// receivableMetadata captures the few bookkeeping fields Routable returns on
// a receivable that we want to keep on the Formance PSPPayment.
func receivableMetadata(r client.Receivable) map[string]string {
	return stripEmpty(map[string]string{
		MetadataPrefix + "type":            r.Type,
		MetadataPrefix + "delivery_method": r.DeliveryMethod,
		MetadataPrefix + "status":          r.Status,
		MetadataPrefix + "external_id":     r.ExternalID,
		MetadataPrefix + "memo":            r.Memo,
		MetadataPrefix + "reference":       r.Reference,
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
