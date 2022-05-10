module github.hpe.com/hpe/hpc-rabsw-nnf-sos

go 1.16

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v1.2.3
	github.com/google/uuid v1.3.0
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.19.0
	github.hpe.com/hpe/hpc-dpm-dws-operator v0.0.0-20220510143827-da96756d3b5a
	github.hpe.com/hpe/hpc-rabsw-lustre-fs-operator v0.0.0-20220509152545-29ee510a0c4f
	github.hpe.com/hpe/hpc-rabsw-nnf-ec v1.0.6-0.20220510143247-0c4cdb680dd4
	go.uber.org/zap v1.21.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	k8s.io/api v0.23.6
	k8s.io/apimachinery v0.23.6
	k8s.io/client-go v0.23.6
	sigs.k8s.io/controller-runtime v0.11.2
)
