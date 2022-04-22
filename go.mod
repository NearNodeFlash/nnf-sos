module github.hpe.com/hpe/hpc-rabsw-nnf-sos

go 1.16

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v1.2.3
	github.com/google/uuid v1.3.0
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.hpe.com/hpe/hpc-dpm-dws-operator v0.0.0-20220422161013-5b41c4cc5d48
	github.hpe.com/hpe/hpc-rabsw-lustre-fs-operator v0.0.0-20220214205302-00d98e6cd7b7
	github.hpe.com/hpe/hpc-rabsw-nnf-ec v1.0.6-0.20220421203431-de8207e808b5
	go.uber.org/zap v1.21.0
	k8s.io/api v0.23.6
	k8s.io/apimachinery v0.23.6
	k8s.io/client-go v0.23.6
	sigs.k8s.io/controller-runtime v0.11.2
)
