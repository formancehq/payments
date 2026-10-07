package engine_test

import (
	"context"
	"net"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
)

// fakeTemporal is an in-process stand-in for the Temporal frontend, just
// enough for worker.Start to succeed and its pollers to idle. Everything else
// answers Unimplemented, which the SDK tolerates for the start-up handshake
// (GetSystemInfo, DescribeNamespace).
type fakeTemporal struct {
	workflowservice.UnimplementedWorkflowServiceServer
}

// Polls long-poll like the real server: hold the request until the worker
// gives up on it, then return an empty task.
func (fakeTemporal) PollWorkflowTaskQueue(ctx context.Context, _ *workflowservice.PollWorkflowTaskQueueRequest) (*workflowservice.PollWorkflowTaskQueueResponse, error) {
	<-ctx.Done()
	return &workflowservice.PollWorkflowTaskQueueResponse{}, nil
}

func (fakeTemporal) PollActivityTaskQueue(ctx context.Context, _ *workflowservice.PollActivityTaskQueueRequest) (*workflowservice.PollActivityTaskQueueResponse, error) {
	<-ctx.Done()
	return &workflowservice.PollActivityTaskQueueResponse{}, nil
}

func (fakeTemporal) ShutdownWorker(context.Context, *workflowservice.ShutdownWorkerRequest) (*workflowservice.ShutdownWorkerResponse, error) {
	return &workflowservice.ShutdownWorkerResponse{}, nil
}

// startFakeTemporal serves fakeTemporal on a loopback port for the current
// spec and returns its address, to be used as client.Options.HostPort.
func startFakeTemporal() string {
	return serveFakeTemporal()
}

// startGatedFakeTemporal is startFakeTemporal with every unary call held until
// release is called, so a test can act while worker.Start is blocked in its
// start-up handshake. The returned channel is closed once the first call
// arrives, i.e. once Start is known to be in flight. release is idempotent and
// also runs on cleanup, so a failing spec does not leave Start hanging.
func startGatedFakeTemporal() (addr string, entered <-chan struct{}, release func()) {
	gate := make(chan struct{})
	in := make(chan struct{})
	var enterOnce, releaseOnce sync.Once
	release = func() { releaseOnce.Do(func() { close(gate) }) }
	DeferCleanup(release)

	addr = serveFakeTemporal(grpc.UnaryInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		enterOnce.Do(func() { close(in) })
		select {
		case <-gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return handler(ctx, req)
	}))
	return addr, in, release
}

func serveFakeTemporal(opts ...grpc.ServerOption) string {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).To(BeNil())
	srv := grpc.NewServer(opts...)
	workflowservice.RegisterWorkflowServiceServer(srv, fakeTemporal{})
	go func() { _ = srv.Serve(lis) }()
	DeferCleanup(srv.Stop)
	return lis.Addr().String()
}
