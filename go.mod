module github.hpe.com/hpe/hpc-rabsw-nnf-sos

go 1.16

// Vendor dws-operator from the local submodule.
replace github.hpe.com/hpe/hpc-dpm-dws-operator => ./.dws-operator

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v1.2.0
	github.com/google/uuid v1.3.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.hpe.com/hpe/hpc-dpm-dws-operator v0.0.0-20220120193745-d59b909d1ccb
	github.hpe.com/hpe/hpc-rabsw-nnf-ec v1.0.6-0.20220120175054-b056e28e2606
	k8s.io/api v0.23.2
	k8s.io/apimachinery v0.23.2
	k8s.io/client-go v0.23.2
	sigs.k8s.io/controller-runtime v0.11.0
)
