/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package flowconfig

import (
	"context"
	"fmt"
	"net"
	"time"

	flowapi "github.com/otcshare/intel-ethernet-operator/pkg/flowconfig/rpc/v1/flow"
	"google.golang.org/grpc"
)

var dcfEndpoint = "/var/run/dcf/dcf_tool.sock"

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
	conn, err := grpc.Dial(dcfEndpoint, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to DCF grpc endpoint: %s", err)
	}
	return conn, nil
}
