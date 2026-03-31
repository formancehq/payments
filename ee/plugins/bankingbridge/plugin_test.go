package bankingbridge

import (
	"testing"

	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/payments/ee/plugins/bankingbridge/client"
	"github.com/formancehq/payments/internal/connectors/plugins"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type PluginTestSuite struct {
	suite.Suite

	client *client.MockClient
	plugin *Plugin
}

func TestPluginTestSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(PluginTestSuite))
}

func (suite *PluginTestSuite) SetupTest() {
	logger := logging.Testing()
	ctrl := gomock.NewController(suite.T())
	suite.client = client.NewMockClient(ctrl)
	suite.plugin = &Plugin{
		Plugin: plugins.NewBasePlugin(),
		name:   "test",
		logger: logger,
		client: suite.client,
	}
}
