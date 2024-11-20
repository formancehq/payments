package plugins

import (
	"context"
	"sync"

	"github.com/formancehq/payments/internal/connectors/grpc"
	"github.com/formancehq/payments/internal/models"
	"github.com/hashicorp/go-plugin"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	ggrpc "google.golang.org/grpc"
)

type Server interface {
	Serve(wg *sync.WaitGroup, shutdowner fx.Shutdowner)
}

type server struct {
	plugin models.Plugin
}

func NewServer(lc fx.Lifecycle, shutdowner fx.Shutdowner, plg models.Plugin) Server {
	srv := &server{plugin: plg}
	wg := &sync.WaitGroup{}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			wg.Add(1)
			go srv.Serve(wg, shutdowner)
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// plugin.Serve is expected to block until the plugin.Client tells it to stop
			// this ensures the main plugin process doesn't exit before the plugin server shutsdown
			wg.Wait()
			return nil
		},
	})
	return srv
}

func (s *server) Serve(wg *sync.WaitGroup, shutdowner fx.Shutdowner) {
	defer func() {
		wg.Done()
		// when Serve has ended the server closed usually because the plugin.Client told it to
		// if the parent application (managed by fx) is still running we need to tell it the plugin is done
		shutdowner.Shutdown(fx.ExitCode(0))
	}()

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: grpc.Handshake(),
		Plugins: map[string]plugin.Plugin{
			"psp": &grpc.PSPGRPCPlugin{Impl: NewGRPCImplem(s.plugin)},
		},
		// server instrumented with OTel Contrib middleware
		GRPCServer: func(opts []ggrpc.ServerOption) *ggrpc.Server {
			opts = append(opts, ggrpc.StatsHandler(otelgrpc.NewServerHandler()))
			return ggrpc.NewServer(opts...)
		},
	})
}
