package testserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nats-io/nats.go"

	_ "github.com/formancehq/payments/internal/connectors/plugins/public"
	formance "github.com/formancehq/payments/pkg/client"

	"github.com/formancehq/go-libs/v3/otlp"
	"github.com/formancehq/go-libs/v3/otlp/otlpmetrics"
	"github.com/google/uuid"
	"github.com/uptrace/bun"

	"github.com/formancehq/go-libs/v3/bun/bunconnect"
	"github.com/formancehq/go-libs/v3/httpclient"
	"github.com/formancehq/go-libs/v3/httpserver"
	"github.com/formancehq/go-libs/v3/logging"
	"github.com/formancehq/go-libs/v3/service"
	"github.com/formancehq/payments/cmd"
	"github.com/stretchr/testify/require"
)

var defaultHttpClientTimeout = 3 * time.Second

type T interface {
	require.TestingT
	Cleanup(func())
	Helper()
	Logf(format string, args ...any)
}

type OTLPConfig struct {
	BaseConfig otlp.Config
	Metrics    *otlpmetrics.ModuleConfig
}

type Configuration struct {
	Stack                 string
	PostgresConfiguration bunconnect.ConnectionOptions
	TemporalNamespace     string
	TemporalAddress       string
	NatsURL               string
	ConfigEncryptionKey   string
	HttpClientTimeout     time.Duration
	Output                io.Writer
	Debug                 bool
	OTLPConfig            *OTLPConfig
}

type Logger interface {
	Logf(fmt string, args ...any)
}

type Server struct {
	id            string
	configuration Configuration
	logger        Logger
	worker        *Worker
	httpClient    *Client
	sdk           *formance.Formance
	cancel        func()
	ctx           context.Context
	errorChan     chan error
}

//nolint:govet
func (s *Server) Start() error {
	rootCmd := cmd.NewRootCommand()
	args := Flags("serve", s.id, s.configuration)

	s.logger.Logf("Starting application with flags: %s", strings.Join(args, " "))
	rootCmd.SetArgs(args)
	rootCmd.SilenceErrors = true
	output := s.configuration.Output
	if output == nil {
		output = io.Discard
	}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)

	ctx := logging.TestingContext()
	ctx = service.ContextWithLifecycle(ctx)
	ctx = httpserver.ContextWithServerInfo(ctx)
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		s.errorChan <- rootCmd.ExecuteContext(ctx)
	}()

	select {
	case <-ctx.Done():
		return errors.New("unexpected context cancel before server ready")
	case <-service.Ready(ctx):
	case err := <-s.errorChan:
		cancel()
		if err != nil {
			return err
		}

		return errors.New("unexpected service stop")
	}

	s.ctx, s.cancel = ctx, cancel

	var transport http.RoundTripper = &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
	}
	if s.configuration.Debug {
		transport = httpclient.NewDebugHTTPTransport(transport)
	}

	httpClient, err := NewClient(httpserver.URL(s.ctx), s.configuration.HttpClientTimeout, transport)
	if err != nil {
		return err
	}
	s.httpClient = httpClient

	s.sdk, err = NewStackClient(httpserver.URL(s.ctx), s.configuration.HttpClientTimeout, transport)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if s.cancel == nil {
		return nil
	}
	s.cancel()
	s.cancel = nil

	// Wait app to be marked as stopped
	select {
	case <-service.Stopped(s.ctx):
	case <-ctx.Done():
		return errors.New("service should have been stopped")
	}

	// Ensure the app has been properly shutdown
	select {
	case err := <-s.errorChan:
		return err
	case <-ctx.Done():
		return errors.New("service should have been stopped without error")
	}
}

func (s *Server) Client() *Client {
	return s.httpClient
}

func (s *Server) SDK() *formance.Formance {
	return s.sdk
}

func (s *Server) Restart(ctx context.Context) error {
	if err := s.Stop(ctx); err != nil {
		return err
	}
	if err := s.Start(); err != nil {
		return err
	}

	return nil
}

func (s *Server) Database() (*bun.DB, error) {
	db, err := bunconnect.OpenSQLDB(s.ctx, s.configuration.PostgresConfiguration)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (s *Server) Subscribe() (*nats.Subscription, chan *nats.Msg, error) {
	if s.configuration.NatsURL == "" {
		return nil, nil, errors.New("NATS URL must be set")
	}

	ret := make(chan *nats.Msg)
	conn, err := nats.Connect(s.configuration.NatsURL)
	if err != nil {
		return nil, nil, err
	}

	subscription, err := conn.Subscribe(s.id, func(msg *nats.Msg) {
		ret <- msg
	})
	if err != nil {
		return nil, nil, err
	}

	return subscription, ret, nil
}

func (s *Server) URL() string {
	return httpserver.URL(s.ctx)
}

func New(t T, configuration Configuration) *Server {
	t.Helper()

	if configuration.HttpClientTimeout == 0 {
		configuration.HttpClientTimeout = defaultHttpClientTimeout
	}

	serverID := uuid.NewString()[:8]
	worker := NewWorker(t, configuration, serverID)

	srv := &Server{
		id:            serverID,
		logger:        t,
		configuration: configuration,
		worker:        worker,
		errorChan:     make(chan error, 1),
	}
	t.Logf("Start testing server")
	require.NoError(t, srv.Start())
	t.Cleanup(func() {
		t.Logf("Stop testing server")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		require.NoError(t, srv.Stop(ctx))
	})

	return srv
}
