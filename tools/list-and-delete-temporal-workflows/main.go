package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"sync"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
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

	var nextPageToken []byte
	wg := sync.WaitGroup{}
	for {
		resp, err := temporalClient.WorkflowService().ListWorkflowExecutions(
			ctx,
			&workflowservice.ListWorkflowExecutionsRequest{
				Namespace:     "local-operator.sihc8",
				PageSize:      100,
				NextPageToken: nextPageToken,
				Query:         fmt.Sprintf("Stack=\"%s\"", *temporalStack),
			},
		)
		if err != nil {
			log.Fatalln("Unable to list workflows", err)
		}

		for _, e := range resp.Executions {
			if e.Status != enums.WORKFLOW_EXECUTION_STATUS_RUNNING {
				continue
			}

			wg.Add(1)

			go func() {
				defer wg.Done()

				// close workflow
				_, err := temporalClient.WorkflowService().TerminateWorkflowExecution(
					ctx,
					&workflowservice.TerminateWorkflowExecutionRequest{
						Namespace:         *temporalNamespace,
						WorkflowExecution: e.Execution,
						Reason:            "done",
					},
				)
				if err != nil {
					return
				}

				fmt.Println("workflow terminated: ", e.Execution.GetWorkflowId(), e.Execution.GetRunId())
			}()
		}

		wg.Wait()

		if resp.NextPageToken == nil {
			break
		}

		nextPageToken = resp.NextPageToken
	}
}
