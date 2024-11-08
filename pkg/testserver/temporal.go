package testserver

import (
	"context"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/server/temporaltest"
)

type TemporalT interface {
	require.TestingT
	Cleanup(func())
}

type TemporalServer struct {
	*temporaltest.TestServer `json:"-"`
}

func CreateTemporalServer(t TemporalT) *TemporalServer {
	srv := temporaltest.NewServer()
	return &TemporalServer{TestServer: srv}
}

func (s *TemporalServer) Client(ctx context.Context) client.Client {
	return s.GetDefaultClient()
}

func (s *TemporalServer) Address() string {
	return s.GetFrontendHostPort()
}
