module github.com/NearNodeFlash/nnf-sos

go 1.16

replace github.com/NearNodeFlash/nnf-dm => ../nnf-dm

require (
	github.com/HewlettPackard/dws v0.0.0-20220826164140-9f0237adcfe7
	github.com/NearNodeFlash/lustre-fs-operator v0.0.0-20220727174249-9b7004c2cb38
	github.com/NearNodeFlash/nnf-dm v0.0.0-20220830153611-0be8f906cb59
	github.com/NearNodeFlash/nnf-ec v0.0.0-20220829193624-66025314aec0
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v1.2.3
	github.com/google/uuid v1.3.0
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.20.1
	github.com/prometheus/client_golang v1.12.1
	go.uber.org/zap v1.22.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	k8s.io/api v0.24.4
	k8s.io/apimachinery v0.24.4
	k8s.io/client-go v0.24.4
	k8s.io/mount-utils v0.24.4
	sigs.k8s.io/controller-runtime v0.12.3
)
