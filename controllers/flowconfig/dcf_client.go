// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2020-2022 Intel Corporation

package flowconfig

import (
	"context"
	"fmt"
	"time"

	flowapi "github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	"google.golang.org/grpc"
)

const (
	grpcUrl = "localhost:50051"
)

type dcfClient struct{}

// GetDCFClient return an instance of DCF FlowServiceClient
func GetDCFClient() flowapi.FlowServiceClient {
	return &dcfClient{}
}

func (dc *dcfClient) Validate(ctx context.Context, in *flowapi.RequestFlowCreate, opts ...grpc.CallOption) (*flowapi.ResponseFlow, error) {
	conn, err := getDCFFlowClientConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := flowapi.NewFlowServiceClient(conn)
	return client.Validate(ctx, in, opts...)
}

func (dc *dcfClient) Create(ctx context.Context, in *flowapi.RequestFlowCreate, opts ...grpc.CallOption) (*flowapi.ResponseFlowCreate, error) {
	conn, err := getDCFFlowClientConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := flowapi.NewFlowServiceClient(conn)
	return client.Create(ctx, in, opts...)
}

func (dc *dcfClient) Destroy(ctx context.Context, in *flowapi.RequestFlowofPort, opts ...grpc.CallOption) (*flowapi.ResponseFlow, error) {
	conn, err := getDCFFlowClientConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := flowapi.NewFlowServiceClient(conn)
	return client.Destroy(ctx, in, opts...)
}

func (dc *dcfClient) Query(ctx context.Context, in *flowapi.RequestFlowofPort, opts ...grpc.CallOption) (*flowapi.ResponseFlowQuery, error) {
	conn, err := getDCFFlowClientConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := flowapi.NewFlowServiceClient(conn)
	return client.Query(ctx, in, opts...)
}

func (dc *dcfClient) List(ctx context.Context, in *flowapi.RequestofPort, opts ...grpc.CallOption) (*flowapi.ResponseFlowList, error) {
	conn, err := getDCFFlowClientConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := flowapi.NewFlowServiceClient(conn)
	return client.List(ctx, in, opts...)
}

func (dc *dcfClient) Flush(ctx context.Context, in *flowapi.RequestofPort, opts ...grpc.CallOption) (*flowapi.ResponseFlow, error) {
	conn, err := getDCFFlowClientConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := flowapi.NewFlowServiceClient(conn)
	return client.Flush(ctx, in, opts...)
}

func (dc *dcfClient) Isolate(ctx context.Context, in *flowapi.RequestIsolate, opts ...grpc.CallOption) (*flowapi.ResponseFlow, error) {
	conn, err := getDCFFlowClientConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := flowapi.NewFlowServiceClient(conn)
	return client.Isolate(ctx, in, opts...)
}

func (dc *dcfClient) ListPorts(ctx context.Context, in *flowapi.RequestListPorts, opts ...grpc.CallOption) (*flowapi.ResponsePortList, error) {
	conn, err := getDCFFlowClientConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	client := flowapi.NewFlowServiceClient(conn)
	return client.ListPorts(ctx, in, opts...)
}

func getDCFFlowClientConn() (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, grpcUrl, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("unable to connect to DCF grpc endpoint: %s", err)
	}
	return conn, nil
}
