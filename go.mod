module github.com/otcshare/intel-ethernet-operator

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/golang/protobuf v1.5.2
	github.com/jaypipes/ghw v0.8.0
	github.com/jaypipes/pcidb v0.6.0
	github.com/k8snetworkplumbingwg/sriov-network-device-plugin v0.0.0-20210505093143-e112960e14df
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.2.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/stretchr/testify v1.7.0
	go.uber.org/multierr v1.5.0
	golang.org/x/net v0.0.0-20211008194852-3b03d305991f
	golang.org/x/sys v0.0.0-20211007075335-d3039528d8ac // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20211008145708-270636b82663 // indirect
	google.golang.org/grpc v1.41.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.8.3
)
