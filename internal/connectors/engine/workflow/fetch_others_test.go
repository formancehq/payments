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

func (s *UnitTestSuite) Test_FetchNextOthers_WithoutInstance_Success() {
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   fmt.Sprintf("%s-%s", models.CAPABILITY_FETCH_OTHERS.String(), "1"),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextOthersActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextOthersRequest) (*models.FetchNextOthersResponse, error) {
		return &models.FetchNextOthersResponse{
			Others: []models.PSPOther{
				s.pspOther,
			},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.ExecuteWorkflow(RunFetchNextOthers, FetchNextOthers{
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

func (s *UnitTestSuite) Test_FetchNextOthers_Success() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.False(instance.Terminated)
		return nil
	})
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_OTHERS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextOthersActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextOthersRequest) (*models.FetchNextOthersResponse, error) {
		return &models.FetchNextOthersResponse{
			Others: []models.PSPOther{
				s.pspOther,
			},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.True(instance.Terminated)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextOthers, FetchNextOthers{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextOthers_HasMoreLoop_Success() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.False(instance.Terminated)
		return nil
	})
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_OTHERS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextOthersActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextOthersRequest) (*models.FetchNextOthersResponse, error) {
		return &models.FetchNextOthersResponse{
			Others: []models.PSPOther{
				s.pspOther,
			},
			NewState: []byte(`{}`),
			HasMore:  true,
		}, nil
	})
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.OnActivity(activities.PluginFetchNextOthersActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, req activities.FetchNextOthersRequest) (*models.FetchNextOthersResponse, error) {
		return &models.FetchNextOthersResponse{
			Others:   []models.PSPOther{},
			NewState: []byte(`{}`),
			HasMore:  false,
		}, nil
	})
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)

	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.Equal("test", instance.ScheduleID)
		s.Equal(s.connectorID, instance.ConnectorID)
		s.True(instance.Terminated)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextOthers, FetchNextOthers{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextOthers_StorageInstancesStore_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextOthers, FetchNextOthers{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Error(err)
}

func (s *UnitTestSuite) Test_FetchNextOthers_StorageStatesGet_Error() {
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

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextOthers, FetchNextOthers{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextOthers_PluginFetchNextOthers_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_OTHERS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextOthersActivity, mock.Anything, mock.Anything).Once().Return(
		nil,
		temporal.NewNonRetryableApplicationError("test", "PLUGIN", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextOthers, FetchNextOthers{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextOthers_Run_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_OTHERS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextOthersActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextOthersResponse{
		Others: []models.PSPOther{
			s.pspOther,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "WORKFLOW", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextOthers, FetchNextOthers{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextOthers_StorageStatesStore_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_OTHERS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextOthersActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextOthersResponse{
		Others: []models.PSPOther{
			s.pspOther,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(
		temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test")),
	)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.NotNil(instance.Error)
		return nil
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextOthers, FetchNextOthers{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.NoError(err)
}

func (s *UnitTestSuite) Test_FetchNextOthers_StorageInstancesUpdate_Error() {
	s.env.OnActivity(activities.StorageInstancesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesGetActivity, mock.Anything, mock.Anything).Once().Return(
		&models.State{
			ID: models.StateID{
				Reference:   models.CAPABILITY_FETCH_OTHERS.String(),
				ConnectorID: s.connectorID,
			},
			ConnectorID: s.connectorID,
			State:       []byte(`{}`),
		},
		nil,
	)
	s.env.OnActivity(activities.PluginFetchNextOthersActivity, mock.Anything, mock.Anything).Once().Return(&models.FetchNextOthersResponse{
		Others: []models.PSPOther{
			s.pspOther,
		},
		NewState: []byte(`{}`),
		HasMore:  false,
	}, nil)
	s.env.OnWorkflow(Run, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageStatesStoreActivity, mock.Anything, mock.Anything).Once().Return(nil)
	s.env.OnActivity(activities.StorageInstancesUpdateActivity, mock.Anything, mock.Anything).Once().Return(func(ctx context.Context, instance models.Instance) error {
		s.True(instance.Terminated)
		s.Nil(instance.Error)
		return temporal.NewNonRetryableApplicationError("test", "STORAGE", errors.New("test"))
	})

	err := s.env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(temporal.NewSearchAttributeKeyKeyword(SearchAttributeScheduleID).ValueSet("test")))
	s.NoError(err)
	s.env.ExecuteWorkflow(RunFetchNextOthers, FetchNextOthers{
		Config:       models.DefaultConfig(),
		ConnectorID:  s.connectorID,
		FromPayload:  nil,
		Periodically: false,
	}, []models.ConnectorTaskTree{{
		Name: "test",
	}})

	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Error(err)
}
