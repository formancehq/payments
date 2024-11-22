package main

import (
	"github.com/formancehq/payments/cmd"
	_ "github.com/formancehq/payments/internal/connectors/plugins/public"
)

func main() {
	cmd.Execute()
}
