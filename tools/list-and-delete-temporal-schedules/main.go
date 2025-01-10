package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"

	"go.temporal.io/sdk/client"
)

var (
	temporalAddress   = flag.String("temporal_address", "local-operator.sihc8.tmprl.cloud:7233", "Temporal server address")
	temporalNamespace = flag.String("namespace", "local-operator.sihc8", "Temporal namespace")
	temporalKey       = flag.String("key", "", "TLS key")
	temporalCertStr   = flag.String("cert", "", "TLS cert")
	temporalStack     = flag.String("stack", "", "Stack")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	var cert *tls.Certificate
	if temporalKey != nil && *temporalKey != "" && temporalCertStr != nil && *temporalCertStr != "" {
		clientCert, err := tls.X509KeyPair([]byte(*temporalCertStr), []byte(*temporalKey))
		if err != nil {
			panic(err)
		}
		cert = &clientCert
	}

	if temporalStack == nil || *temporalStack == "" {
		log.Fatalln("Stack is required")
	}

	options := client.Options{
		HostPort:  *temporalAddress,
		Namespace: *temporalNamespace,
	}
	if cert != nil {
		options.ConnectionOptions = client.ConnectionOptions{
			TLS: &tls.Config{Certificates: []tls.Certificate{*cert}},
		}
	}
	temporalClient, err := client.Dial(options)
	if err != nil {
		log.Fatalln("Unable to create Temporal Client", err)
	}
	defer temporalClient.Close()

	// list schedules
	listView, _ := temporalClient.ScheduleClient().List(ctx, client.ScheduleListOptions{
		PageSize: 1,
		Query:    fmt.Sprintf("Stack=\"%s\"", *temporalStack),
	})

	for listView.HasNext() {
		s, err := listView.Next()
		if err != nil {
			log.Fatalln("Unable to list schedules", err)
		}

		// get handle
		handle := temporalClient.ScheduleClient().GetHandle(ctx, s.ID)

		// delete schedule
		if err := handle.Delete(ctx); err != nil {
			log.Fatalln("Unable to delete schedule", err)
		}

		log.Println("Deleted schedule", s.ID)
	}
}
