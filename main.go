//go:generate openapi-generator generate -i swagger.yml -g go -o ./pkg/paymentclient --additional-properties=packageName:ledgerclient --git-user-id=numary --git-repo-id=payments --additional-properties=isGoSubmodule=true --additional-properties=packageName=paymentclient -t ./gentpl
package main

import "github.com/numary/payments/cmd"

func main() {
	cmd.Execute()
}
