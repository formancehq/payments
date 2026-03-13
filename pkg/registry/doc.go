// Package registry provides the plugin registration API for payment connectors.
//
// This package enables connectors to be developed in separate git repositories
// by providing a public API for registering connector plugins.
//
// # Usage
//
// Connectors register themselves using the RegisterPlugin function, typically
// in an init() function:
//
//	package myconnector
//
//	import (
//	    "github.com/formancehq/payments/pkg/connector"
//	    "github.com/formancehq/payments/pkg/registry"
//	)
//
//	func init() {
//	    registry.RegisterPlugin("myconnector", connector.PluginTypePSP,
//	        func(id connector.ConnectorID, name string, logger logging.Logger, cfg json.RawMessage) (connector.Plugin, error) {
//	            return New(name, logger, cfg)
//	        },
//	        capabilities, Config{}, PAGE_SIZE,
//	    )
//	}
//
// To include the connector in a build, add a blank import to the connector loader:
//
//	_ "github.com/myorg/myconnector"
package registry
