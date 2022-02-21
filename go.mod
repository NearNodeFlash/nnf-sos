module github.hpe.com/hpe/hpc-rabsw-nnf-sos

go 1.16

// Vendor dws-operator from the local submodule.
replace github.hpe.com/hpe/hpc-dpm-dws-operator => ./.dws-operator

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v1.2.2
	github.com/google/uuid v1.3.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.18.1
	github.hpe.com/hpe/hpc-dpm-dws-operator v0.0.0-20220217151406-daca164ea515
	github.hpe.com/hpe/hpc-rabsw-lustre-fs-operator v0.0.0-20220214205302-00d98e6cd7b7
	github.hpe.com/hpe/hpc-rabsw-nnf-ec v1.0.6-0.20220218184916-7363e5812eb8
	k8s.io/api v0.23.4
	k8s.io/apimachinery v0.23.4
	k8s.io/client-go v0.23.4
	sigs.k8s.io/controller-runtime v0.11.1
)
