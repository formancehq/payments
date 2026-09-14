module github.com/formancehq/payments/plugins/fctl

go 1.26.0

require (
	github.com/formancehq/fctl-v2-poc/pkg/plugin v0.0.0
	github.com/formancehq/payments/pkg/client v0.0.0
	go.bytecodealliance.org/pkg v0.2.2
	google.golang.org/protobuf v1.36.12
	gopkg.in/yaml.v3 v3.0.1
)

require github.com/ericlagergren/decimal v0.0.0-20221120152707-495c53812d05 // indirect

replace github.com/formancehq/payments/pkg/client => ../../pkg/client
