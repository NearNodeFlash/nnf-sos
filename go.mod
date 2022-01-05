module github.hpe.com/hpe/hpc-rabsw-nnf-sos

go 1.16

// Vendor dws-operator from the local submodule.
replace github.hpe.com/hpe/hpc-dpm-dws-operator => ./.dws-operator

replace github.hpe.com/hpe/hpc-rabsw-nnf-dm => ./.nnf-dm

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/google/uuid v1.3.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.hpe.com/hpe/hpc-dpm-dws-operator v0.0.0-20211213160018-2b73fb1030f9
	github.hpe.com/hpe/hpc-rabsw-nnf-dm v0.0.0-20211223144003-13f318944aeb
	github.hpe.com/hpe/hpc-rabsw-nnf-ec v1.0.6-0.20211213185652-ece897e5fe46
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/controller-runtime v0.10.2
)
