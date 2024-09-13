// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package services

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// PluginClient is the client API for Plugin service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type PluginClient interface {
	Install(ctx context.Context, in *InstallRequest, opts ...grpc.CallOption) (*InstallResponse, error)
	Uninstall(ctx context.Context, in *UninstallRequest, opts ...grpc.CallOption) (*UninstallResponse, error)
	FetchNextOthers(ctx context.Context, in *FetchNextOthersRequest, opts ...grpc.CallOption) (*FetchNextOthersResponse, error)
	FetchNextPayments(ctx context.Context, in *FetchNextPaymentsRequest, opts ...grpc.CallOption) (*FetchNextPaymentsResponse, error)
	FetchNextAccounts(ctx context.Context, in *FetchNextAccountsRequest, opts ...grpc.CallOption) (*FetchNextAccountsResponse, error)
	FetchNextExternalAccounts(ctx context.Context, in *FetchNextExternalAccountsRequest, opts ...grpc.CallOption) (*FetchNextExternalAccountsResponse, error)
	FetchNextBalances(ctx context.Context, in *FetchNextBalancesRequest, opts ...grpc.CallOption) (*FetchNextBalancesResponse, error)
	CreateBankAccount(ctx context.Context, in *CreateBankAccountRequest, opts ...grpc.CallOption) (*CreateBankAccountResponse, error)
	CreateWebhooks(ctx context.Context, in *CreateWebhooksRequest, opts ...grpc.CallOption) (*CreateWebhooksResponse, error)
	TranslateWebhook(ctx context.Context, in *TranslateWebhookRequest, opts ...grpc.CallOption) (*TranslateWebhookResponse, error)
}

type pluginClient struct {
	cc grpc.ClientConnInterface
}

func NewPluginClient(cc grpc.ClientConnInterface) PluginClient {
	return &pluginClient{cc}
}

func (c *pluginClient) Install(ctx context.Context, in *InstallRequest, opts ...grpc.CallOption) (*InstallResponse, error) {
	out := new(InstallResponse)
	err := c.cc.Invoke(ctx, "/formance.payments.grpc.services.Plugin/Install", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) Uninstall(ctx context.Context, in *UninstallRequest, opts ...grpc.CallOption) (*UninstallResponse, error) {
	out := new(UninstallResponse)
	err := c.cc.Invoke(ctx, "/formance.payments.grpc.services.Plugin/Uninstall", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) FetchNextOthers(ctx context.Context, in *FetchNextOthersRequest, opts ...grpc.CallOption) (*FetchNextOthersResponse, error) {
	out := new(FetchNextOthersResponse)
	err := c.cc.Invoke(ctx, "/formance.payments.grpc.services.Plugin/FetchNextOthers", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) FetchNextPayments(ctx context.Context, in *FetchNextPaymentsRequest, opts ...grpc.CallOption) (*FetchNextPaymentsResponse, error) {
	out := new(FetchNextPaymentsResponse)
	err := c.cc.Invoke(ctx, "/formance.payments.grpc.services.Plugin/FetchNextPayments", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) FetchNextAccounts(ctx context.Context, in *FetchNextAccountsRequest, opts ...grpc.CallOption) (*FetchNextAccountsResponse, error) {
	out := new(FetchNextAccountsResponse)
	err := c.cc.Invoke(ctx, "/formance.payments.grpc.services.Plugin/FetchNextAccounts", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) FetchNextExternalAccounts(ctx context.Context, in *FetchNextExternalAccountsRequest, opts ...grpc.CallOption) (*FetchNextExternalAccountsResponse, error) {
	out := new(FetchNextExternalAccountsResponse)
	err := c.cc.Invoke(ctx, "/formance.payments.grpc.services.Plugin/FetchNextExternalAccounts", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) FetchNextBalances(ctx context.Context, in *FetchNextBalancesRequest, opts ...grpc.CallOption) (*FetchNextBalancesResponse, error) {
	out := new(FetchNextBalancesResponse)
	err := c.cc.Invoke(ctx, "/formance.payments.grpc.services.Plugin/FetchNextBalances", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) CreateBankAccount(ctx context.Context, in *CreateBankAccountRequest, opts ...grpc.CallOption) (*CreateBankAccountResponse, error) {
	out := new(CreateBankAccountResponse)
	err := c.cc.Invoke(ctx, "/formance.payments.grpc.services.Plugin/CreateBankAccount", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) CreateWebhooks(ctx context.Context, in *CreateWebhooksRequest, opts ...grpc.CallOption) (*CreateWebhooksResponse, error) {
	out := new(CreateWebhooksResponse)
	err := c.cc.Invoke(ctx, "/formance.payments.grpc.services.Plugin/CreateWebhooks", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *pluginClient) TranslateWebhook(ctx context.Context, in *TranslateWebhookRequest, opts ...grpc.CallOption) (*TranslateWebhookResponse, error) {
	out := new(TranslateWebhookResponse)
	err := c.cc.Invoke(ctx, "/formance.payments.grpc.services.Plugin/TranslateWebhook", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// PluginServer is the server API for Plugin service.
// All implementations must embed UnimplementedPluginServer
// for forward compatibility
type PluginServer interface {
	Install(context.Context, *InstallRequest) (*InstallResponse, error)
	Uninstall(context.Context, *UninstallRequest) (*UninstallResponse, error)
	FetchNextOthers(context.Context, *FetchNextOthersRequest) (*FetchNextOthersResponse, error)
	FetchNextPayments(context.Context, *FetchNextPaymentsRequest) (*FetchNextPaymentsResponse, error)
	FetchNextAccounts(context.Context, *FetchNextAccountsRequest) (*FetchNextAccountsResponse, error)
	FetchNextExternalAccounts(context.Context, *FetchNextExternalAccountsRequest) (*FetchNextExternalAccountsResponse, error)
	FetchNextBalances(context.Context, *FetchNextBalancesRequest) (*FetchNextBalancesResponse, error)
	CreateBankAccount(context.Context, *CreateBankAccountRequest) (*CreateBankAccountResponse, error)
	CreateWebhooks(context.Context, *CreateWebhooksRequest) (*CreateWebhooksResponse, error)
	TranslateWebhook(context.Context, *TranslateWebhookRequest) (*TranslateWebhookResponse, error)
	mustEmbedUnimplementedPluginServer()
}

// UnimplementedPluginServer must be embedded to have forward compatible implementations.
type UnimplementedPluginServer struct {
}

func (UnimplementedPluginServer) Install(context.Context, *InstallRequest) (*InstallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Install not implemented")
}
func (UnimplementedPluginServer) Uninstall(context.Context, *UninstallRequest) (*UninstallResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Uninstall not implemented")
}
func (UnimplementedPluginServer) FetchNextOthers(context.Context, *FetchNextOthersRequest) (*FetchNextOthersResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchNextOthers not implemented")
}
func (UnimplementedPluginServer) FetchNextPayments(context.Context, *FetchNextPaymentsRequest) (*FetchNextPaymentsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchNextPayments not implemented")
}
func (UnimplementedPluginServer) FetchNextAccounts(context.Context, *FetchNextAccountsRequest) (*FetchNextAccountsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchNextAccounts not implemented")
}
func (UnimplementedPluginServer) FetchNextExternalAccounts(context.Context, *FetchNextExternalAccountsRequest) (*FetchNextExternalAccountsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchNextExternalAccounts not implemented")
}
func (UnimplementedPluginServer) FetchNextBalances(context.Context, *FetchNextBalancesRequest) (*FetchNextBalancesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FetchNextBalances not implemented")
}
func (UnimplementedPluginServer) CreateBankAccount(context.Context, *CreateBankAccountRequest) (*CreateBankAccountResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateBankAccount not implemented")
}
func (UnimplementedPluginServer) CreateWebhooks(context.Context, *CreateWebhooksRequest) (*CreateWebhooksResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateWebhooks not implemented")
}
func (UnimplementedPluginServer) TranslateWebhook(context.Context, *TranslateWebhookRequest) (*TranslateWebhookResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TranslateWebhook not implemented")
}
func (UnimplementedPluginServer) mustEmbedUnimplementedPluginServer() {}

// UnsafePluginServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to PluginServer will
// result in compilation errors.
type UnsafePluginServer interface {
	mustEmbedUnimplementedPluginServer()
}

func RegisterPluginServer(s grpc.ServiceRegistrar, srv PluginServer) {
	s.RegisterService(&Plugin_ServiceDesc, srv)
}

func _Plugin_Install_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(InstallRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).Install(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/formance.payments.grpc.services.Plugin/Install",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).Install(ctx, req.(*InstallRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_Uninstall_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UninstallRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).Uninstall(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/formance.payments.grpc.services.Plugin/Uninstall",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).Uninstall(ctx, req.(*UninstallRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_FetchNextOthers_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FetchNextOthersRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).FetchNextOthers(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/formance.payments.grpc.services.Plugin/FetchNextOthers",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).FetchNextOthers(ctx, req.(*FetchNextOthersRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_FetchNextPayments_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FetchNextPaymentsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).FetchNextPayments(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/formance.payments.grpc.services.Plugin/FetchNextPayments",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).FetchNextPayments(ctx, req.(*FetchNextPaymentsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_FetchNextAccounts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FetchNextAccountsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).FetchNextAccounts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/formance.payments.grpc.services.Plugin/FetchNextAccounts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).FetchNextAccounts(ctx, req.(*FetchNextAccountsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_FetchNextExternalAccounts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FetchNextExternalAccountsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).FetchNextExternalAccounts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/formance.payments.grpc.services.Plugin/FetchNextExternalAccounts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).FetchNextExternalAccounts(ctx, req.(*FetchNextExternalAccountsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_FetchNextBalances_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FetchNextBalancesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).FetchNextBalances(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/formance.payments.grpc.services.Plugin/FetchNextBalances",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).FetchNextBalances(ctx, req.(*FetchNextBalancesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_CreateBankAccount_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateBankAccountRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).CreateBankAccount(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/formance.payments.grpc.services.Plugin/CreateBankAccount",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).CreateBankAccount(ctx, req.(*CreateBankAccountRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_CreateWebhooks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateWebhooksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).CreateWebhooks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/formance.payments.grpc.services.Plugin/CreateWebhooks",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).CreateWebhooks(ctx, req.(*CreateWebhooksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Plugin_TranslateWebhook_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TranslateWebhookRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PluginServer).TranslateWebhook(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/formance.payments.grpc.services.Plugin/TranslateWebhook",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PluginServer).TranslateWebhook(ctx, req.(*TranslateWebhookRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Plugin_ServiceDesc is the grpc.ServiceDesc for Plugin service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Plugin_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "formance.payments.grpc.services.Plugin",
	HandlerType: (*PluginServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Install",
			Handler:    _Plugin_Install_Handler,
		},
		{
			MethodName: "Uninstall",
			Handler:    _Plugin_Uninstall_Handler,
		},
		{
			MethodName: "FetchNextOthers",
			Handler:    _Plugin_FetchNextOthers_Handler,
		},
		{
			MethodName: "FetchNextPayments",
			Handler:    _Plugin_FetchNextPayments_Handler,
		},
		{
			MethodName: "FetchNextAccounts",
			Handler:    _Plugin_FetchNextAccounts_Handler,
		},
		{
			MethodName: "FetchNextExternalAccounts",
			Handler:    _Plugin_FetchNextExternalAccounts_Handler,
		},
		{
			MethodName: "FetchNextBalances",
			Handler:    _Plugin_FetchNextBalances_Handler,
		},
		{
			MethodName: "CreateBankAccount",
			Handler:    _Plugin_CreateBankAccount_Handler,
		},
		{
			MethodName: "CreateWebhooks",
			Handler:    _Plugin_CreateWebhooks_Handler,
		},
		{
			MethodName: "TranslateWebhook",
			Handler:    _Plugin_TranslateWebhook_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "services/plugin.proto",
}
