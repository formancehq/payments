package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	golibtemporal "github.com/formancehq/go-libs/v3/temporal"
	"github.com/formancehq/payments/internal/connectors/engine"
	"github.com/formancehq/payments/internal/connectors/engine/workflow"
	"github.com/formancehq/payments/internal/models"
	"github.com/spf13/cobra"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

const (
	DryRunFlag = "dry-run"
)

// generic command which can be used to trigger workflows with custom inputs
// keep in mind that the workflows will be executed by the payments-worker responsible
// for reading from the queue specified
func main() {
	cmd := &cobra.Command{
		Use:               "payments-tool",
		Short:             "pt",
		DisableAutoGenTag: true,
		Version:           "v0",
	}

	stripeMigration := newStripeMigration()
	stripeMigration.Flags().Bool(DryRunFlag, true, "if set to true the command should execute exclusively read only operations")
	cmd.AddCommand(stripeMigration)

	err := cmd.Execute()
	if err != nil {
		log.Fatalf("failed: %s", err)
	}
}

func newStripeMigration() *cobra.Command {
	return &cobra.Command{
		Use:   "stripe-migration",
		Short: "stripe-migration",
		Run:   runStripeMigration,
	}
}

func runStripeMigration(cmd *cobra.Command, args []string) {
	temporalClient, err := temporalClientFromFlags(cmd)
	if err != nil {
		log.Fatalf("failed initiating temporal: %s", err)
	}

	if len(args) != 2 {
		log.Fatalf("expected stackID and connectorID as arguments, got: %s", args)
	}

	stack := args[0]
	if len(stack) != 17 {
		log.Fatalf("stack %q is invalid", stack)
	}

	connectorIDStr := args[1]
	connectorID, err := models.ConnectorIDFromString(connectorIDStr)
	if err != nil {
		log.Fatalf("connectorID %q is invalid", connectorIDStr)
	}

	dryRun, _ := cmd.Flags().GetBool(DryRunFlag)
	if dryRun {
		fmt.Printf("dry-run enabled: %t\n", dryRun)
	}

	if err := doStripeMigration(cmd.Context(), temporalClient, stack, connectorID, dryRun); err != nil {
		log.Fatalf("failed to do migration process: %s", err)
	}
}

func doStripeMigration(
	ctx context.Context,
	temporalClient client.Client,
	stack string,
	connectorID models.ConnectorID,
	dryRun bool,
) error {
	if connectorID.Provider != "stripe" {
		return fmt.Errorf("connectorID must belong to stripe, but got: %q ", connectorID.Provider)
	}

	future, err := temporalClient.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:                                       fmt.Sprintf("migrate-stripe-schedules-%s-list", stack),
			TaskQueue:                                engine.GetDefaultTaskQueue(stack),
			WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
			WorkflowExecutionErrorWhenAlreadyStarted: true,
			SearchAttributes: map[string]interface{}{
				workflow.SearchAttributeStack: stack,
			},
		},
		workflow.RunListActiveSchedules,
		workflow.ListActiveSchedules{
			// TODO paging if we expect more than one page
			ConnectorID: connectorID,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to start workflow: %w", err)
	}

	var result workflow.ListActiveSchedulesResult
	err = future.Get(ctx, &result)
	if err != nil {
		return fmt.Errorf("failed to fetch result of workflow: %w", err)
	}

	deprecatedCapability := models.CAPABILITY_FETCH_BALANCES

	fmt.Printf("total schedules: %d - Next PageToken: %q\n\n", len(result.Data), result.NextPageToken)
	if result.NextPageToken != "" {
		return fmt.Errorf("please implement paging for this tool, connector %s has more than %d schedules", connectorID.String(), len(result.Data))
	}

	deprecatedSchedules := make([]models.Schedule, 0, len(result.Data))
	workflowIDs := make(map[string]string)
	for _, schedule := range result.Data {
		if !strings.Contains(schedule.ID, deprecatedCapability.String()) {
			continue
		}

		deprecatedSchedules = append(deprecatedSchedules, schedule)
		if dryRun {
			continue
		}

		future, err := temporalClient.ExecuteWorkflow(
			ctx,
			client.StartWorkflowOptions{
				ID:                                       fmt.Sprintf("migrate-stripe-schedules-%s-terminate-%d", stack, len(deprecatedSchedules)),
				TaskQueue:                                engine.GetDefaultTaskQueue(stack),
				WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
				WorkflowExecutionErrorWhenAlreadyStarted: true,
				SearchAttributes: map[string]interface{}{
					workflow.SearchAttributeStack: stack,
				},
			},
			workflow.RunTerminateScheduleByID,
			workflow.TerminateScheduleByID{
				ConnectorID: connectorID,
				ScheduleID:  schedule.ID,
			},
		)
		if err != nil {
			fmt.Printf("failed to trigger termination of schedule %s: %s\n", schedule.ID, err)
			break
		}
		workflowIDs[schedule.ID] = future.GetID()
	}

	if len(workflowIDs) != len(deprecatedSchedules) {
		fmt.Printf("\nworkflowIDs: %+v\n\ndeprecated schedules: %+v\n", workflowIDs, deprecatedSchedules)
		return fmt.Errorf("failed to complete deletion of deprecated schedules")
	}
	for scheduleID, workflowID := range workflowIDs {
		fmt.Printf("deletion of %q scheduled: %s\n", scheduleID, workflowID)
	}
	fmt.Printf("\nfinished scheduling deletion of %d deprecated schedules\n", len(deprecatedSchedules))
	return nil
}

func temporalClientFromFlags(cmd *cobra.Command) (client.Client, error) {
	golibtemporal.AddFlags(cmd.Flags())

	address, _ := cmd.Flags().GetString(golibtemporal.TemporalAddressFlag)
	namespace, _ := cmd.Flags().GetString(golibtemporal.TemporalNamespaceFlag)
	certStr, _ := cmd.Flags().GetString(golibtemporal.TemporalSSLClientCertFlag)
	temporalKey, _ := cmd.Flags().GetString(golibtemporal.TemporalSSLClientKeyFlag)

	var cert *tls.Certificate
	if temporalKey != "" && certStr != "" {
		clientCert, err := tls.X509KeyPair([]byte(certStr), []byte(temporalKey))
		if err != nil {
			return nil, fmt.Errorf("invalid certificate: %w", err)
		}
		cert = &clientCert
	}

	options := client.Options{
		HostPort:  address,
		Namespace: namespace,
	}
	if cert != nil {
		options.ConnectionOptions = client.ConnectionOptions{
			TLS: &tls.Config{Certificates: []tls.Certificate{*cert}},
		}
	}
	temporalClient, err := client.Dial(options)
	if err != nil {
		return nil, fmt.Errorf("unable to create Temporal Client: %w", err)
	}
	return temporalClient, nil
}
