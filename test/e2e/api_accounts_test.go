//go:build it

package test_suite

import (
	"context"
	"strings"
	"time"

	"github.com/formancehq/go-libs/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/testing/utils"
	v3 "github.com/formancehq/payments/internal/api/v3"
	"github.com/formancehq/payments/internal/models"
	evts "github.com/formancehq/payments/pkg/events"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"

	. "github.com/formancehq/payments/pkg/testserver"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Context("Payments API Accounts", func() {
	var (
		db  = UseTemplatedDatabase()
		ctx = logging.TestingContext()

		createRequest v3.CreateAccountRequest

		app *utils.Deferred[*Server]
	)

	app = NewTestServer(func() Configuration {
		return Configuration{
			Stack:                 stack,
			PostgresConfiguration: db.GetValue().ConnectionOptions(),
			NatsURL:               natsServer.GetValue().ClientURL(),
			TemporalNamespace:     temporalServer.GetValue().DefaultNamespace(),
			TemporalAddress:       temporalServer.GetValue().Address(),
			Output:                GinkgoWriter,
		}
	})

	createdAt, _ := time.Parse("2006-Jan-02", "2024-Nov-29")
	createRequest = v3.CreateAccountRequest{
		Reference:    "ref",
		AccountName:  "foo",
		CreatedAt:    createdAt,
		DefaultAsset: "USD",
		Type:         string(models.ACCOUNT_TYPE_INTERNAL),
		Metadata:     map[string]string{"key": "val"},
	}

	When("creating a new account", func() {
		var (
			connectorRes   struct{ Data string }
			createResponse struct{ Data models.Account }
			getResponse    struct{ Data models.Account }
			e              chan *nats.Msg
			err            error
		)

		DescribeTable("should be successful",
			func(ver int) {
				e = Subscribe(GinkgoT(), app.GetValue())
				connectorConf := newConnectorConfigurationFn()(uuid.New())
				err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
				Expect(err).To(BeNil())

				createRequest.ConnectorID = connectorRes.Data
				err = CreateAccount(ctx, app.GetValue(), ver, createRequest, &createResponse)
				Expect(err).To(BeNil())

				err = GetAccount(ctx, app.GetValue(), ver, createResponse.Data.ID.String(), &getResponse)
				Expect(err).To(BeNil())
				Expect(getResponse.Data).To(Equal(createResponse.Data))

				Eventually(e).Should(Receive(Event(evts.EventTypeSavedAccounts)))
			},
			Entry("with v2", 2),
			Entry("with v3", 3),
		)
	})

	When("fetching account balances", func() {
		var (
			connectorRes struct{ Data string }
			accountsRes  struct {
				Cursor bunpaginate.Cursor[models.Account]
			}
			res struct {
				Cursor bunpaginate.Cursor[models.Balance]
			}
			cl  client.Client
			err error
		)

		DescribeTable("should be successful",
			func(ver int) {
				cl = temporalServer.GetValue().DefaultClient()
				connectorConf := newConnectorConfigurationFn()(uuid.New())
				GeneratePSPData(connectorConf.Directory)
				Expect(err).To(BeNil())
				ver = 3

				err = ConnectorInstall(ctx, app.GetValue(), ver, connectorConf, &connectorRes)
				Expect(err).To(BeNil())

				waitForAccountImport(ctx, cl, connectorRes.Data)

				err = ListAccounts(ctx, app.GetValue(), ver, &accountsRes)
				Expect(err).To(BeNil())
				Expect(len(accountsRes.Cursor.Data) > 0).To(BeTrue())

				account := accountsRes.Cursor.Data[0]
				err = GetAccountBalances(ctx, app.GetValue(), ver, account.ID.String(), &res)
				Expect(err).To(BeNil())
				Expect(res.Cursor.Data).To(HaveLen(1))

				balance := res.Cursor.Data[0]
				Expect(balance.AccountID.String()).To(Equal(account.ID.String()))
				Expect(balance.Balance).NotTo(BeNil())
			},
			Entry("with v2", 2),
			Entry("with v3", 3),
		)
	})
})

func waitForAccountImport(ctx context.Context, cl client.Client, connectorID string) {
	var workflowID string
	var runID string

	req := &workflowservice.ListOpenWorkflowExecutionsRequest{Namespace: temporalServer.GetValue().DefaultNamespace()}
	workflowRes, err := cl.ListOpenWorkflow(ctx, req)
	for _, info := range workflowRes.Executions {
		if strings.Contains(info.Execution.WorkflowId, "run-tasks") && strings.Contains(info.Execution.WorkflowId, connectorID) {
			workflowID = info.Execution.WorkflowId
			runID = info.Execution.RunId
			break
		}
	}

	Expect(workflowID).NotTo(Equal(""))
	workflowRun := cl.GetWorkflow(ctx, workflowID, runID)
	err = workflowRun.Get(ctx, nil) // blocks to ensure workflow is finished
	Expect(err).To(BeNil())

	itr, err := cl.ScheduleClient().List(ctx, client.ScheduleListOptions{})
	Expect(err).To(BeNil())
	Expect(itr.HasNext()).To(BeTrue())

	var accountsSchedule *client.ScheduleListEntry
	for itr.HasNext() {
		schedule, err := itr.Next()
		Expect(err).To(BeNil())
		if strings.Contains(schedule.ID, "FETCH_ACCOUNTS") && strings.Contains(schedule.ID, connectorID) {
			accountsSchedule = schedule
			break
		}
	}
	Expect(accountsSchedule).NotTo(BeNil())
	Expect(len(accountsSchedule.RecentActions) > 0).To(BeTrue())

	var accountWorkflowID string
	var accountRunID string
	for _, action := range accountsSchedule.RecentActions {
		accountWorkflowID = action.StartWorkflowResult.WorkflowID
		accountRunID = action.StartWorkflowResult.FirstExecutionRunID
	}

	workflowRun = cl.GetWorkflow(ctx, accountWorkflowID, accountRunID)
	err = workflowRun.Get(ctx, nil) // blocks to ensure workflow is finished
	Expect(err).To(BeNil())

	itr, err = cl.ScheduleClient().List(ctx, client.ScheduleListOptions{})
	Expect(err).To(BeNil())
	Expect(itr.HasNext()).To(BeTrue())

	var balancesSchedule *client.ScheduleListEntry
	for itr.HasNext() {
		schedule, err := itr.Next()
		Expect(err).To(BeNil())
		if strings.Contains(schedule.ID, "FETCH_BALANCES") && strings.Contains(schedule.ID, connectorID) {
			balancesSchedule = schedule
			break
		}
	}
	Expect(balancesSchedule).NotTo(BeNil())
	Expect(len(balancesSchedule.RecentActions) > 0).To(BeTrue())

	var balanceWorkflowID string
	var balanceRunID string
	for _, action := range balancesSchedule.RecentActions {
		balanceWorkflowID = action.StartWorkflowResult.WorkflowID
		balanceRunID = action.StartWorkflowResult.FirstExecutionRunID
	}

	workflowRun = cl.GetWorkflow(ctx, balanceWorkflowID, balanceRunID)
	err = workflowRun.Get(ctx, nil) // blocks to ensure workflow is finished
	Expect(err).To(BeNil())
}
