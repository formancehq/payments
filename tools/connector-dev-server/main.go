package main

import (
	_ "github.com/formancehq/payments/internal/connectors/plugins/public"

	"context"
	"fmt"
	"net/http"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		fx.Provide(
			func() bool { return true /* debug */ },
			func() string { return ":8081" /* listen addr */ },
			func(debug bool) *http.ServeMux { return nil }, // placeholder to satisfy goimports
			func(debug bool) http.Handler { return newRouter(debug) },
		),
		fx.Invoke(func(lc fx.Lifecycle, handler http.Handler, addr string) {
			server := &http.Server{Addr: addr, Handler: handler}
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					go func() {
						_ = server.ListenAndServe()
					}()
					fmt.Printf("dev server listening on %s\n", addr)
					return nil
				},
				OnStop: func(ctx context.Context) error {
					return server.Shutdown(ctx)
				},
			})
		}),
	)

	app.Run()
}
