module stash.us.cray.com/RABSW/nnf-sos

go 1.16

// Vendor dws-operator from the local submodule.
replace stash.us.cray.com/dpm/dws-operator => ./.dws-operator

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	k8s.io/api v0.22.2
	k8s.io/apimachinery v0.22.2
	k8s.io/client-go v0.22.2
	sigs.k8s.io/controller-runtime v0.10.2
	stash.us.cray.com/dpm/dws-operator v0.0.0-20211015191327-a7b6dd87f0c5
	stash.us.cray.com/rabsw/nnf-ec v1.0.6-0.20211015143100-4850b352b44e
)
