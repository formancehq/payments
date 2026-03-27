// Package connector provides the public interface for Payment Service Provider connectors.
//
// This package re-exports types from internal/models that external connector
// implementations need. It serves as a stable public API while keeping the
// canonical type definitions internal.
//
// External connector developers should import this package:
//
//	import "github.com/formancehq/payments/pkg/connector"
//
// Example usage:
//
//	func (p *MyConnector) FetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
//	    return connector.FetchNextAccountsResponse{
//	        Accounts: []connector.PSPAccount{
//	            {Reference: "acc-123", CreatedAt: time.Now(), Raw: json.RawMessage(`{}`)},
//	        },
//	    }, nil
//	}
package connector
