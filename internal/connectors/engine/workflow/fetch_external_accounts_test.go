package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/temporal"
)

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_WithoutInstance_Success() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextExternalAccountsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextExternalAccountsRequest) (*models.FetchNextExternalAccountsResponse, error) {
		return &models.FetchNextExternalAccountsResponse{
			ExternalAccounts: []models.PSPAccount{
				s.pspAccount,
			},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, accounts []models.Account) error {
		s.Equal(1, len(accounts))
		s.Equal(s.accountID, accounts[0].ID)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:      models.DefaultConfig(),
		ConnectorID: s.connectorID,
		FromPayload: &FromPayload{
			ID:      "1",
			Payload: []byte(`{}`),
		},
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_Success() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.False(instance.Terminated)
		return nil
	})
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextExternalAccountsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextExternalAccountsRequest) (*models.FetchNextExternalAccountsResponse, error) {
		return &models.FetchNextExternalAccountsResponse{
			ExternalAccounts: []models.PSPAccount{
				s.pspAccount,
			},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, accounts []models.Account) error {
		s.Equal(1, len(accounts))
		s.Equal(s.accountID, accounts[0].ID)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.True(instance.Terminated)
		return nil
	})

	s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_HasMoreLoop_Success() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.False(instance.Terminated)
		return nil
	})
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextExternalAccountsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextExternalAccountsRequest) (*models.FetchNextExternalAccountsResponse, error) {
		return &models.FetchNextExternalAccountsResponse{
			ExternalAccounts: []models.PSPAccount{
				s.pspAccount,
			},
			NewState: []byte(`{}`),
			HasMore:  true,
		}, nil
	})
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, accounts []models.Account) error {
		s.Equal(1, len(accounts))
		s.Equal(s.accountID, accounts[0].ID)
		return nil
	})
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.OnActivity(activities.PluginFetchNextExternalAccountsActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextExternalAccountsRequest) (*models.FetchNextExternalAccountsResponse, error) {
		return &models.FetchNextExternalAccountsResponse{
			ExternalAccounts: []models.PSPAccount{},
			NewState:         []byte(`{}`),
			HasMore:          false,
		}, nil
	})
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.True(instance.Terminated)
		return nil
	})

	s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_StorageInstancesStore_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)

	s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_StorageStatesGet_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_PluginFetchNextExternalAccounts_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextExternalAccountsActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "PLUGIN", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_StorageAccountsStore_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextExternalAccountsActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextExternalAccountsResponse{
		ExternalAccounts: []models.PSPAccount{
			s.pspAccount,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_RunSendEvents_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextExternalAccountsActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextExternalAccountsResponse{
		ExternalAccounts: []models.PSPAccount{
			s.pspAccount,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "WORKFLOW", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_Run_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextExternalAccountsActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextExternalAccountsResponse{
		ExternalAccounts: []models.PSPAccount{
			s.pspAccount,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "WORKFLOW", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_StorageStatesStore_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextExternalAccountsActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextExternalAccountsResponse{
		ExternalAccounts: []models.PSPAccount{
			s.pspAccount,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextExternalAccounts_StorageInstancesUpdate_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_EXTERNAL_ACCOUNTS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextExternalAccountsActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextExternalAccountsResponse{
		ExternalAccounts: []models.PSPAccount{
			s.pspAccount,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnActivity(activities.StorageAccountsStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(RunSendEvents, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.Nil(instance.Error)
		return temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test"))
	})

	s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.env.ExecuteWorkflow(RunFetchNextExternalAccounts, FetchNextExternalAccounts{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err := s.env.GetWorkflowError()
	s.Error(err)
}
