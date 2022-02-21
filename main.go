//go:generate openapi-generator generate -i swagger.yml -g go -o ./pkg/paymentclient --additional-properties=packageName:ledgerclient --git-user-id=numary --git-repo-id=payments-cloud --additional-properties=isGoSubmodule=true --additional-properties=packageName=paymentclient
package main

import "github.com/numary/payment/cmd"

func main() {
	cmd.Execute()
}
