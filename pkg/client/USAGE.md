<!-- Start SDK Example Usage [usage] -->
```go
package main

import (
	"context"
	"github.com/formancehq/payments/pkg/client"
	"log"
	"os"
)

func main() {
	ctx := context.Background()

	s := client.New(
		"https://api.example.com",
		client.WithSecurity(os.Getenv("FORMANCE_AUTHORIZATION")),
	)

	res, err := s.Payments.V1.GetServerInfo(ctx)
	if err != nil {
		log.Fatal(err)
	}
	if res.ServerInfo != nil {
		// handle response
	}
}

```
<!-- End SDK Example Usage [usage] -->