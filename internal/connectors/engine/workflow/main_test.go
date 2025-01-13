package workflow

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/formancehq/go-libs/v2/logging"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/connectors/engine/plugins"
	"github.com/formancehq/payments/internal/connectors/plugins/registry"
	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
	temporalworkflow "go.temporal.io/sdk/workflow"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	w   *Workflow
	a   *activities.Activities
	env *testsuite.TestWorkflowEnvironment

	connectorID         models.ConnectorID
	accountID           models.AccountID
	paymentPayoutID     models.PaymentID
	paymentInitiationID models.PaymentInitiationID

	bankAccount               models.BankAccount
	paymentPayout             models.Payment
	account                   models.Account
	paymentInitiationPayout   models.PaymentInitiation
	paymentInitiationTransfer models.PaymentInitiation

	pspAccount models.PSPAccount
	pspPayment models.PSPPayment
}

func (s *UnitTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()

	for _, def := range s.w.DefinitionSet() {
		s.env.RegisterWorkflowWithOptions(def.Func, temporalworkflow.RegisterOptions{
			Name: def.Name,
		})
	}

	s.addData()
}

func (s *UnitTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func TestUnitTestSuite(t *testing.T) {
	w := Workflow{
		plugins:        plugins.New(plugins.CallerWorker, logging.Testing(), true),
		name:           "toto",
		stackPublicURL: "http://localhost:8080",
	}
	a := activities.Activities{}

	suite.Run(t, &UnitTestSuite{w: &w, a: &a})
}

func (s *UnitTestSuite) addData() {
	now := s.env.Now().UTC()

	s.connectorID = models.ConnectorID{
		Reference: uuid.New(),
		Provider:  "test",
	}

	registry.RegisterPlugin("test", func(string, logging.Logger, json.RawMessage) (models.Plugin, error) {
		return nil, nil
	}, []models.Capability{})
	s.w.plugins.RegisterPlugin(s.connectorID, "test", models.DefaultConfig(), json.RawMessage(`{}`), true)

	s.accountID = models.AccountID{
		Reference:   "test",
		ConnectorID: s.connectorID,
	}

	s.paymentPayoutID = models.PaymentID{
		PaymentReference: models.PaymentReference{
			Reference: "test",
			Type:      models.PAYMENT_TYPE_PAYOUT,
		},
		ConnectorID: s.connectorID,
	}

	s.paymentInitiationID = models.PaymentInitiationID{
		Reference:   "test",
		ConnectorID: s.connectorID,
	}

	s.bankAccount = models.BankAccount{
		ID:        uuid.New(),
		CreatedAt: now,
		Name:      "test",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	s.paymentPayout = models.Payment{
		ID:          s.paymentPayoutID,
		ConnectorID: s.connectorID,
		Reference:   "test",
		CreatedAt:   now,
		Type:        models.PAYMENT_TYPE_PAYOUT,
		Amount:      big.NewInt(100),
		Asset:       "USD/2",
		Scheme:      models.PAYMENT_SCHEME_A2A,
		Status:      models.PAYMENT_STATUS_SUCCEEDED,
		SourceAccountID: &models.AccountID{
			Reference:   "test1",
			ConnectorID: s.connectorID,
		},
		DestinationAccountID: &models.AccountID{
			Reference:   "test2",
			ConnectorID: s.connectorID,
		},
		Metadata: map[string]string{
			"key": "value",
		},
		Adjustments: []models.PaymentAdjustment{
			{
				ID: models.PaymentAdjustmentID{
					PaymentID: s.paymentPayoutID,
					Reference: "test",
					CreatedAt: now,
					Status:    models.PAYMENT_STATUS_SUCCEEDED,
				},
				Reference: "test",
				CreatedAt: now,
				Status:    models.PAYMENT_STATUS_SUCCEEDED,
				Amount:    big.NewInt(100),
				Asset:     pointer.For("USD/2"),
				Metadata: map[string]string{
					"key": "value",
				},
				Raw: []byte(`{}`),
			},
		},
	}

	s.paymentInitiationPayout = models.PaymentInitiation{
		ID:          s.paymentInitiationID,
		ConnectorID: s.connectorID,
		Reference:   "test",
		CreatedAt:   s.env.Now().UTC(),
		ScheduledAt: s.env.Now().UTC(),
		Description: "test_payout",
		Type:        models.PAYMENT_INITIATION_TYPE_PAYOUT,
		SourceAccountID: &models.AccountID{
			Reference:   "test1",
			ConnectorID: s.connectorID,
		},
		DestinationAccountID: &models.AccountID{
			Reference:   "test2",
			ConnectorID: s.connectorID,
		},
		Amount: big.NewInt(100),
		Asset:  "USD/2",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	s.paymentInitiationTransfer = models.PaymentInitiation{
		ID:          s.paymentInitiationID,
		ConnectorID: s.connectorID,
		Reference:   "test",
		CreatedAt:   s.env.Now().UTC(),
		ScheduledAt: s.env.Now().UTC(),
		Description: "test_payout",
		Type:        models.PAYMENT_INITIATION_TYPE_TRANSFER,
		SourceAccountID: &models.AccountID{
			Reference:   "test1",
			ConnectorID: s.connectorID,
		},
		DestinationAccountID: &models.AccountID{
			Reference:   "test2",
			ConnectorID: s.connectorID,
		},
		Amount: big.NewInt(100),
		Asset:  "USD/2",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	s.account = models.Account{
		ID: models.AccountID{
			Reference:   "test1",
			ConnectorID: s.connectorID,
		},
		ConnectorID:  s.connectorID,
		Reference:    "test1",
		CreatedAt:    s.env.Now().UTC(),
		Type:         models.ACCOUNT_TYPE_INTERNAL,
		Name:         pointer.For("test1"),
		DefaultAsset: pointer.For("USD/2"),
		Metadata: map[string]string{
			"key": "value",
		},
		Raw: []byte(`{}`),
	}

	s.pspAccount = models.PSPAccount{
		Reference:    "test",
		CreatedAt:    s.env.Now().UTC(),
		Name:         pointer.For("test"),
		DefaultAsset: pointer.For("USD/2"),
		Metadata: map[string]string{
			"key": "value",
		},
		Raw: []byte(`{}`),
	}

	s.pspPayment = models.PSPPayment{
		Reference:                   "test",
		CreatedAt:                   now,
		Type:                        models.PAYMENT_TYPE_PAYOUT,
		Amount:                      big.NewInt(100),
		Asset:                       "USD/2",
		Scheme:                      models.PAYMENT_SCHEME_A2A,
		Status:                      models.PAYMENT_STATUS_SUCCEEDED,
		SourceAccountReference:      pointer.For("test1"),
		DestinationAccountReference: pointer.For("test2"),
		Metadata: map[string]string{
			"key": "value",
		},
		Raw: []byte(`{}`),
	}
}
