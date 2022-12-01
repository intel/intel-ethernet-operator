// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package flow

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

// FlowServiceClient is the client API for FlowService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type FlowServiceClient interface {
	Validate(ctx context.Context, in *RequestFlowCreate, opts ...grpc.CallOption) (*ResponseFlow, error)
	Create(ctx context.Context, in *RequestFlowCreate, opts ...grpc.CallOption) (*ResponseFlowCreate, error)
	Destroy(ctx context.Context, in *RequestFlowofPort, opts ...grpc.CallOption) (*ResponseFlow, error)
	Query(ctx context.Context, in *RequestFlowofPort, opts ...grpc.CallOption) (*ResponseFlowQuery, error)
	List(ctx context.Context, in *RequestofPort, opts ...grpc.CallOption) (*ResponseFlowList, error)
	Flush(ctx context.Context, in *RequestofPort, opts ...grpc.CallOption) (*ResponseFlow, error)
	Isolate(ctx context.Context, in *RequestIsolate, opts ...grpc.CallOption) (*ResponseFlow, error)
	ListPorts(ctx context.Context, in *RequestListPorts, opts ...grpc.CallOption) (*ResponsePortList, error)
}

type flowServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewFlowServiceClient(cc grpc.ClientConnInterface) FlowServiceClient {
	return &flowServiceClient{cc}
}

func (c *flowServiceClient) Validate(ctx context.Context, in *RequestFlowCreate, opts ...grpc.CallOption) (*ResponseFlow, error) {
	out := new(ResponseFlow)
	err := c.cc.Invoke(ctx, "/flow.FlowService/Validate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowServiceClient) Create(ctx context.Context, in *RequestFlowCreate, opts ...grpc.CallOption) (*ResponseFlowCreate, error) {
	out := new(ResponseFlowCreate)
	err := c.cc.Invoke(ctx, "/flow.FlowService/Create", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowServiceClient) Destroy(ctx context.Context, in *RequestFlowofPort, opts ...grpc.CallOption) (*ResponseFlow, error) {
	out := new(ResponseFlow)
	err := c.cc.Invoke(ctx, "/flow.FlowService/Destroy", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowServiceClient) Query(ctx context.Context, in *RequestFlowofPort, opts ...grpc.CallOption) (*ResponseFlowQuery, error) {
	out := new(ResponseFlowQuery)
	err := c.cc.Invoke(ctx, "/flow.FlowService/Query", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowServiceClient) List(ctx context.Context, in *RequestofPort, opts ...grpc.CallOption) (*ResponseFlowList, error) {
	out := new(ResponseFlowList)
	err := c.cc.Invoke(ctx, "/flow.FlowService/List", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowServiceClient) Flush(ctx context.Context, in *RequestofPort, opts ...grpc.CallOption) (*ResponseFlow, error) {
	out := new(ResponseFlow)
	err := c.cc.Invoke(ctx, "/flow.FlowService/Flush", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowServiceClient) Isolate(ctx context.Context, in *RequestIsolate, opts ...grpc.CallOption) (*ResponseFlow, error) {
	out := new(ResponseFlow)
	err := c.cc.Invoke(ctx, "/flow.FlowService/Isolate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *flowServiceClient) ListPorts(ctx context.Context, in *RequestListPorts, opts ...grpc.CallOption) (*ResponsePortList, error) {
	out := new(ResponsePortList)
	err := c.cc.Invoke(ctx, "/flow.FlowService/ListPorts", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FlowServiceServer is the server API for FlowService service.
// All implementations must embed UnimplementedFlowServiceServer
// for forward compatibility
type FlowServiceServer interface {
	Validate(context.Context, *RequestFlowCreate) (*ResponseFlow, error)
	Create(context.Context, *RequestFlowCreate) (*ResponseFlowCreate, error)
	Destroy(context.Context, *RequestFlowofPort) (*ResponseFlow, error)
	Query(context.Context, *RequestFlowofPort) (*ResponseFlowQuery, error)
	List(context.Context, *RequestofPort) (*ResponseFlowList, error)
	Flush(context.Context, *RequestofPort) (*ResponseFlow, error)
	Isolate(context.Context, *RequestIsolate) (*ResponseFlow, error)
	ListPorts(context.Context, *RequestListPorts) (*ResponsePortList, error)
	mustEmbedUnimplementedFlowServiceServer()
}

// UnimplementedFlowServiceServer must be embedded to have forward compatible implementations.
type UnimplementedFlowServiceServer struct {
}

func (UnimplementedFlowServiceServer) Validate(context.Context, *RequestFlowCreate) (*ResponseFlow, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Validate not implemented")
}
func (UnimplementedFlowServiceServer) Create(context.Context, *RequestFlowCreate) (*ResponseFlowCreate, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedFlowServiceServer) Destroy(context.Context, *RequestFlowofPort) (*ResponseFlow, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Destroy not implemented")
}
func (UnimplementedFlowServiceServer) Query(context.Context, *RequestFlowofPort) (*ResponseFlowQuery, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Query not implemented")
}
func (UnimplementedFlowServiceServer) List(context.Context, *RequestofPort) (*ResponseFlowList, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedFlowServiceServer) Flush(context.Context, *RequestofPort) (*ResponseFlow, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Flush not implemented")
}
func (UnimplementedFlowServiceServer) Isolate(context.Context, *RequestIsolate) (*ResponseFlow, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Isolate not implemented")
}
func (UnimplementedFlowServiceServer) ListPorts(context.Context, *RequestListPorts) (*ResponsePortList, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListPorts not implemented")
}
func (UnimplementedFlowServiceServer) mustEmbedUnimplementedFlowServiceServer() {}

// UnsafeFlowServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to FlowServiceServer will
// result in compilation errors.
type UnsafeFlowServiceServer interface {
	mustEmbedUnimplementedFlowServiceServer()
}

func RegisterFlowServiceServer(s grpc.ServiceRegistrar, srv FlowServiceServer) {
	s.RegisterService(&FlowService_ServiceDesc, srv)
}

func _FlowService_Validate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestFlowCreate)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlowServiceServer).Validate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/flow.FlowService/Validate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlowServiceServer).Validate(ctx, req.(*RequestFlowCreate))
	}
	return interceptor(ctx, in, info, handler)
}

func _FlowService_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestFlowCreate)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlowServiceServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/flow.FlowService/Create",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlowServiceServer).Create(ctx, req.(*RequestFlowCreate))
	}
	return interceptor(ctx, in, info, handler)
}

func _FlowService_Destroy_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestFlowofPort)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlowServiceServer).Destroy(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/flow.FlowService/Destroy",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlowServiceServer).Destroy(ctx, req.(*RequestFlowofPort))
	}
	return interceptor(ctx, in, info, handler)
}

func _FlowService_Query_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestFlowofPort)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlowServiceServer).Query(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/flow.FlowService/Query",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlowServiceServer).Query(ctx, req.(*RequestFlowofPort))
	}
	return interceptor(ctx, in, info, handler)
}

func _FlowService_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestofPort)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlowServiceServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/flow.FlowService/List",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlowServiceServer).List(ctx, req.(*RequestofPort))
	}
	return interceptor(ctx, in, info, handler)
}

func _FlowService_Flush_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestofPort)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlowServiceServer).Flush(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/flow.FlowService/Flush",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlowServiceServer).Flush(ctx, req.(*RequestofPort))
	}
	return interceptor(ctx, in, info, handler)
}

func _FlowService_Isolate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestIsolate)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlowServiceServer).Isolate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/flow.FlowService/Isolate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlowServiceServer).Isolate(ctx, req.(*RequestIsolate))
	}
	return interceptor(ctx, in, info, handler)
}

func _FlowService_ListPorts_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RequestListPorts)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(FlowServiceServer).ListPorts(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/flow.FlowService/ListPorts",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(FlowServiceServer).ListPorts(ctx, req.(*RequestListPorts))
	}
	return interceptor(ctx, in, info, handler)
}

// FlowService_ServiceDesc is the grpc.ServiceDesc for FlowService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FlowService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "flow.FlowService",
	HandlerType: (*FlowServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Validate",
			Handler:    _FlowService_Validate_Handler,
		},
		{
			MethodName: "Create",
			Handler:    _FlowService_Create_Handler,
		},
		{
			MethodName: "Destroy",
			Handler:    _FlowService_Destroy_Handler,
		},
		{
			MethodName: "Query",
			Handler:    _FlowService_Query_Handler,
		},
		{
			MethodName: "List",
			Handler:    _FlowService_List_Handler,
		},
		{
			MethodName: "Flush",
			Handler:    _FlowService_Flush_Handler,
		},
		{
			MethodName: "Isolate",
			Handler:    _FlowService_Isolate_Handler,
		},
		{
			MethodName: "ListPorts",
			Handler:    _FlowService_ListPorts_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "pkg/flowconfig/rpc/v1/flow/flow.proto",
}
