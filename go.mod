module github.com/otcshare/intel-ethernet-operator

go 1.16

require (
	github.com/go-logr/logr v1.2.0
	github.com/golang/protobuf v1.5.2
	github.com/google/gofuzz v1.2.0
	github.com/jaypipes/ghw v0.8.0
	github.com/jaypipes/pcidb v0.6.0
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.1.1-0.20201119153432-9d213757d22d
	github.com/k8snetworkplumbingwg/sriov-network-device-plugin v0.0.0-20211118081735-7488066fa720
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	github.com/openshift/api v0.0.0-20220218143101-271bd7e1834c
	github.com/stretchr/testify v1.7.0
	google.golang.org/grpc v1.43.0
	google.golang.org/protobuf v1.27.1
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v0.23.4
	k8s.io/kubectl v0.23.4
	sigs.k8s.io/controller-runtime v0.11.1
	sigs.k8s.io/yaml v1.3.0
)

require google.golang.org/genproto v0.0.0-20211008145708-270636b82663 // indirect
