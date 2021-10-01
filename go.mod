module github.com/otcshare/intel-ethernet-operator

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/golang/protobuf v1.4.3
	github.com/jaypipes/ghw v0.8.0
	github.com/jaypipes/pcidb v0.6.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/stretchr/testify v1.6.1
	go.uber.org/multierr v1.5.0
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	google.golang.org/grpc v1.27.1
	google.golang.org/protobuf v1.25.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	k8s.io/klog v1.0.0
	sigs.k8s.io/controller-runtime v0.8.3
)
