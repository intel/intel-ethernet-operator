module github.com/otcshare/intel-ethernet-operator

go 1.16

require (
	github.com/go-logr/logr v1.2.3
	github.com/golang/protobuf v1.5.2
	github.com/google/gofuzz v1.2.0
	github.com/jaypipes/ghw v0.9.0
	github.com/jaypipes/pcidb v1.0.0
	github.com/k8snetworkplumbingwg/network-attachment-definition-client v1.1.1-0.20201119153432-9d213757d22d
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.22.1
	github.com/openshift/api v0.0.0-20220218143101-271bd7e1834c
	github.com/stretchr/testify v1.8.1
	google.golang.org/grpc v1.50.1
	google.golang.org/protobuf v1.28.1
	k8s.io/api v0.25.3
	k8s.io/apimachinery v0.25.3
	k8s.io/client-go v0.25.3
	k8s.io/kubectl v0.25.3
	sigs.k8s.io/controller-runtime v0.13.0
	sigs.k8s.io/yaml v1.3.0
)

require google.golang.org/genproto v0.0.0-20221024183307-1bc688fe9f3e // indirect
