package workflow

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/pointer"
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
	paymentReversalID   models.PaymentInitiationReversalID

	connector                   models.Connector
	bankAccount                 models.BankAccount
	paymentServiceUser          models.PaymentServiceUser
	paymentPayout               models.Payment
	paymentWithAdjustmentAmount models.Payment
	account                     models.Account
	paymentInitiationPayout     models.PaymentInitiation
	paymentInitiationTransfer   models.PaymentInitiation
	paymentReversal             models.PaymentInitiationReversal

	pspAccount         models.PSPAccount
	pspPayment         models.PSPPayment
	pspPaymentReversed models.PSPPayment
	pspBalance         models.PSPBalance
	pspOther           models.PSPOther
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
	logger := logging.Testing()
	w := Workflow{
		plugins:        plugins.New(logger, true),
		stackPublicURL: "http://localhost:8080",
		stack:          "test",
		logger:         logger,
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

	registry.RegisterPlugin("test", models.PluginTypePSP, func(models.ConnectorID, string, logging.Logger, json.RawMessage) (models.Plugin, error) {
		return nil, nil
	}, []models.Capability{}, struct{}{})
	err := s.w.plugins.LoadPlugin(s.connectorID, "test", "test", models.DefaultConfig(), json.RawMessage(`{}`), true)
	s.NoError(err)

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

	s.paymentReversalID = models.PaymentInitiationReversalID{
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

	s.paymentServiceUser = models.PaymentServiceUser{
		ID:        uuid.New(),
		Name:      "test",
		CreatedAt: now,
		ContactDetails: &models.ContactDetails{
			Email:       pointer.For("test"),
			PhoneNumber: pointer.For("test"),
		},
		Address: &models.Address{
			StreetName:   pointer.For("test"),
			StreetNumber: pointer.For("test"),
			City:         pointer.For("test"),
			Region:       pointer.For("test"),
			PostalCode:   pointer.For("test"),
			Country:      pointer.For("test"),
		},
		BankAccountIDs: []uuid.UUID{
			s.bankAccount.ID,
		},
		Metadata: map[string]string{
			"foo": "bar",
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

	s.paymentWithAdjustmentAmount = models.Payment{
		ID:          s.paymentPayoutID,
		ConnectorID: s.connectorID,
		Reference:   "test",
		CreatedAt:   now,
		Type:        models.PAYMENT_TYPE_PAYOUT,
		Amount:      big.NewInt(100),
		Asset:       "USD/2",
		Scheme:      models.PAYMENT_SCHEME_A2A,
		Status:      models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT,
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
					Status:    models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT,
				},
				Reference: "test",
				CreatedAt: now,
				Status:    models.PAYMENT_STATUS_AMOUNT_ADJUSTMENT,
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

	s.paymentReversal = models.PaymentInitiationReversal{
		ID:                  s.paymentReversalID,
		ConnectorID:         s.connectorID,
		PaymentInitiationID: s.paymentInitiationID,
		Reference:           "test",
		CreatedAt:           now,
		Description:         "test",
		Amount:              big.NewInt(50),
		Asset:               "USD/2",
		Metadata: map[string]string{
			"key": "value",
		},
	}

	s.connector = models.Connector{
		ID:                   s.connectorID,
		Name:                 "test",
		CreatedAt:            now,
		Provider:             "test",
		ScheduledForDeletion: false,
		Config:               []byte(`{"name": "test", "pollingPeriod": "2m", "pageSize": 25}`),
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

	s.pspPaymentReversed = models.PSPPayment{
		Reference:                   "test",
		CreatedAt:                   now,
		Type:                        models.PAYMENT_TYPE_PAYOUT,
		Amount:                      big.NewInt(100),
		Asset:                       "USD/2",
		Scheme:                      models.PAYMENT_SCHEME_A2A,
		Status:                      models.PAYMENT_STATUS_REFUNDED,
		SourceAccountReference:      pointer.For("test1"),
		DestinationAccountReference: pointer.For("test2"),
		Metadata: map[string]string{
			"key": "value",
		},
		Raw: []byte(`{}`),
	}

	s.pspBalance = models.PSPBalance{
		AccountReference: "test",
		CreatedAt:        now,
		Amount:           big.NewInt(100),
		Asset:            "USD/2",
	}

	s.pspOther = models.PSPOther{
		ID:    "test",
		Other: []byte(`{}`),
	}
}
